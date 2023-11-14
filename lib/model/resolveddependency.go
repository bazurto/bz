// SPDX-FileCopyrightText: 2023 RH America LLC <info@rhamerica.com>
// SPDX-License-Identifier: GPL-3.0-only

package model

import (
	"fmt"
	"path/filepath"

	"github.com/bazurto/bz/lib/model"
	"github.com/bazurto/bz/lib/utils"
	"mvdan.cc/sh/shell"
)

type ResolvedDependency struct {
	Coord    LockedCoord
	Dir      string                // directory where it is extracted
	BinDir   string                // directory where binaries are extracted
	Exports  map[string]string     // environment vars
	Alias    map[string]string     // aliases
	Triggers Triggers              // triggers
	Sub      []*ResolvedDependency // Sub Dependencies
}

func (ed *ResolvedDependency) BinDirOrDefault() string {
	if ed.BinDir == "" {
		return filepath.Join(ed.Dir, "bin") // default bindir
	}
	return ed.BinDir
}

// func (ed *ResolvedDependency) ExecuteStringWithIO(str string, stdout io.Writer, stderr io.Writer, stdin io.Reader) int {
// 	_, env := ed.Resolve()
// 	//cdd := NewCircularDependencyDetector()
// 	expandedArgs, err := ed.expandAlias(str, env /*, cdd*/)
// 	if err != nil {
// 		fmt.Fprintf(os.Stderr, "%s\n", err)
// 		os.Exit(1)
// 	}
// 	return ed.ExecuteWithIO(expandedArgs, stdout, stderr, stdin)
// }

// ExpandCommand expands command by resolving alias if no alias
// is found it returns the prog variable unmodified in a slice
func (ed *ResolvedDependency) ExpandCommand(args []string) ([]string, error) {
	_, env := ed.Resolve()
	//cdd := NewCircularDependencyDetector()
	return ed.expandCommand(args, env /*, cdd*/)
}

func (ed *ResolvedDependency) expandCommand(
	args []string,
	env map[string]string,
	//cdd *CircularDependencyDetector,
) ([]string, error) {
	if args == nil {
		return nil, nil
	}
	if len(args) < 1 {
		return args, nil
	}

	//Debug.Printf("expandCommand(%s)", strings.Join(args, " "))
	prog := args[0]

	expanded, err := ed.expandAlias(prog, env /*, cdd*/)
	if err != nil {
		return args, err
	}
	expanded = append(expanded, args[1:]...)

	return expanded, nil
}

func (ed *ResolvedDependency) expandAlias(
	arg0 string,
	env map[string]string,
	//cdd *CircularDependencyDetector,
) ([]string, error) {
	//if err := cdd.Push(arg0); err != nil {
	//	return nil, fmt.Errorf("expandAlias(): %w", err)
	//}

	var result []string

	if str, ok := ed.Alias[arg0]; ok {
		expanded, err := shell.Fields(str, func(k string) string {
			if v, ok := env[k]; ok {
				return v
			}
			return fmt.Sprintf("$%s", k)
		})
		if err != nil {
			return nil, fmt.Errorf("ExtractedDependency.ExpandCommand(): %w", err)
		}
		result = expanded
	} else {
		result = append(result, arg0)
	}

	for _, sub := range ed.Sub {
		expanded, err := sub.expandAlias(result[0], env /*, cdd*/)
		if err != nil {
			return nil, err
		}
		result = append(expanded, result[1:]...)
	}

	return result, nil
}

// Resolve returns a slice with bin dirs to be prepended to PATH os var
// and a map with all environment variables to be added.  It resolves all
// of these values recursevily
func (ed *ResolvedDependency) Resolve(ctx *model.ExecContext) {
	//env := make(map[string]string)

	// Sub
	//var subPaths []string
	for _, sub := range ed.Sub {
		//subSubPaths, subEnv := sub.Resolve(ctx)
		sub.Resolve(ctx)

		// // Sub Env Vars
		// for k, v := range subEnv {
		// 	env[k] = parseShellTpl(v, env)
		// }
		// subPaths = append(subPaths, subSubPaths...)
	}

	// Local
	//env = utils.MapMerge(env, ed.resolveLocalEnvVars(env))
	ed.resolveLocalEnvVars(ctx)

	// binDir
	binDir := ed.BinDirOrDefault()

	ctx.Path = append([]string{parseShellTpl(binDir, ctx)}, ctx.Path...)

	//fmt.Printf("#%s: %s\n", &ed.Coord, ed.Triggers.PreRunScript)
	return ed.Triggers.RunPreRun(ctx) // modify env by script
}

// resolveLocalEnvVars returns map with implicit variables and exported ones
// for only this extracted dependecy
func (ed *ResolvedDependency) resolveLocalEnvVars(ctx *model.ExecContextsg) {
	env := make(map[string]string)
	env["DIR"] = ed.Dir // DIR
	env["CURDIR"] = utils.GetCurrentDir()
	env = utils.MapMerge(ctx.Env, env)
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
		ctx.Env[k] = v
	}
}

func calculateImplicitDirEnvironmentVars(ea ResolvedDependency, env map[string]string) map[string]string {
	c := ea.Coord
	m := make(map[string]string)

	nameSpaceVarPrefixes := []string{
		// GITHUB_COM_BAZURTO_GROOVY_V1.2.3
		utils.ToEnvKey(fmt.Sprintf("%s_%s_%s_%s", c.Server, c.Owner, c.Repo, c.Version.Canonical())),
		// GITHUB_COM_BAZURTO_GROOVY
		utils.ToEnvKey(fmt.Sprintf("%s_%s_%s", c.Server, c.Owner, c.Repo)),
		// BAZURTO_GROOVY_V1.2.3
		utils.ToEnvKey(fmt.Sprintf("%s_%s_%s", c.Owner, c.Repo, c.Version.Canonical())),
		// BAZURTO_GROOVY
		utils.ToEnvKey(fmt.Sprintf("%s_%s", c.Owner, c.Repo)),
		// GROOVY_V1.2.3
		utils.ToEnvKey(fmt.Sprintf("%s_%s", c.Repo, c.Version.Canonical())),
		// GROOVY
		utils.ToEnvKey(c.Repo),
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
	//
	f := func(tplVar string) string {
		if v, ok := env[tplVar]; ok {
			return v
		}
		return tplVar
	}

	//
	parsedStr, _ := shell.Expand(tpl, f)
	// if err != nil {
	// 	Warn.Printf("WARNING: %s\n", err)
	// }
	return parsedStr
}
