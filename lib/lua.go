package lib

import (
	"fmt"
	"os"
	"path"

	lua "github.com/yuin/gopher-lua"
)

func RunLua(code string) error {
	L := lua.NewState()
	defer L.Close()
	L.SetGlobal("mkdir", L.NewFunction(Mkdir)) /* Original lua_setglobal uses stack... */
	L.PreloadModule("file", PathModLoader)
	return L.DoString(code)
}

func Mkdir(L *lua.LState) int {
	dir := L.ToString(1) /* get argument */
	if err := os.MkdirAll(dir, 0755); err != nil {
		L.RaiseError("%s", err)
		return 0
	}
	//L.Push(lua.LNumber(2)) /* push result */
	return 0 /* number of results */
}

// =====================================================
func PathModLoader(L *lua.LState) int {
	// register functions to the table
	mod := L.SetFuncs(L.NewTable(), fileExports)

	/*
		// register other stuff
		L.SetField(mod, "name", lua.LString("value"))
	*/

	// returns the module
	L.Push(mod)
	return 1
}

var fileExports = map[string]lua.LGFunction{
	"join":       join,
	"exists":     exists,
	"mkdir":      fileMkdir,
	"tgz":        fileTgz,
	"untgz":      fileUntgz,
	"uncompress": fileUncompress,
}

func join(L *lua.LState) int {
	var args []string
	i := 1
	for {
		v := L.Get(i)
		if v == lua.LNil {
			break
		}
		args = append(args, lua.LVAsString(v))
		i++
	}
	ret := path.Join(args...)
	L.Push(lua.LString(ret)) /* push result */
	return 1                 /* number of results */
}

func exists(L *lua.LState) int {
	file := L.ToString(1)
	var fileExists bool
	_, err := os.Stat(file)
	if err != nil {
		if !os.IsNotExist(err) {
			L.RaiseError("%s", err)
			return 0
		}
		fileExists = false
	} else {
		fileExists = true
	}
	L.Push(lua.LBool(fileExists)) /* push result */
	return 1
}

func fileMkdir(L *lua.LState) int {
	dir := L.ToString(1) /* get argument */
	if err := os.MkdirAll(dir, 0755); err != nil {
		L.RaiseError("%s", err)
		return 0
	}
	//L.Push(lua.LNumber(2)) /* push result */
	return 0 /* number of results */
}

func fileUncompress(L *lua.LState) int {
	filename := L.ToString(1) /* get argument */
	dir := L.ToString(2)      /* get argument */
	err := Uncompress(filename, dir)
	if err != nil {
		L.RaiseError("%s", err)
	}
	return 0
}

func fileTgz(L *lua.LState) int {
	L.RaiseError("%s", fmt.Errorf("Not Implemented"))
	return 0
}

func fileUntgz(L *lua.LState) int {
	filename := L.ToString(1) /* get argument */
	dir := L.ToString(2)      /* get argument */
	err := Untgz(filename, dir)
	if err != nil {
		L.RaiseError("%s", err)
	}
	return 0
}
