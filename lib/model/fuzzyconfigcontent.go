// SPDX-FileCopyrightText: 2023 RH America LLC <info@rhamerica.com>
// SPDX-License-Identifier: GPL-3.0-only

package model

import (
	"fmt"
	"strings"

	"github.com/bazurto/bz/lib/utils"
	"github.com/hashicorp/hcl/v2"
)


type FuzzyConfigContent struct {
	BinDir   string            `ion:"binDir" json:"binDir" hcl:"binDir,optional"`
	Deps     []string          `ion:"deps" json:"deps" hcl:"deps,optional"`
	Export   map[string]string `ion:"env" json:"env" hcl:"env,optional"`
	Alias    map[string]string `ion:"alias" json:"alias" hcl:"alias,optional"`
	Triggers *Triggers         `ion:"triggers" json:"triggers,omitempty" hcl:"triggers,block"`
	Remain   hcl.Body          `ion:"-" json:"-" hcl:",remain"`
}

func FuzzyConfigContentFromFile(f string) (*FuzzyConfigContent, error) {
	var err error
	cfg := FuzzyConfigContent{}
	if utils.IsIonFile(f) {
		err = utils.IonLoad(f, &cfg)
	} else {
		err = utils.HclLoad(f, &cfg)
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