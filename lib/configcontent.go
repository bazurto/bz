// SPDX-FileCopyrightText: 2023 RH America LLC <info@rhamerica.com>
// SPDX-License-Identifier: GPL-3.0-only

package lib

import (
	"fmt"
	"strings"

	"github.com/hashicorp/hcl/v2"
)

type LockedConfigContent struct {
	BinDir   string            `ion:"binDir" json:"binDir,omitempty"`
	Deps     []*LockedCoord    `ion:"deps" json:"deps,omitempty"`
	Export   map[string]string `ion:"env" json:"env,omitempty"`
	Alias    map[string]string `ion:"alias" json:"alias,omitempty"`
	Triggers *Triggers         `ion:"triggers" json:"triggers,omitempty"`
}

type FuzzyConfigContent struct {
	BinDir   string            `ion:"binDir" json:"binDir" hcl:"binDir,optional"`
	Deps     []string          `ion:"deps" json:"deps" hcl:"deps,optional"`
	Export   map[string]string `ion:"env" json:"env" hcl:"env,optional"`
	Alias    map[string]string `ion:"alias" json:"alias" hcl:"alias,optional"`
	Triggers *Triggers         `ion:"triggers" json:"triggers,omitempty" hcl:"triggers,optional"`
	Remain   hcl.Body          `hcl:",remain"`
}

type Triggers struct {
	InstallScript *string `ion:"installScript" json:"installScript,omitempty" hcl:"installScript,optional"`
	PreRunScript  *string `ion:"preRunScript" json:"preRunScript,omitempty" hcl:"preRunScript,optional"`
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

func (o *Triggers) RunInstallScript(lcc *LockedConfigContent) error {
	if o == nil {
		return nil
	}

	if o.InstallScript == nil {
		return nil
	}

	if err := RunScriptCode(*o.InstallScript); err != nil {
		return err
	}

	return nil
}

func (o *Triggers) RunPreRun(d *ResolvedDependency, path []string, env map[string]string) ([]string, map[string]string) {
	if o == nil {
		return path, env
	}

	if o.PreRunScript == nil {
		return path, env
	}

	exports, _ := RunScriptCode(*o.PreRunScript, path, env)
	//exports["binDir"]
	//exports["env"]

	return path, env
}
