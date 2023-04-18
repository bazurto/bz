// SPDX-FileCopyrightText: 2023 RH America LLC <info@rhamerica.com>
// SPDX-License-Identifier: GPL-3.0-only

package lib

type Resolver interface {
	// Resolves a fuzzy coord to a hard resolved coord
	ResolveCoord(c *FuzzyCoord) (*LockedCoord, error)
	// Download coord pointed by c into file
	DownloadResolvedCoord(c *LockedCoord, dir string) (string, error)
}
