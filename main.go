// SPDX-FileCopyrightText: 2023 RH America LLC <info@rhamerica.com>
// SPDX-License-Identifier: GPL-3.0-only

package main

import (
	"fmt"
	"os"

	"github.com/bazurto/bz/lib"
)

func main() {
	lib.Debug.Printf("Look for project files: %s", lib.ConfigFileNames)
	// current project
	projectLocation, _ := lib.FindFileUpwards(lib.ConfigFileNames, nil)
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
	ghr := lib.NewGithubResolver()
	engine := lib.NewEngine()
	engine.AddResolver(ghr)

	// Get Context From Config
	rctx, err := engine.ContextFromConfigDir(projectLocation.Root) // does resolving and downloading
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s", err)
		os.Exit(1)
	}

	// rctx has env vars, aliases and all resolved information
	exitCode := rctx.Execute(os.Args[1:])
	os.Exit(exitCode)
}
