// SPDX-FileCopyrightText: 2023 RH America LLC <info@rhamerica.com>
// SPDX-License-Identifier: GPL-3.0-only

package model

import (
	"fmt"

	"github.com/bazurto/bz/lib/utils"
)

type LockedConfigContent struct {
	BinDir   string            `ion:"binDir" json:"binDir,omitempty"`
	Deps     []*LockedCoord    `ion:"deps" json:"deps,omitempty"`
	Export   map[string]string `ion:"env" json:"env,omitempty"`
	Alias    map[string]string `ion:"alias" json:"alias,omitempty"`
	Triggers Triggers          `ion:"triggers" json:"triggers,omitempty"`
}

func LockedConfigContentFromFile(f string) (*LockedConfigContent, error) {
	var err error
	cfg := LockedConfigContent{}
	err = utils.JsonLoad(f, &cfg)
	fmt.Printf("error reading %s: %s", f, err)
	return &cfg, err
}