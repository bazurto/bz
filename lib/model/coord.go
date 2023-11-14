// SPDX-FileCopyrightText: 2023 RH America LLC <info@rhamerica.com>
// SPDX-License-Identifier: GPL-3.0-only

package model

type Coord interface {
	isCoord() // implements this to signal it is an implementation of coord
	CanonicalNameNoVersion() string
}



