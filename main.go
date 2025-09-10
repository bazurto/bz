// SPDX-FileCopyrightText: 2023 RH America LLC <info@rhamerica.com>
// SPDX-License-Identifier: GPL-3.0-only

package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/bazurto/bz/lib"
	"github.com/bazurto/bz/lib/model"
	"github.com/bazurto/bz/lib/resolver"
)

var (
	buildInfo string
)

func main() {
	os.Setenv("BZ_INFO", buildInfo)

	// Basic flag parsing (only for global flags before dependency resolution)
	showHelp := false
	showVersion := false
	debugFlag := false

	// We use a custom FlagSet to allow passing through unknown flags to target command.
	fs := flag.NewFlagSet("bz", flag.ContinueOnError)
	fs.BoolVar(&showHelp, "help", false, "Show help")
	fs.BoolVar(&showHelp, "h", false, "Show help (shorthand)")
	fs.BoolVar(&showVersion, "version", false, "Show version/build info")
	fs.BoolVar(&debugFlag, "debug", false, "Enable debug logging (sets DEBUG=1)")
	fs.SetOutput(os.Stderr)

	// Parse only until first non-flag (stop on '--')
	parseArgs := os.Args[1:]
	_ = fs.Parse(parseArgs)

	if debugFlag {
		os.Setenv("DEBUG", "1")
		lib.ReinitLoggers()
	}

	if showVersion {
		fmt.Fprintf(os.Stdout, "bz %s\n", buildInfo)
		return
	}

	if showHelp || (len(os.Args) == 1) {
		printHelp()
		return
	}

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

	// Generate Execution Context from root directory
	execCtx, err := engine.ContextFromConfigDir(projectLocation.Root) // resolves and downloads
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}

	// Remaining arguments after flag parsing
	remainingArgs := fs.Args()
	if len(remainingArgs) == 0 { // e.g. only flags like --debug passed
		printHelp()
		return
	}

	exitCode := engine.Execute(execCtx, remainingArgs)
	os.Exit(exitCode)
}

func printHelp() {
	name := filepath.Base(os.Args[0])
	fmt.Fprintf(os.Stderr, "Usage: %s [global flags] <command> [args...]\n", name)
	fmt.Fprintf(os.Stderr, "\nGlobal Flags:\n")
	fmt.Fprintf(os.Stderr, "  --help, -h       Show this help\n")
	fmt.Fprintf(os.Stderr, "  --version        Show version/build info\n")
	fmt.Fprintf(os.Stderr, "  --debug          Enable debug logging (env DEBUG=1)\n")
	fmt.Fprintf(os.Stderr, "\nExamples:\n")
	fmt.Fprintf(os.Stderr, "  %s --version\n", name)
	fmt.Fprintf(os.Stderr, "  %s python --version\n", name)
	fmt.Fprintf(os.Stderr, "  %s bash\n", name)
	fmt.Fprintf(os.Stderr, "\nSet DEBUG=1 (or use --debug) to see resolver and execution details.\n")
}
