// SPDX-FileCopyrightText: 2023 RH America LLC <info@rhamerica.com>
// SPDX-License-Identifier: GPL-3.0-only

package lib

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
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

	if exports, err := RunScriptCode(*o.InstallScript); err != nil {
		return err
	} else {
		fmt.Println(exports)
	}

	return nil
}

// RunPreRun modifies the context.  It modifies the path and env variables and returns them
func (o *Triggers) RunPreRun(d *ResolvedDependency, path []string, env map[string]string) ([]string, map[string]string) {
	preRunCtx := PreRunCtx{Env: env, Path: path}
	if o != nil && o.PreRunScript != nil && *o.PreRunScript != "" {
		b, err := json.Marshal(preRunCtx)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error marshaling jsons tring on PreRunScript(%s):%s\n", *o.PreRunScript, err)
			os.Exit(1)
		}
		stdout := bytes.NewBuffer(nil)
		stderr := bytes.NewBuffer(nil)
		stdin := bytes.NewBuffer(b)
		errno := d.ExecuteStringWithIO(*o.PreRunScript, stdout, stderr, stdin)
		if errno != 0 {
			fmt.Fprintf(os.Stderr, "Error running PreRunScript(%s): %s\n", *o.PreRunScript, stderr.String())
			os.Exit(1)
		}
		jsonStr := stdout.String()

		err = jsonDecode(jsonStr, &preRunCtx)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error running unmarshaling response from PreRunScript(%s):%s:\n%s\n", *o.PreRunScript, err, stdout.String())
			os.Exit(1)
		}
	}
	return preRunCtx.Path, preRunCtx.Env
}

type PreRunCtx struct {
	Path []string          `json:"path"`
	Env  map[string]string `json:"env"`
}
