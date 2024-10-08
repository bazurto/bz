// SPDX-FileCopyrightText: 2023 RH America LLC <info@rhamerica.com>
// SPDX-License-Identifier: GPL-3.0-only

package model

import (
	"fmt"
	"os"
	"path/filepath"

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

// func (ed *ResolvedDependency) ResolveAlias(alias string) []string {
// 	var result []string
// 	if str, ok := ed.Alias[alias]; ok {
// 		expanded, err := shell.Fields(str, func(k string) string {
// 			if v, ok := ctx.Env()[k]; ok {
// 				return v
// 			}
// 			return fmt.Sprintf("$%s", k)
// 		})
// 		if err != nil {
// 			return nil, fmt.Errorf("error expanding alias %s: %w", alias, err)
// 		}
// 		result = expanded
// 	} else {
// 		result = []string{alias}
// 	}
// 	return result
// }

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

// // ExpandCommand expands command by resolving alias if no alias
// // is found it returns the prog variable unmodified in a slice
// func (ed *ResolvedDependency) ExpandCommand(args []string, ec ExecContext) ([]string, error) {
// 	return ed.expandCommand(args, ec.Env())
// }

// func (ed *ResolvedDependency) expandCommand(
// 	args []string,
// 	env map[string]string,
// ) ([]string, error) {
// 	if args == nil {
// 		return nil, nil
// 	}
// 	if len(args) < 1 {
// 		return args, nil
// 	}

// 	//Debug.Printf("expandCommand(%s)", strings.Join(args, " "))
// 	prog := args[0]

// 	expanded, err := ed.expandAlias(prog, env /*, cdd*/)
// 	if err != nil {
// 		return args, err
// 	}
// 	expanded = append(expanded, args[1:]...)

// 	return expanded, nil
// }

// func (ed *ResolvedDependency) expandAlias(
// 	arg0 string,
// 	env map[string]string,
// 	//cdd *CircularDependencyDetector,
// ) ([]string, error) {
// 	//if err := cdd.Push(arg0); err != nil {
// 	//	return nil, fmt.Errorf("expandAlias(): %w", err)
// 	//}

// 	var result []string

// 	if str, ok := ed.Alias[arg0]; ok {
// 		expanded, err := shell.Fields(str, func(k string) string {
// 			if v, ok := env[k]; ok {
// 				return v
// 			}
// 			return fmt.Sprintf("$%s", k)
// 		})
// 		if err != nil {
// 			return nil, fmt.Errorf("ExtractedDependency.ExpandCommand(): %w", err)
// 		}
// 		result = expanded
// 	} else {
// 		result = append(result, arg0)
// 	}

// 	for _, sub := range ed.Sub {
// 		expanded, err := sub.expandAlias(result[0], env /*, cdd*/)
// 		if err != nil {
// 			return nil, err
// 		}
// 		result = append(expanded, result[1:]...)
// 	}

// 	return result, nil
// }

// Resolve returns a slice with bin dirs to be prepended to PATH os var
// and a map with all environment variables to be added.  It resolves all
// of these values recursevily
func (ed *ResolvedDependency) Resolve() *ExecContext {
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
func (ed *ResolvedDependency) resolveLocalEnvVars(subCtx []ExecContext) *ExecContext {
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
	f := func(tplVar string) string {
		if v, ok := env[tplVar]; ok {
			return v
		}
		return tplVar
	}
	parsedStr, _ := shell.Expand(tpl, f)
	return parsedStr
}
