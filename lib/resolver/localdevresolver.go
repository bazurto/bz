// SPDX-FileCopyrightText: 2023 RH America LLC <info@rhamerica.com>
// SPDX-License-Identifier: GPL-3.0-only

package resolver

import (
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

func (o *LocalDevResolver) ResolveCoord(c model.FuzzyCoord) (*model.LockedCoord, error) {
	Debug.Printf("Start LocalDevResolver.ResolveCoord(%s)", c.String())

	if c.URL.Scheme != "file" {
		return nil, nil
	}

	return &model.LockedCoord{
		URL: c.URL,
	}, nil
}

func (o *LocalDevResolver) DownloadResolvedCoord(lc model.LockedCoord) (string, error, bool) {
	if lc.URL.Scheme != "file" {
		return "", nil, false
	}

	return lc.URL.Path, nil, true
}
