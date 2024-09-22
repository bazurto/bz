// SPDX-FileCopyrightText: 2023 RH America LLC <info@rhamerica.com>
// SPDX-License-Identifier: GPL-3.0-only

package main

import (
	"fmt"
	"os"

	"github.com/bazurto/bz/lib"
	"github.com/bazurto/bz/lib/model"
	"github.com/bazurto/bz/lib/resolver"
)

var (
	buildInfo string
)

func main() {
	os.Setenv("BZ_INFO", buildInfo)

	appCtx := model.NewDefaultAppContext()

	lib.Debug.Printf("Look for project files: %s", appCtx.ConfigFileNames)
	// current project
	projectLocation, _ := lib.FindFileUpwards(appCtx.ConfigFileNames, nil)
	if projectLocation == nil {
		wd, _ := os.Getwd()
		projectLocation = &lib.PathFound{
			File: "",
			Root: wd,
		}
		lib.Debug.Printf("project configuration Not Found.  setting to: %s", projectLocation)
	} else {
		lib.Debug.Printf("found project location: %s", projectLocation)
	}

	// Resolvers
	ghr := resolver.NewGithubResolver(appCtx)
	local := resolver.NewLocalDevResolver(appCtx)
	engine := lib.NewEngine(*appCtx)
	engine.AddResolver(ghr)
	engine.AddResolver(local)

	// Get Context From Config
	rdep, err := engine.ContextFromConfigDir(projectLocation.Root) // does resolving and downloading
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s", err)
		os.Exit(1)
	}

	// rctx has env vars, aliases and all resolved information
	exitCode := engine.Execute(rdep, os.Args[1:])
	//exitCode = rdep.Execute(os.Args[1:])
	os.Exit(exitCode)
}
