// SPDX-FileCopyrightText: 2025 RH America LLC <info@rhamerica.com>
// SPDX-License-Identifier: GPL-3.0-only

package luautils

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	lua "github.com/yuin/gopher-lua"
)

func TestRunLuaScript_ReturnsTable(t *testing.T) {
	luaContent := `return {env={foo="bar"}}`
	_, file := writeTempLuaFile(t, luaContent)
	env := map[string]string{"foo": "baz"} // should be replaced by lua script's 'bar'
	var called bool
	err := RunLuaScript(file, env, func(retval *lua.LTable) {
		called = true
		envTbl := retval.RawGetString("env")
		if tbl, ok := envTbl.(*lua.LTable); ok {
			val := tbl.RawGetString("foo")
			if val.String() != "bar" {
				t.Errorf("expected foo=bar, got %s", val.String())
			}
		} else {
			t.Errorf("env is not a table")
		}
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Errorf("callback was not called")
	}
}

func TestRunLuaScript_ReturnsNonTable(t *testing.T) {
	luaContent := `return "notatable"`
	_, file := writeTempLuaFile(t, luaContent)
	env := map[string]string{}
	err := RunLuaScript(file, env, func(retval *lua.LTable) {
		t.Error("callback should not be called for non-table return")
	})
	if err == nil {
		t.Fatal("expected error for non-table return, got nil")
	}
}

func TestRunLuaScript_LuaError(t *testing.T) {
	luaContent := `error("fail")`
	_, file := writeTempLuaFile(t, luaContent)
	env := map[string]string{}
	err := RunLuaScript(file, env, nil)
	if err == nil {
		t.Fatal("expected error from lua script, got nil")
	}
}

func TestRunLuaScript_NilCallback(t *testing.T) {
	luaContent := `return {env={foo="bar"}}`
	_, file := writeTempLuaFile(t, luaContent)
	env := map[string]string{}
	err := RunLuaScript(file, env, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunLuaScript_FunctionsAreAreCalled(t *testing.T) {
	dir := t.TempDir()
	env := map[string]string{
		"a":      "a",
		"b":      "b",
		"luadir": dir,
	}
	luaContent := `
	local a = os.getenv("a")
	local b = os.getenv("b")
	local dir = os.getenv("luadir")
	bz.mkdir(dir .. "/testdir")
	local f = io.open(dir .. "/testdir/testfile.txt", "w")
	f:write("hello")
	f:close()
	return {
		env = {
			a=a,
			b=b,
		},
	}
	`
	_, file := writeTempLuaFile(t, luaContent)
	err := RunLuaScript(file, env, func(retval *lua.LTable) {
		envTbl := retval.RawGetString("env")
		if tbl, ok := envTbl.(*lua.LTable); ok {
			valA := tbl.RawGetString("a")
			if valA.String() != "a" {
				t.Errorf("expected a=a, got %s", valA.String())
			}
			valB := tbl.RawGetString("b")
			if valB.String() != "b" {
				t.Errorf("expected b=b, got %s", valB.String())
			}
		} else {
			t.Errorf("env is not a table")
		}
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Check if the file was created by the Lua script
	createdFile := filepath.Join(dir, "testdir", "testfile.txt")
	if _, err := os.Stat(createdFile); os.IsNotExist(err) {
		t.Errorf("expected file %s to be created by Lua script, but it does not exist", createdFile)
	}
}

func TestRunLuaScript_BzDownloadFunction(t *testing.T) {
	DownloadContent := "Hello, World!"
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(DownloadContent))
	}))
	defer s.Close()

	dir := t.TempDir()
	env := map[string]string{
		"luadir":       dir,
		"download_url": s.URL + "/",
	}
	luaContent := `
	local dir = os.getenv("luadir")
	local url = os.getenv("download_url")
	local r = bz.download(url, dir .. "/example.txt")
	if not r.ok then
		error("download failed: " .. r.error)
	end
	return {
		env = {
			luadir=dir,
		},
	}
	`
	_, file := writeTempLuaFile(t, luaContent)
	err := RunLuaScript(file, env, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Check if the file was created by the Lua script
	downloadedFile := filepath.Join(dir, "example.txt")
	data, err := os.ReadFile(downloadedFile)
	if err != nil {
		t.Fatalf("expected file %s to be created by Lua script, but got error: %v", downloadedFile, err)
	}
	if string(data) != DownloadContent {
		t.Errorf("expected downloaded content to be '%s', got '%s'", DownloadContent, string(data))
	}
}

func TestRunLuaScript_BzDownloadFunctionError(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("Not Found"))
	}))
	defer s.Close()

	dir := t.TempDir()
	env := map[string]string{
		"luadir":       dir,
		"download_url": s.URL + "/",
	}
	luaContent := `
	local dir = os.getenv("luadir")
	local url = os.getenv("download_url")
	local r = bz.download(url, dir .. "/example.txt")
	if not r.ok then
		error("download failed: " .. r.error)
	end
	return {
		env = {
			luadir=dir,
		},
	}
	`
	_, file := writeTempLuaFile(t, luaContent)
	err := RunLuaScript(file, env, nil)
	if err == nil {
		t.Fatalf("expected error due to download failure, got nil")
	}
}

