// SPDX-FileCopyrightText: 2023 RH America LLC <info@rhamerica.com>
// SPDX-License-Identifier: GPL-3.0-only

package model

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/bazurto/bz/lib/utils"
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
}

func (ed *DependencyTree) BinDirOrDefault() string {
	if ed.BinDir == "" {
		return filepath.Join(ed.Dir, "bin") // default bindir
	}
	return ed.BinDir
}

// Resolve returns a slice with bin dirs to be prepended to PATH os var
// and a map with all environment variables to be added.  It resolves all
// of these values recursively.  The callback function can be null
func (ed *DependencyTree) Resolve() *ExecContext {
	// Sub
	var subCtx []ExecContext
	for _, sub := range ed.Sub {
		tmp := sub.Resolve()
		subCtx = append(subCtx, *tmp)
	}
	// Local
	ctx := ed.resolveLocalEnvVars(subCtx)

	// binDir
	binDir := ed.BinDirOrDefault()

	ctx.SetPath([]string{
		parseShellTpl(binDir, ctx.Env()),
	})

	return ctx
}

// resolveLocalEnvVars returns map with implicit variables and exported ones
// for only this extracted dependecy
func (ed *DependencyTree) resolveLocalEnvVars(subCtx []ExecContext) *ExecContext {
	ctx := &ExecContext{}
	env := make(map[string]string)
	env["DIR"] = ed.Dir // DIR
	env["CURDIR"] = utils.GetCurrentDir()
	env["BZ_PROJECT_DIR"] = os.Getenv("BZ_PROJECT_DIR")
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

	// execute preRun
	//TODO: preRun
	// if ed.Triggers.PreRunScript != "" {
	// 	b, err := json.Marshal(ctx)
	// 	if err != nil {
	// 		return ctx
	// 	}
	// 	fmt.Println(string(b))
	// 	in := bytes.NewBuffer(b)
	// 	out := bytes.NewBuffer(nil)
	// 	exec.ExecCommandStr(
	// 		ctx,
	// 		ed.Triggers.PreRunScript,
	// 		out,
	// 		os.Stderr,
	// 		in,
	// 	)
	// 	var newCtx *ExecContext
	// 	if err := json.Unmarshal(out.Bytes(), newCtx); err != nil {
	// 		return ctx
	// 	}
	// 	fmt.Println(string(out.Bytes()))
	// 	//ctx = newCtx
	// }

	return ctx
}

func calculateImplicitDirEnvironmentVars(ea DependencyTree, env map[string]string) map[string]string {
	c := ea.Coord
	m := make(map[string]string)

	host := c.URL.Hostname()
	path := strings.Trim(c.URL.Path, "/") // remove leading slacshes to avoid creating variables with two underscores
	version := fmt.Sprintf("V%s", c.URL.Fragment)
	if version == "V" {
		version = "V0"
	}
	nameSpaceVarPrefixes := []string{
		// GITHUB_COM_BAZURTO_GROOVY_V1.2.3
		utils.ToEnvKey(fmt.Sprintf("%s_%s_%s", host, path, version)),
		// GITHUB_COM_BAZURTO_GROOVY
		utils.ToEnvKey(fmt.Sprintf("%s_%s", host, path)),
		// BAZURTO_GROOVY_V1.2.3
		utils.ToEnvKey(fmt.Sprintf("%s_%s", path, version)),

		// BAZURTO_GROOVY_V1.2.3
		utils.ToEnvKey(fmt.Sprintf("%s_%s", path, version)),

		// BAZURTO_GROOVY
		utils.ToEnvKey(path),

		// // GROOVY
		// utils.ToEnvKey(c.Repo),

		// // GROOVY_V1.2.3
		// utils.ToEnvKey(fmt.Sprintf("%s_%s", c.Repo, c.Version.Canonical())),
	}

	// github.com/bazurto/groovy-v1.2.3 =>  {
	//  "GITHUB_COM_BAZURTO_GROOVY_V1.2.3_DIR" : "/path/to/dir/extracted",
	//  "GITHUB_COM_BAZURTO_GROOVY_V1.2.3_BINDIR" : "/path/to/dir/extracted/bin",
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
