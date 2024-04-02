// SPDX-FileCopyrightText: 2023 RH America LLC <info@rhamerica.com>
// SPDX-License-Identifier: GPL-3.0-only

package exec

import (
	"fmt"
	"io"
	"os"
	"strings"

	oexec "os/exec"

	"mvdan.cc/sh/shell"
)

type ExecContext interface {
	ResolveAlias(args []string) []string
	Env() map[string]string
}

func ExecCommandStr(
	ctx ExecContext,
	args string,
	stdout io.Writer,
	stderr io.Writer,
	stdin io.Reader,
) int {
	expanded, err := shell.Fields(args, func(k string) string {
		if v, ok := ctx.Env()[k]; ok {
			return v
		}
		return fmt.Sprintf("$%s", k)
	})

	if err != nil {
		return -1
	}
	return ExecCommand(ctx, expanded, stdout, stderr, stdin)
}

func ExecCommand(
	ctx ExecContext,
	args []string,
	stdout io.Writer,
	stderr io.Writer,
	stdin io.Reader,
) int {
	// Args
	args = ctx.ResolveAlias(args)

	// Set environment
	for k, v := range ctx.Env() {
		os.Setenv(k, v)
	}

	// fmt.Println("----------------------------------------------------")
	// fmt.Printf("PATH=%s\n", os.Getenv("PATH"))
	// fmt.Println("----------------------------------------------------")

	prog := args[0]
	progArgs := args[1:]
	cmd := oexec.Command(prog, progArgs...)
	cmd.Stdout = stdout
	cmd.Stdin = stdin
	cmd.Stderr = stderr
	err := cmd.Run()
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
