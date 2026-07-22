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
			if k == "PATH" {
				newPaths := strings.Split(v, string([]rune{os.PathListSeparator})) // v == /path1:/path2
				o.prependPath(m, newPaths...)
			} else {
				m[k] = v
			}
		}
	}
	if o.envMap != nil {
		for k, v := range o.envMap {
			m[k] = v
		}
	}

	//
	o.prependPath(m, o.path...)

	return m
}

func (o *ExecContext) prependPath(m map[string]string, newPaths0 ...string) {
	o.pathAsSlice(m, newPaths0, func(existingPaths []string, newPaths1 []string) []string {
		var p []string
		p = append(p, newPaths1...) // add new before existing paths
		p = append(p, existingPaths...)
		return p
	})
}

// func (o *ExecContext) appendPath(m map[string]string, newPaths0 ...string) {
// 	o.pathAsSlice(m, newPaths0, func(existingPaths []string, newPaths0 []string) []string {
// 		var p []string
// 		p = append(p, existingPaths...) // add existing before new paths
// 		p = append(p, newPaths0...)
// 		return p
// 	})
// }

func (o *ExecContext) pathAsSlice(m map[string]string, pathsToAdd []string, f func(existing []string, new []string) []string) {
	if pathsToAdd == nil {
		return
	}
	if len(pathsToAdd) < 1 {
		return
	}

	pathStr, ok := m["PATH"]
	var pathParts []string
	if ok && len(pathStr) > 0 {
		pathParts = strings.Split(pathStr, string([]rune{os.PathListSeparator}))
	}

	// don't add duplicates to PATH
	existing := make(map[string]bool)
	for _, p := range pathParts {
		existing[p] = true
	}
	var newParts []string
	for _, p := range pathsToAdd {
		if _, ok := existing[p]; !ok {
			newParts = append(newParts, p)
			existing[p] = true
		}
	}
	if len(newParts) < 1 {
		return
	}

	p := f(pathParts, newParts)
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

func (o *ExecContext) StrToArgs(str string) ([]string, error) {
	e := o.Env()
	args, err := shell.Fields(str, func(k string) string {
		if v, ok := e[k]; ok {
			return v
		}
		return fmt.Sprintf("$%s", k)
	})
	return args, err
}

func (o *ExecContext) Expand(str string) (string, error) {
	e := o.Env()
	args, err := shell.Expand(str, func(k string) string {
		if v, ok := e[k]; ok {
			return v
		}
		return fmt.Sprintf("$%s", k)
	})
	return args, err
}
