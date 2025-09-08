// SPDX-FileCopyrightText: 2023 RH America LLC <info@rhamerica.com>
// SPDX-License-Identifier: GPL-3.0-only

package model

import (
	"fmt"
	"net/url"
	"strings"
)

type FuzzyCoord struct {
	OriginalString string
	URL            url.URL
}

// NewCoordFromStr parses dependency strings from .bz file
// possible strings:
//   - github.com/owner/repo#1.2.3	-> https://github.com/owner/repo#1.2.3
//   - https://github.com/owner/repo#1.2.3	-> https://github.com/owner/repo#1.2.3
//   - file:///home/user/workspace/mypackage
func NewCoordFromStr(depStrArg string) (FuzzyCoord, error) {
	var result FuzzyCoord
	depStr := depStrArg
	depStrLower := strings.ToLower(depStrArg)
	if !strings.HasPrefix(depStrLower, "https://") &&
		!strings.HasPrefix(depStrLower, "http://") &&
		!strings.HasPrefix(depStrLower, "file://") {
		depStr = fmt.Sprintf("https://%s", depStr)
	}
	u, err := url.Parse(depStr)
	if err != nil {
		return result, err
	}

	result.OriginalString = depStrArg
	result.URL = *u
	return result, nil
}

//	func (d *FuzzyCoord) CanonicalNameNoVersion() string {
//		return fmt.Sprintf("%s/%s/%s", d.Server, d.Owner, d.Repo)
//	}
func (o *FuzzyCoord) String() string {
	return o.URL.String()
}

//
// func (o *FuzzyCoord) isCoord() {
// }
//
// func splitPattern2(s, glue string) (string, string) {
// 	parts := strings.Split(s, glue)
// 	l := len(parts)
// 	if l < 2 {
// 		return s, ""
// 	}
// 	return parts[0], parts[1]
// }
//
// func splitPattern3(s, glue string) (string, string, string) {
// 	parts := strings.Split(s, glue)
// 	l := len(parts)
// 	if l < 2 {
// 		return s, "", ""
// 	} else if l < 3 {
// 		return parts[0], parts[1], ""
// 	}
// 	return parts[0], parts[1], parts[2]
// }
