// SPDX-FileCopyrightText: 2023 RH America LLC <info@rhamerica.com>
// SPDX-License-Identifier: GPL-3.0-only

package lib

import (
	"fmt"
	"runtime"

	"github.com/Masterminds/semver"
)

type Resolver interface {
	// Resolves a fuzzy coord to a hard resolved coord
	ResolveCoord(c *FuzzyCoord) (*LockedCoord, error)
	// Download coord pointed by c into file
	DownloadResolvedCoord(c *LockedCoord, dir string) (string, error)
}

func possibleAssetNames(c *LockedCoord) []BzAsset {
	osArch := fmt.Sprintf("%s-%s", runtime.GOOS, runtime.GOARCH)

	extensions := []string{"zip", "tgz", "tar.gz"} // possible extensions

	var res []BzAsset
	for _, ext := range extensions {
		res = append(res,
			BzAsset{Canonical: fmt.Sprintf("%s-%s-v%s", c.Repo, osArch, c.Version.Canonical()), Ext: ext}, // openjdk-linux-amd64-v1.2.3.zip
			BzAsset{Canonical: fmt.Sprintf("%s-v%s", c.Repo, c.Version.Canonical()), Ext: ext},            // openjdk-v1.2.3.zip
			BzAsset{Canonical: c.Repo, Ext: ext},                                                          // openjdk.zip
		)
	}
	return res
}

type BzAsset struct {
	Ext       string // zip
	Canonical string // project-name-linux-amd64-v1.2.3
}

func (a *BzAsset) NameWithExt() string {
	return fmt.Sprintf("%s.%s", a.Canonical, a.Ext)
}

type BzAssetArrHelper []BzAsset

func (o BzAssetArrHelper) CollectNames() []string {
	var names []string
	for _, i := range o {
		names = append(names, i.NameWithExt())
	}
	return names
}

func versionCompare(a, b string) int {
	v1, err := semver.NewVersion(a)
	if err != nil {
		Warn.Printf("invalid semver `%s`", a)
		return 0
	}
	v2, err := semver.NewVersion(b)
	if err != nil {
		Warn.Printf("invalid semver `%s`", b)
		return 0
	}

	return v1.Compare(v2)
}
