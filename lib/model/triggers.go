// SPDX-FileCopyrightText: 2023 RH America LLC <info@rhamerica.com>
// SPDX-License-Identifier: GPL-3.0-only

package model

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"

	"github.com/bazurto/bz/lib/exec"
	"github.com/bazurto/bz/lib/model"
	"github.com/bazurto/bz/lib/utils"
	"github.com/hashicorp/hcl/v2"
)

type Triggers struct {
	InstallScript string   `ion:"installScript" json:"installScript,omitempty" hcl:"installScript,optional"`
	PreRunScript  string   `ion:"preRunScript" json:"preRunScript,omitempty" hcl:"preRunScript,optional"`
	Remain        hcl.Body `ion:"-" json:"-" hcl:",remain"`
}

func (o *Triggers) RunInstallScript(lcc *LockedConfigContent) error {
	if o == nil {
		return nil
	}

	if o.InstallScript == "" {
		return nil
	}

	if exports, err := utils.RunScriptCode(o.InstallScript); err != nil {
		return err
	} else {
		fmt.Println(exports)
	}

	return nil
}

// RunPreRun modifies the context.  It modifies the path and env variables and returns them
// func (o *Triggers) RunPreRun(d *ResolvedDependency, path []string, env map[string]string) ([]string, map[string]string) {
func (o *Triggers) RunPreRun(ctx *model.ExecContext) {
	if o == nil {
		return
	}
	if o.PreRunScript == "" {
		return
	}

	b, err := json.Marshal(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error marshaling jsons tring on PreRunScript(%s):%s\n", o.PreRunScript, err)
		os.Exit(1)
	}
	stdout := bytes.NewBuffer(nil)
	stderr := bytes.NewBuffer(nil)
	stdin := bytes.NewBuffer(b)
	//fmt.Println("Executing " + o.PreRunScript + "...Start...")
	exec.ExecCommand()
	errno := d.ExecuteStringWithIO(o.PreRunScript, stdout, stderr, stdin)
	if errno != 0 {
		fmt.Fprintf(os.Stderr, "Error running PreRunScript(%s): %s\n", o.PreRunScript, stderr.String())
		os.Exit(1)
	}
	jsonStr := stdout.String()
	fmt.Println("Execuring " + o.PreRunScript + "...end")

	err = utils.JsonDecode(jsonStr, &preRunCtx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error running unmarshaling response from PreRunScript(%s):%s:\n%s\n", o.PreRunScript, err, stdout.String())
		os.Exit(1)
	}
}
