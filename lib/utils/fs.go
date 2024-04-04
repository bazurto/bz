// SPDX-FileCopyrightText: 2023 RH America LLC <info@rhamerica.com>
// SPDX-License-Identifier: GPL-3.0-only

package utils

import (
	"os"
	"path/filepath"
)

func FsAbs(f string) string {
	var err error
	f, err = filepath.Abs(f)
	if err != nil {
		return filepath.Clean(f)
	}
	return f
}

func GetCurrentDir() string {
	dir, _ := os.Getwd()
	return FsAbs(dir)
}
