// SPDX-FileCopyrightText: 2023 RH America LLC <info@rhamerica.com>
// SPDX-License-Identifier: GPL-3.0-only

package model

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/bazurto/bz/lib/utils"
)

type AppContext struct {
	AppName            string
	LockFileName       string
	HomeDir            string
	UserDir            string
	UserConfigFileName string
	UserCacheDirName   string
	ConfigFileNames    []string
	UserConfig         UserConfig
}

func NewDefaultAppContext() *AppContext {
	appName := "bz"
	homeDir, _ := os.UserHomeDir()
	userDir := filepath.Join(homeDir, fmt.Sprintf(".%s", appName))
	userConfigFileName := filepath.Join(userDir, "config")

	// load user config if it exists
	var userConfig *UserConfig
	if utils.FileExists(userConfigFileName) {
		var err error
		userConfig, err = NewUserConfigFromFile(userConfigFileName)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading user config (%s): %s\n", userConfigFileName, err)
			os.Exit(1)
		}
	} else {
		userConfig = &UserConfig{}
	}

	return &AppContext{
		AppName:            appName,
		LockFileName:       fmt.Sprintf(".%s.lock", appName),
		HomeDir:            homeDir,
		UserDir:            userDir,
		UserConfigFileName: filepath.Join(userDir, "config"),
		UserCacheDirName:   filepath.Join(userDir, "cache"),
		ConfigFileNames: []string{
			fmt.Sprintf(".%s.hcl", appName),
			fmt.Sprintf(".%s.json", appName),
			fmt.Sprintf(".%s", appName),
		},
		UserConfig: *userConfig,
	}
}
