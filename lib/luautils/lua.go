// SPDX-FileCopyrightText: 2025 RH America LLC <info@rhamerica.com>
// SPDX-License-Identifier: GPL-3.0-only

package luautils

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/bazurto/bz/lib/utils"

	lua "github.com/yuin/gopher-lua"
)

// RunLuaScript takes a lua file as argument `file` and and execution environment `execCtx` to use
// to be passed as environment variable for the script.  It will call `cb` and apss the lua return table
// and the modified environment variables that were originally passed to the script
func RunLuaScript(file string, env map[string]string, cb func(retval *lua.LTable)) error {
	l := newBzLuaState(env) // the lua env will be modified if the script calls os.setenv
	defer l.Close()

	// Load and run the Lua config file
	if err := l.DoFile(file); err != nil {
		return err
	}

	if cb != nil {
		ret := l.Get(-1) // L.Get(-1) is the return from the file
		if ret.Type() != lua.LTTable {
			return fmt.Errorf("%s script must return a table. e.g.: return {env={...}}", file)
		}
		luaTbl := ret.(*lua.LTable)
		cb(luaTbl)
	}
	return nil
}

func luaError(L *lua.LState, ok bool, err string) *lua.LTable {
	t := L.NewTable()
	t.RawSetString("ok", lua.LBool(ok))
	t.RawSetString("error", lua.LString(err))
	return t
}

func newBzLuaState(env map[string]string) *lua.LState {
	L := lua.NewState()

	luaEnv := L.NewTable()
	for k, v := range env {
		luaEnv.RawSetString(k, lua.LString(v))
	}

	//
	// Replace the os.getenv and os.setenv with an isolated environment
	// instead of the one wired to Go's os.SetEnv and os.GetEnv
	//
	osNS := L.GetGlobal("os")
	if tbl, ok := osNS.(*lua.LTable); ok {
		L.SetFuncs(tbl, map[string]lua.LGFunction{
			"getenv": getBzGetEnvFunc(luaEnv),
			"setenv": getBzSetEnvFunc(luaEnv),
		})
		tbl.RawSetString("environment", luaEnv)

	}

	//
	// bz package
	//  bz.zip
	bzNS := L.NewTable()
	L.SetFuncs(bzNS, map[string]lua.LGFunction{
		"zip":      luaZip,
		"mkdir":    luaMkdir,
		"download": luaDownload,
	})
	L.SetGlobal("bz", bzNS)
	return L
}

func getBzSetEnvFunc(env *lua.LTable) func(L *lua.LState) int {
	return func(l *lua.LState) int {
		env.RawSetString(l.CheckString(1), l.CheckAny(2))
		//env[] = L.CheckString(2)
		l.Push(lua.LTrue)
		return 1
	}
}

func getBzGetEnvFunc(env *lua.LTable) func(L *lua.LState) int {
	return func(l *lua.LState) int {
		v := env.RawGetString(l.CheckString(1))
		l.Push(v)
		return 1
	}
}

func luaMkdir(l *lua.LState) int {
	dir := l.CheckString(1) // get first argument

	var perm os.FileMode = 0755
	if l.GetTop() >= 2 { // if second argument is provided
		perm = os.FileMode(l.CheckInt(2))
	}

	if err := os.MkdirAll(dir, perm); err != nil {
		l.Push(luaError(l, false, err.Error())) // push to stack
		return 1                                // number of return values
	}

	l.Push(luaError(l, true, "")) // push to stack
	return 1                      // number of return values
}

func luaDownload(l *lua.LState) int {
	url := l.CheckString(1)     // get first argument
	dstFile := l.CheckString(2) // get first argument
	basePath := filepath.Dir(dstFile)

	var perm os.FileMode = 0755
	if err := os.MkdirAll(basePath, perm); err != nil {
		l.Push(luaError(l, false, err.Error())) // push to stack
		return 1                                // number of return values
	}

	if err := downloadFile(url, dstFile); err != nil {
		l.Push(luaError(l, false, err.Error())) // push to stack
		return 1                                // number of return values
	}

	l.Push(luaError(l, true, "")) // push to stack
	return 1                      // number of return values
}

func luaZip(l *lua.LState) int {
	src := l.CheckString(1) // get first argument
	dst := l.CheckString(2) // get second argument
	opts := utils.ZipAllOpts{}
	if l.GetTop() >= 3 { // if second argument is provided
		luaOpts := l.CheckTable(3)
		if echo, ok := luaOpts.RawGetString("echo").(*lua.LBool); ok {
			opts.Echo = bool(*echo)
		}
	}

	dstWriter, err := os.Create(dst)
	if err != nil {
		l.Push(luaError(l, false, err.Error())) // push to stack
		return 1
	}
	defer dstWriter.Close()

	if err := utils.ZipAll(src, dstWriter, opts); err != nil {
		l.Push(luaError(l, false, err.Error())) // push to stack
		return 1
	}

	l.Push(luaError(l, true, "")) // push to stack
	return 1
}

func downloadFile(url string, filepath string) error {
	// Create the file
	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	// Get the data
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Check server response
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %s", resp.Status)
	}

	// Write the body to file
	_, err = io.Copy(out, resp.Body)
	return err
}

func doHttp(
	method string,
	url string,
	body io.Reader,
	headers map[string]string,
	proc func(*http.Response),
) error {
	c := http.Client{}
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return err
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	rep, err := c.Do(req)
	proc(rep)
	return err
}
