package lib

import (
	"fmt"
	"os"
	"path"

	"github.com/robertkrimen/otto"
)

func RunScriptCode(code string, args ...any) (map[string]any, error) {
	pathModule := make(map[string]func(call otto.FunctionCall) otto.Value)
	fileModule := make(map[string]func(call otto.FunctionCall) otto.Value)
	stdModule := make(map[string]func(call otto.FunctionCall) otto.Value)

	stdModule["printf"] = printf
	pathModule["join"] = join

	fileModule["exists"] = exists
	fileModule["mkdir"] = fileMkdir
	fileModule["tgz"] = fileTgz
	fileModule["untgz"] = fileUntgz
	fileModule["uncompress"] = fileUncompress

	exports := make(map[string]any)
	vm := otto.New()

	vm.Set("path", pathModule)
	vm.Set("file", fileModule)
	vm.Set("std", stdModule)
	vm.Set("args", args)

	vm.Set("exports", exports)

	_, err := vm.Run(code)

	return exports, err
}

func join(call otto.FunctionCall) otto.Value {
	var args []string
	for _, v := range call.ArgumentList {
		v := v.String()
		args = append(args, v)
	}
	ret := path.Join(args...)
	v, _ := call.Otto.ToValue(ret)
	return v
}

func exists(call otto.FunctionCall) otto.Value {
	file, err := call.Argument(0).ToString()
	if err != nil {
		panic(err)
	} else if true {
		panic("test")
	}
	var fileExists bool
	_, err = os.Stat(file)
	if err != nil {
		if !os.IsNotExist(err) {
			panic(err)
		}
		fileExists = false
	} else {
		fileExists = true
	}
	v, err := call.Otto.ToValue(fileExists)
	if err != nil {
		panic(err)
	}
	return v
}

func fileMkdir(call otto.FunctionCall) otto.Value {
	dir := call.Argument(0).String()
	if err := os.MkdirAll(dir, 0755); err != nil {
		panic(err)
	}
	return otto.NullValue()
}

func fileUncompress(call otto.FunctionCall) otto.Value {
	filename := call.Argument(0).String()
	dir := call.Argument(1).String()
	err := Uncompress(filename, dir)
	if err != nil {
		panic(err)
	}
	return otto.NullValue()
}

func fileTgz(call otto.FunctionCall) otto.Value {
	panic("TGZ NOT IMPLEMENTED")
	//return otto.NullValue()
}

func fileUntgz(call otto.FunctionCall) otto.Value {
	filename := call.Argument(0).String()
	dir := call.Argument(1).String()
	err := Untgz(filename, dir)
	if err != nil {
		panic(err)
	}
	return otto.NullValue()
}

func printf(call otto.FunctionCall) otto.Value {
	if len(call.ArgumentList) < 1 {
		return otto.NullValue()
	}

	var format = fmt.Sprintf("%s", call.Argument(0).String())
	var args []any
	for _, v := range call.ArgumentList[1:] {
		tmp, _ := v.Export()
		args = append(args, tmp)
	}
	fmt.Printf(format, args...)
	return otto.NullValue()
}
