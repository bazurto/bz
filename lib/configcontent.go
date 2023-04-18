// SPDX-FileCopyrightText: 2023 RH America LLC <info@rhamerica.com>
// SPDX-License-Identifier: GPL-3.0-only

package lib

import (
	"fmt"
	"strings"

	"github.com/hashicorp/hcl/v2"
)

type LockedConfigContent struct {
	BinDir string            `ion:"binDir" json:"binDir,omitempty"`
	Deps   []*LockedCoord    `ion:"deps" json:"deps,omitempty"`
	Export map[string]string `ion:"env" json:"env,omitempty"`
	Alias  map[string]string `ion:"alias" json:"alias,omitempty"`
}

type FuzzyConfigContent struct {
	BinDir string            `ion:"binDir" json:"binDir" hcl:"binDir,optional"`
	Deps   []string          `ion:"deps" json:"deps" hcl:"deps,optional"`
	Export map[string]string `ion:"env" json:"env" hcl:"env,optional"`
	Alias  map[string]string `ion:"alias" json:"alias" hcl:"alias,optional"`
	Remain hcl.Body          `hcl:",remain"`
}

func LockedConfigContentFromFile(f string) (*LockedConfigContent, error) {
	var err error
	cfg := LockedConfigContent{}
	err = jsonLoad(f, &cfg)
	return &cfg, err
}

func FuzzyConfigContentFromFile(f string) (*FuzzyConfigContent, error) {
	var err error
	cfg := FuzzyConfigContent{}
	if isIonFile(f) {
		err = ionLoad(f, &cfg)
	} else {
		err = hclLoad(f, &cfg)
	}
	return &cfg, err
}

func (c *FuzzyConfigContent) String() string {
	var exports []string
	for k, v := range c.Export {
		exports = append(exports, fmt.Sprintf("%s=%s", k, v))
	}
	return fmt.Sprintf(
		"Config{BinDir:%s; Deps:%s; Export:%s;}",
		c.BinDir,
		strings.Join(c.Deps, ";"),
		strings.Join(exports, ";"),
	)
}
