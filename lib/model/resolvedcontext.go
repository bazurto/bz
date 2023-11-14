// SPDX-FileCopyrightText: 2023 RH America LLC <info@rhamerica.com>
// SPDX-License-Identifier: GPL-3.0-only

package model

/*
import (
	"fmt"
	"os"
	"strings"
)

type ResolvedContext struct {
	newPath            []string
	newEnv             map[string]string
	ResolvedDependency ResolvedDependency
}

func (o *ResolvedContext) Execute(args []string) (int, error) {

	// env vars
	//newPath, newEnv := ed.Resolve()

	//
	path := os.Getenv("PATH")
	pathParts := strings.Split(path, string([]rune{os.PathListSeparator}))
	newPath := append(o.newPath, pathParts...)
	os.Setenv("PATH", strings.Join(newPath, string([]rune{os.PathListSeparator})))

	// Args
	args, err := ed.ExpandCommand(os.Args[1:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}

	if len(os.Args) > 1 {
		os.Exit(executeCommand(args, newEnv))
	}

	return 0, nil
}

*/
