// SPDX-FileCopyrightText: 2023 RH America LLC <info@rhamerica.com>
// SPDX-License-Identifier: GPL-3.0-only

package exec

import (
	"fmt"
	"os"
	"strings"

	oexec "os/exec"

	"github.com/bazurto/bz/lib/model"
)

type ExecContext interface {
	Env()
}

func ExecCommand(ctx ExecContext) int {
	// Set the path
	//path := os.Getenv("PATH") // get original path
	//pathParts := strings.Split(path, string([]rune{os.PathListSeparator}))
	//newPath = append(newPath, pathParts...)
	//newEnv["PATH"] = strings.Join(newPath, string([]rune{os.PathListSeparator}))

	// Args
	args, err := ed.ExpandCommand(args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}

	//executeCommand(args, newEnv)
	//debug.Printf("env: \n%s\n", mapJoin(env, "=", "\n"))
	//Debug.Printf("command: %s", strings.Join(args, " "))

	// Set environment
	for k, v := range newEnv {
		os.Setenv(k, v)
	}

	prog := args[0]
	progArgs := args[1:]
	cmd := oexec.Command(prog, progArgs...)
	cmd.Stdout = os.Stdout
	cmd.Stdin = os.Stdin
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	if err != nil {
		if exitError, ok := err.(*oexec.ExitError); ok {
			return exitError.ExitCode()
		} else {
			fmt.Fprintf(os.Stderr, "`%s`: %s\n", strings.Join(args, " "), err)
			return cmd.ProcessState.ExitCode()
		}
	}
	return 0
}
