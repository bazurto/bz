// SPDX-FileCopyrightText: 2023 RH America LLC <info@rhamerica.com>
// SPDX-License-Identifier: GPL-3.0-only

package resolver

import (
	"fmt"

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
	return nil, nil
}

func (o *LocalDevResolver) DownloadResolvedCoord(rc *model.LockedCoord, dir string) (string, error) {
	return "", fmt.Errorf("Not implemented")
}