func writeTempLuaFile(t *testing.T, content string) (string, string) {
	t.Helper()
	dir := t.TempDir()
	file := filepath.Join(dir, "script.lua")
	if err := os.WriteFile(file, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write temp lua file: %v", err)
	}
	return dir, file
}

func TestRunLuaScript_ZipDirectory(t *testing.T) {
	dir := t.TempDir()
	env := map[string]string{
		"luadir": dir,
	}
	luaContent := `
	local dir = os.getenv("luadir")

	bz.mkdir(dir .. "/ziptest")
	local f = io.open(dir .. "/ziptest/file1.txt", "w")
	f:write("file1")
	f:close()

	local f2 = io.open(dir .. "/ziptest/file2.txt", "w")
	f2:write("file2")
	f2:close()
	bz.chmod(dir .. "/ziptest/file2.txt", "rw-rw-rwx")

	-- create file3 with executable permissions
	local f3 = io.open(dir .. "/ziptest/file3.sh", "w")
	f3:write("#!/bin/bash\necho Hello")
	f3:close()
	local chmoderr = bz.chmod(dir .. "/ziptest/file3.sh", "0755")
	if not chmoderr.ok then
		error("chmod failed: " .. chmoderr.error)
	end

	local zipPath = dir .. "/test.zip"
	local r = bz.zip(dir .. "/ziptest", zipPath)
	if not r.ok then
		error("zip failed: " .. r.error)
	end

	-- Check if the zip file was created
	local fzip = io.open(zipPath, "r")
	if not fzip then
		error("zip file was not created")
	end
	fzip:close()

	local e = bz.unzip(zipPath, dir .. "/unzipped")
	if not e.ok then
		error("unzip failed: " .. e.error)
	end

	-- Check if the unzipped files exist
	local f1 = io.open(dir .. "/unzipped/file1.txt", "r")
	if not f1 then
		error("unzipped file1.txt does not exist")
	end
	f1:close()
	local f2 = io.open(dir .. "/unzipped/file2.txt", "r")
	if not f2 then
		error("unzipped file2.txt does not exist")
	end
	f2:close()
	local stat, err = bz.stat(dir .. "/unzipped/file2.txt")
	if not err.ok then
		error("stat failed: " .. err.error)
	end
	bz.debug(dir)
	bz.debug("hello")
	if stat.permission ~= "rw-rw-rwx" then
		error("file2.txt does not have expected permissions. expected rw-rw-rwx, but Got: " .. stat.permission)
	end

	-- check file3 permissions
	stat, err = bz.stat(dir .. "/unzipped/file3.sh")
	if not err.ok then
		error("stat failed: " .. err.error)
	end
	if stat.permission ~= "rwxr-xr-x" then
		error("file3.sh does not have expected permissions. expected rwxr-xr-x, but Got: " .. stat.permission)
	end

	return {
		env = {
			luadir=dir,
		},
	}
	`
	_, file := writeTempLuaFile(t, luaContent)
	err := RunLuaScript(file, env, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
