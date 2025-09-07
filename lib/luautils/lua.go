// SPDX-FileCopyrightText: 2025 RH America LLC <info@rhamerica.com>
// SPDX-License-Identifier: GPL-3.0-only

package luautils

import (
	"fmt"
	"os"

	"github.com/bazurto/bz/lib/model"
	"github.com/bazurto/bz/lib/utils"

	lua "github.com/yuin/gopher-lua"
)

// RunLuaInstallScript takes a lua file as argument `file` and and execution environment `execCtx` to use
// to be passed as environment variable for the script.  It will call `cb` and apss the lua return table
// and the modified environment variables that were originally passed to the script
func RunLuaInstallScript(file string, execCtx *model.ExecContext, cb func(retval *lua.LTable, env map[string]string)) error {
	luaEnv := execCtx.Env()

	l := newBzLuaState(luaEnv) // the lua env will be modified if the script calls os.setenv
	defer l.Close()

	// Load and run the Lua config file
	if err := l.DoFile(file); err != nil {
		return err
	}

	ret := l.Get(-1) // L.Get(-1) is the return from the file
	if ret.Type() != lua.LTTable {
		return fmt.Errorf("lua script did not return a table")
	}

	if cb != nil {
		luaTbl := ret.(*lua.LTable)
		cb(luaTbl, luaEnv)
	}
	return nil
}

func luaError(L *lua.LState, ok bool, err string) *lua.LTable {
	t := L.NewTable()
	t.RawSetString("ok", lua.LBool(ok))
	t.RawSetString("error", lua.LString(err))
	return t
}

func newBzLuaState(luaNs map[string]string) *lua.LState {
	L := lua.NewState()

	//
	// Replace the os.getenv and os.setenv with an isolated environment
	// instead of the one wired to Go's os.SetEnv and os.GetEnv
	//
	osNS := L.GetGlobal("os")
	if tbl, ok := osNS.(*lua.LTable); ok {
		L.SetFuncs(tbl, map[string]lua.LGFunction{
			"getenv": getBzGetEnvFunc(luaNs),
			"setenv": getBzSetEnvFunc(luaNs),
		})
	}

	//
	// bz package
	//  bz.zip
	bzNS := L.NewTable()
	L.SetFuncs(bzNS, map[string]lua.LGFunction{
		"zip":   luaZip,
		"mkdir": luaMkdir,
	})
	L.SetGlobal("bz", bzNS)
	return L
}

func getBzSetEnvFunc(env map[string]string) func(L *lua.LState) int {
	return func(L *lua.LState) int {
		env[L.CheckString(1)] = L.CheckString(2)
		L.Push(lua.LTrue)
		return 1
	}
}

func getBzGetEnvFunc(env map[string]string) func(L *lua.LState) int {
	return func(L *lua.LState) int {
		v := env[L.CheckString(1)]
		if len(v) == 0 {
			L.Push(lua.LNil)
		} else {
			L.Push(lua.LString(v))
		}
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
