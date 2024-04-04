// SPDX-FileCopyrightText: 2023 RH America LLC <info@rhamerica.com>
// SPDX-License-Identifier: GPL-3.0-only

package model

import (
	"fmt"
	"strings"
)

type FuzzyCoord struct {
	OriginalString string // original string
	Server         string // github.com | local.local
	Owner          string // rhamerica
	Repo           string // myrepo
	Version        string // no v
}

func NewCoordFromStr(depStr string) (*FuzzyCoord, error) {

	// github.com/owner/repo@1.2.3 -> github.com,owner, repo-v1.2.3
	server, owner, repoVersion := splitPattern3(depStr, "/")

	if server == "" || owner == "" || repoVersion == "" {
		return nil, fmt.Errorf("unable to parse dependency '%s': invalid format", depStr)
	}

	if server == "" {
		return nil, fmt.Errorf("unable to parse dependency '%s': server name is required", depStr)
	}

	if owner == "" {
		return nil, fmt.Errorf("unable to parse dependency '%s': owner name is required", depStr)
	}

	// repo@1.2.3
	// my-repo-no-version
	// myrepo
	repo, version := splitPattern2(repoVersion, "@")

	// remove v
	if len(version) > 0 && version[0] == 'v' {
		version = version[1:]
	}

	if repo == "" {
		return nil, fmt.Errorf("unable to parse dependency '%s': repo name is required", depStr)
	}

	return &FuzzyCoord{
		OriginalString: depStr,
		Server:         server,
		Owner:          owner,
		Repo:           repo,
		Version:        version,
	}, nil
}

func (d *FuzzyCoord) CanonicalNameNoVersion() string {
	return fmt.Sprintf("%s/%s/%s", d.Server, d.Owner, d.Repo)
}

func (d *FuzzyCoord) String() string {
	return fmt.Sprintf("%s/%s/%s-%s", d.Server, d.Owner, d.Repo, d.Version)
}

func (o *FuzzyCoord) isCoord() {
}

func splitPattern2(s, glue string) (string, string) {
	parts := strings.Split(s, glue)
	l := len(parts)
	if l < 2 {
		return s, ""
	}
	return parts[0], parts[1]
}

func splitPattern3(s, glue string) (string, string, string) {
	parts := strings.Split(s, glue)
	l := len(parts)
	if l < 2 {
		return s, "", ""
	} else if l < 3 {
		return parts[0], parts[1], ""
	}
	return parts[0], parts[1], parts[2]
}
