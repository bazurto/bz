// SPDX-FileCopyrightText: 2023 RH America LLC <info@rhamerica.com>
// SPDX-License-Identifier: GPL-3.0-only

package utils

import (
	"io"
	"log"
	"os"
	"strings"
)

func Loggers() (Debug *log.Logger, Warn *log.Logger, Info *log.Logger) {
	// DEBUG
	debugEnv := os.Getenv("DEBUG")
	if debugEnv != "" && debugEnv != "0" && !strings.EqualFold(debugEnv, "false") {
		Debug = log.New(os.Stderr, "[D]", log.LstdFlags)
	} else {
		Debug = log.New(io.Discard, "", 0)
	}
	Warn = log.New(os.Stderr, "[W]", log.LstdFlags)
	Info = log.New(os.Stderr, "[I]", log.LstdFlags)
	return Debug, Warn, Info
}
