// SPDX-FileCopyrightText: 2023 RH America LLC <info@rhamerica.com>
// SPDX-License-Identifier: GPL-3.0-only

package model

import (
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/bazurto/bz/lib/luautils"
	"github.com/bazurto/bz/lib/utils"
	lua "github.com/yuin/gopher-lua"
	"mvdan.cc/sh/shell"
)

// DependencyTree is used to resolve execution environment configuration.
// It was previously called ResolvedDependency
type DependencyTree struct {
	Coord    LockedCoord
	Dir      string            // directory where it is extracted
	BinDir   string            // directory where binaries are extracted
	Exports  map[string]string // environment vars
	Alias    map[string]string // aliases
	Triggers Triggers          // triggers
	Sub      []*DependencyTree // Sub Dependencies

	resolved *ExecContext // memoized Resolve result so each preRunScript runs at most once per process
}

func (ed *DependencyTree) BinDirOrDefault() string {
	if ed.BinDir == "" {
		return filepath.Join(ed.Dir, "bin") // default bindir
	}
	return ed.BinDir
}

// Resolve returns a slice with bin dirs to be prepended to PATH os var
// and a map with all environment variables to be added.  It resolves all
// of these values recursively.  The result is memoized per node so that
// preRunScript triggers execute at most once per process.
func (ed *DependencyTree) Resolve() (*ExecContext, error) {
	if ed.resolved != nil {
		return ed.resolved, nil
	}
	ctx, err := ed.resolve(true)
	if err != nil {
		return nil, err
	}
	ed.resolved = ctx
	return ctx, nil
}

// ResolveSkipSelfPreRun resolves like Resolve but without running this
// node's own preRunScript.  It is used to expand the install script path and
// environment: at that point the install script has not run yet, so the
// node's pre-run script (which may inspect what the install produced) must
// wait until execution time.  Sub dependency pre-run scripts still run
// (memoized) because their install scripts have already completed.  The
// result is not memoized.
func (ed *DependencyTree) ResolveSkipSelfPreRun() (*ExecContext, error) {
	return ed.resolve(false)
}

func (ed *DependencyTree) resolve(runSelfPreRun bool) (*ExecContext, error) {
	// Sub
	var subCtx []ExecContext
	for _, sub := range ed.Sub {
		tmp, err := sub.Resolve()
		if err != nil {
			return nil, err
		}
		subCtx = append(subCtx, *tmp)
	}

	// Local
	ctx, err := ed.resolveLocalEnvVars(subCtx, runSelfPreRun)
	if err != nil {
		return nil, err
	}

	// binDir
	binDir := ed.BinDirOrDefault()

	ctx.SetPath([]string{
		parseShellTpl(binDir, ctx.Env()),
	})

	return ctx, nil
}

// resolveLocalEnvVars returns map with implicit variables and exported ones
// for only this extracted dependecy
func (ed *DependencyTree) resolveLocalEnvVars(subCtx []ExecContext, runPreRun bool) (*ExecContext, error) {
	ctx := &ExecContext{}
	env := make(map[string]string)
	env["DIR"] = ed.Dir // DIR
	env["CURDIR"] = utils.GetCurrentDir()
	env["BZ_PROJECT_DIR"] = os.Getenv("BZ_PROJECT_DIR")
	env["GOOS"] = runtime.GOOS
	env["GOARCH"] = runtime.GOARCH
	env = utils.MapMerge(ctx.Env(), env)
	//
	for k, v := range ed.Exports {
		env[k] = parseShellTpl(v, env)
	}

	// do BINDIR after Exports
	env["BINDIR"] = parseShellTpl(ed.BinDirOrDefault(), env) // BINDIR=/path/to/bin

	// all of github.com/owner/repo-v1.2.3 =>  GITHUB_COM_OWNER_REPO_V1.2.3 = /path/to/dir/extracted
	implicitVars := calculateImplicitDirEnvironmentVars(*ed, env)
	env = utils.MapMerge(env, implicitVars)

	for k, v := range env {
		ctx.Set(k, v)
	}

	ctx.Alias = ed.Alias
	ctx.Sub = subCtx

	// Override variables with Lua
	if runPreRun && ed.Triggers.PreRunScript != "" {
		preRunScriptFinal, err := ctx.Expand(ed.Triggers.PreRunScript) // parses variables $DIR/script.lua
		if err != nil {
			return nil, err
		}

		env := utils.OsEnvironment() // os.Environ
		maps.Copy(env, ctx.Env())    // copy dependency environment to pass to lua
		if err := luautils.RunLuaScript(preRunScriptFinal, env, func(retval *lua.LTable) {
			//
			v := retval.RawGetString("env")
			if t, ok := v.(*lua.LTable); ok {
				t.ForEach(func(l1, l2 lua.LValue) {
					ctx.Set(l1.String(), l2.String())
				})
			}

			//
			v = retval.RawGetString("alias")
			if t, ok := v.(*lua.LTable); ok {
				t.ForEach(func(l1, l2 lua.LValue) {
					ctx.Alias[l1.String()] = l2.String()
				})
			}

			//
			v = retval.RawGetString("path")
			if t, ok := v.(*lua.LTable); ok {
				var path []string
				t.ForEach(func(l1, l2 lua.LValue) {
					path = append(path, l2.String())
				})
				ctx.SetPath(path)
			} else if t, ok := v.(*lua.LString); ok {
				ctx.SetPath([]string{t.String()})
			}

		}); err != nil {
			return nil, err
		}
	}

	return ctx, nil
}

