// SPDX-FileCopyrightText: 2023 RH America LLC <info@rhamerica.com>
// SPDX-License-Identifier: GPL-3.0-only

package model

import (
	"fmt"
)

type LockedCoord struct {
	Server  string  `ion:"server" json:"server"`
	Owner   string  `ion:"owner" json:"owner"`
	Repo    string  `ion:"repo" json:"repo"`
	Version Version `ion:"version" json:"version"` // no v
}

func NewLockedCoordLocalBlank() LockedCoord {
	c := LockedCoord{
		Server:  "localhost",
		Owner:   "local",
		Repo:    "local",
		Version: NewVersion("0.0.0"),
	}
	return c
}

func (o *LockedCoord) isCoord() {
}

func (d *LockedCoord) CanonicalNameNoVersion() string {
	return fmt.Sprintf("%s/%s/%s", d.Server, d.Owner, d.Repo)
}

func (o *LockedCoord) String() string {
	return fmt.Sprintf(
		"%s/%s/%s@%s",
		o.Server,
		o.Owner,
		o.Repo,
		o.Version.Canonical(),
	)
}
