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
		"unzip":    luaUnZip,
		"mkdir":    luaMkdir,
		"download": luaDownload,
		"chmod":    luaChmod,
		"stat":     luaStat,
		"debug":    luaDebug,
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

func luaChmod(l *lua.LState) int {
	path := l.CheckString(1)    // get first argument
	modeStr := l.CheckString(2) // get second argument
	mode := modeStrToInt(modeStr)
	err := os.Chmod(path, os.FileMode(mode))
	if err != nil {
		l.Push(luaError(l, false, err.Error())) // push to stack
		return 1                                // number of return values
	}
	l.Push(luaError(l, true, "")) // push to stack
	return 1                      // number of return value
}

// modeStrToInt converts a string representation of a file mode to an os.FileMode value.
// The input string can be either an octal number (e.g., "0755") or a symbolic representation
// (e.g., "rwxr-xr-x"). If the string is empty or invalid, it returns 0.
// Octal parsing is attempted first; if it fails, symbolic parsing is used.
// Symbolic representation must be exactly 9 characters, corresponding to "rwxrwxrwx".
func modeStrToInt(modeStr string) os.FileMode {
	if len(modeStr) == 0 {
		return 0
	}
	// Try to parse as octal number first
	var mode uint32
	n, err := fmt.Sscanf(modeStr, "%o", &mode)
	if err == nil && n == 1 {
		return os.FileMode(mode)
	}

	// If parsing as octal fails, try to parse as symbolic representation
	var perm os.FileMode
	if len(modeStr) != 9 && len(modeStr) != 10 {
		return 0 // Invalid length for symbolic representation
	}
	for i, c := range modeStr {
		switch i {
		case 0:
			if c == 'r' {
				perm |= 0400
			}

		case 1:
			if c == 'w' {
				perm |= 0200
			}
		case 2:
			switch c {
			case 'x':
				perm |= 0100
			case 's':
				perm |= os.ModeSetuid | 0100
			case 'S':
				perm |= os.ModeSetuid
			}
		case 3:
			if c == 'r' {
				perm |= 0040
			}
		case 4:
			if c == 'w' {
				perm |= 0020
			}
		case 5:
			switch c {
			case 'x':
				perm |= 0010
			case 's':
				perm |= os.ModeSetgid | 0010
			case 'S':
				perm |= os.ModeSetgid
			}
		case 6:
			if c == 'r' {
				perm |= 0004
			}
		case 7:
			if c == 'w' {
				perm |= 0002
			}
		case 8:
			switch c {
			case 'x':
				perm |= 0001
			case 't':
				perm |= os.ModeSticky | 0001
			case 'T':
				perm |= os.ModeSticky
			}
		}
	}
	return perm
}

func luaDebug(l *lua.LState) int {
	v := l.Get(1)            // get first argument
	fmt.Println("DEBUG:", v) // print to stdout
	return 0
}

func luaStat(l *lua.LState) int {
	path := l.CheckString(1) // get first argument
	stat, err := os.Stat(path)
	if err != nil {
		l.Push(lua.LNil)
		l.Push(luaError(l, false, err.Error())) // push to stack
		return 1                                // number of return values
	}
	//
	t := l.NewTable()
	t.RawSetString("name", lua.LString(stat.Name()))
	t.RawSetString("size", lua.LNumber(stat.Size()))
	t.RawSetString("mode", lua.LNumber(uint32(stat.Mode())))
	t.RawSetString("permission", lua.LString(
		string([]rune(stat.Mode().String())[1:]), // skip the first character which indicates the file type
	),
	)

	t.RawSetString("modtime", lua.LNumber(stat.ModTime().Unix()))
	t.RawSetString("isdir", lua.LBool(stat.IsDir()))
	l.Push(t)
	l.Push(luaError(l, true, "")) // push to stack
	return 2                      // number of return value
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
		//fmt.Fprintf(os.Stderr, "DEBUG: %s", err)
		l.Push(luaError(l, false, err.Error())) // push to stack
		return 1                                // number of return values
	}
	//fmt.Fprintf(os.Stderr, "DOWNLOAD SUSCESSSFUL")

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

func luaUnZip(l *lua.LState) int {
	src := l.CheckString(1) // get first argument
	dst := l.CheckString(2) // get second argument

	err := utils.Unzip(src, dst)
	if err != nil {
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
