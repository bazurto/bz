// SPDX-FileCopyrightText: 2023 RH America LLC <info@rhamerica.com>
// SPDX-License-Identifier: GPL-3.0-only

package lib

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
)

var (
	Debug *log.Logger
	Warn  *log.Logger
	Info  *log.Logger
)

func init() {

	// DEBUG
	debugEnv := os.Getenv("DEBUG")
	if debugEnv != "" && debugEnv != "0" && !strings.EqualFold(debugEnv, "false") {
		Debug = log.New(os.Stderr, "[D]", log.LstdFlags)
	} else {
		Debug = log.New(io.Discard, "", 0)
	}
	Warn = log.New(os.Stderr, "[W]", log.LstdFlags)
	Info = log.New(os.Stderr, "[I]", log.LstdFlags)
}

// PathFound struct returned by FindFileUpwards
type PathFound struct {
	Root string
	File string
}

func (p *PathFound) String() string {
	return fmt.Sprintf("PathFound{Root:%s; File:%s}", p.Root, p.File)
}

// FindFileUpwards Find a file within a specified directory.
// If not found it walks up the directory tree until the file
// is found or no more parent directories exists.
// subPathToFind Path to be found as an array
func FindFileUpwards(
	subPathToFind []string,
	givenRoot *string,
) (*PathFound, error) {

	dir, _ := os.Getwd()
	if givenRoot != nil {
		dir = *givenRoot
	}

	Debug.Printf("| FindFilesUpwards() look in %s", dir)
	for dir != "/" && dir != "" && dir != "." {
		for _, pathToFind := range subPathToFind {
			foundPath := filepath.Join(dir, pathToFind)
			Debug.Printf(" | try %s", foundPath)
			if _, err := os.Stat(foundPath); !os.IsNotExist(err) {
				return &PathFound{File: foundPath, Root: dir}, nil
			}
		}

		tmpDir := filepath.Dir(dir)
		if tmpDir == dir {
			break // we arrived at root
		}
		dir = tmpDir
	}

	return nil, fmt.Errorf("`%s` file not found", strings.Join(subPathToFind, " "))
}
