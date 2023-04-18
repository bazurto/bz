// SPDX-FileCopyrightText: 2023 RH America LLC <info@rhamerica.com>
// SPDX-License-Identifier: GPL-3.0-only

package lib

import (
	"os"
	"path/filepath"
)

func fsAbs(f string) string {
	var err error
	f, err = filepath.Abs(f)
	if err != nil {
		return filepath.Clean(f)
	}
	return f
}

func getCurrentDir() string {
	dir, _ := os.Getwd()
	return fsAbs(dir)
}