func calculateImplicitDirEnvironmentVars(dt DependencyTree, env map[string]string) map[string]string {
	c := dt.Coord
	m := make(map[string]string)

	host := c.URL.Hostname()
	path := strings.Trim(c.URL.Path, "/") // remove leading slacshes to avoid creating variables with two underscores
	version := c.URL.Fragment
	if version == "" {
		version = "0"
	}

	var nameSpaceVarPrefixes []string
	// GITHUB_COM_BAZURTO_GROOVY_1_2_3
	if host != "" { // host is empty for file:///
		nameSpaceVarPrefixes = append(nameSpaceVarPrefixes, utils.ToEnvKey(fmt.Sprintf("%s_%s_%s", host, path, version)))
	}
	// GITHUB_COM_BAZURTO_GROOVY
	if host != "" {
		nameSpaceVarPrefixes = append(nameSpaceVarPrefixes, utils.ToEnvKey(fmt.Sprintf("%s_%s", host, path)))
	}

	// file:///home/user/myproject
	//	- USER_PROJECT
	// github.com/bazurto/groovy-v1.2.3
	//	- BAZURTO_GROOVY_1_2_3
	//  - GROOVY_1_2_3
	//  - BAZURTO_GROOVY
	//  - GROOVY
	lastTwoPathParts := strings.Split(path, "/")
	if len(lastTwoPathParts) >= 2 {
		lastTwoPathParts = lastTwoPathParts[len(lastTwoPathParts)-2:]
	}

	for tmp := lastTwoPathParts; len(tmp) > 0; tmp = tmp[1:] {
		nameSpaceVarPrefixes = append(nameSpaceVarPrefixes, utils.ToEnvKey(fmt.Sprintf("%s_%s", strings.Join(tmp, "_"), version)))
		nameSpaceVarPrefixes = append(nameSpaceVarPrefixes, utils.ToEnvKey(strings.Join(tmp, "_")))
	}

	// github.com/bazurto/groovy-v1.2.3 =>  {
	//  "GITHUB_COM_BAZURTO_GROOVY_1_2_3_DIR" : "/path/to/dir/extracted",
	//  "GITHUB_COM_BAZURTO_GROOVY_1_2_3_BINDIR" : "/path/to/dir/extracted/bin",
	//  "GITHUB_COM_BAZURTO_GROOVY_DIR" : "/path/to/dir/extracted",
	//  "GITHUB_COM_BAZURTO_GROOVY_BINDIR" : "/path/to/dir/extracted",
	//  "BAZURTO_GROOVY_V1.2.3_DIR" : "/path/to/dir/extracted",
	//  "BAZURTO_GROOVY_V1.2.3_BINDIR" : "/path/to/dir/extracted/bin",
	//  "GROOVY_V1.2.3_DIR" : "/path/to/dir/extracted",
	//  "GROOVY_V1.2.3_BINDIR" : "/path/to/dir/extracted/bin",
	//  "GROOVY_DIR" : "/path/to/dir/extracted",
	//  "GROOVY_BINDIR" : "/path/to/dir/extracted/bin",
	// }
	for _, ns := range nameSpaceVarPrefixes {
		m[fmt.Sprintf("%s_DIR", ns)] = env["DIR"]
		m[fmt.Sprintf("%s_BINDIR", ns)] = env["BINDIR"]
	}

	return m
}

func parseShellTpl(tpl string, env map[string]string) string {
	f := func(tplVar string) string {
		if v, ok := env[tplVar]; ok {
			return v
		}
		return tplVar
	}
	parsedStr, _ := shell.Expand(tpl, f)
	return parsedStr
}
