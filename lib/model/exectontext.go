package model

import (
	"fmt"
	"os"
	"strings"

	"mvdan.cc/sh/shell"
)

type ExecContext struct {
	path   []string
	envMap map[string]string
	Sub    []ExecContext
	Alias  map[string]string
}

func (o *ExecContext) SetPath(path []string) {
	o.path = path
}

func (o *ExecContext) Env() map[string]string {
	m := make(map[string]string)
	for _, s := range o.Sub {
		for k, v := range s.Env() {
			m[k] = v
		}
	}
	if o.envMap != nil {
		for k, v := range o.envMap {
			m[k] = v
		}
	}

	//
	o.prependPath(m)

	return m
}

func (o *ExecContext) prependPath(m map[string]string) {
	if o.path == nil {
		return
	}
	if len(o.path) < 1 {
		return
	}

	pathStr, ok := m["PATH"]
	var pathParts []string
	if ok && len(pathStr) > 0 {
		pathParts = strings.Split(pathStr, string([]rune{os.PathListSeparator}))
	}

	// path
	var p []string
	p = append(p, o.path...)
	p = append(p, pathParts...)
	if len(p) > 0 {
		m["PATH"] = strings.Join(p, string([]rune{os.PathListSeparator}))
	}
}

func (o *ExecContext) Set(k, v string) *ExecContext {
	if o.envMap == nil {
		o.envMap = make(map[string]string)
	}

	o.envMap[k] = v
	return o
}

func (o *ExecContext) ResolveAlias(args []string) []string {
	var result []string = args
	arg := args[0]
	if str, ok := o.Alias[arg]; ok {
		e := o.Env()
		expanded, _ := shell.Fields(str, func(k string) string {
			if v, ok := e[k]; ok {
				return v
			}
			return fmt.Sprintf("$%s", k)
		})
		result = append(expanded, args[1:]...)
	}

	// Resolve Sub Aliases
	for _, s := range o.Sub {
		result = s.ResolveAlias(result)
	}

	return result
}
