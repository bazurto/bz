// SPDX-FileCopyrightText: 2023 RH America LLC <info@rhamerica.com>
// SPDX-License-Identifier: GPL-3.0-only

package resolver

import (
	"strings"

	"github.com/bazurto/bz/lib/model"
	"github.com/bazurto/bz/lib/utils"
)

var (
	Debug, Warn, Info = utils.Loggers()
)

type LocalDevResolver struct {
}

func NewLocalDevResolver(appCtx *model.AppContext) *LocalDevResolver {
	return &LocalDevResolver{}
}

func (o *LocalDevResolver) String() string {
	return "LocalDevResolver{}"
}

func (o *LocalDevResolver) ResolveCoord(c *model.FuzzyCoord) (*model.LockedCoord, error) {
	Debug.Printf("Start LocalDevResolver.ResolveCoord(%s)", c)

	if c.Server != "local.local" && c.Server != "local" {
		return nil, nil
	}

	depString := strings.TrimPrefix(c.OriginalString, "local.local")
	depString = strings.TrimPrefix(depString, "local")

	return &model.LockedCoord{
		Server:  c.Server,
		Owner:   c.Owner,
		Repo:    depString,
		Version: model.NewVersion(c.Version),
	}, nil
}

func (o *LocalDevResolver) DownloadResolvedCoord(lc *model.LockedCoord) (string, error, bool) {
	if lc.Server != "local.local" && lc.Server != "local" {
		return "", nil, false
	}

	return lc.Repo, nil, true
}
