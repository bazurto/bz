// SPDX-FileCopyrightText: 2023 RH America LLC <info@rhamerica.com>
// SPDX-License-Identifier: GPL-3.0-only

package model

type PreRunCtx struct {
	Path []string          `json:"path"`
	Env  map[string]string `json:"env"`
}
