// SPDX-FileCopyrightText: 2023 RH America LLC <info@rhamerica.com>
// SPDX-License-Identifier: GPL-3.0-only

package model

import (
	"fmt"
	"net/url"
	"strings"
)

/* {
"Scheme":"https",
"Opaque":"",
"User":null,
"Host":"github.com",
"Path":"/owner/repo@1.2.3",
"RawPath":"",
"OmitHost":false,"ForceQuery":false,"RawQuery":"","Fragment":"","RawFragment":""}
*/

type LockedCoord struct {
	URL url.URL
	// Properties() map[string]string
	// Version() Version
	// CanonicalNameNoVersion() string //return fmt.Sprintf("%s/%s/%s", d.Server(), d.Owner(), d.Repo())
	// String() string                 //return fmt.Sprintf("%s/%s/%s@%s", o.Server(), o.Owner(), o.Repo(), o.v.Canonical())
	//Server() string
	//Owner() string
	//Repo() string
}

func NewLockedCoord(
	scheme string,
	host string,
	path string,
	version Version,
	props map[string]string,
) (LockedCoord, error) {
	var result LockedCoord
	baseURL := fmt.Sprintf("%s://%s/%s", scheme, host, strings.Trim(path, "/")) //"https://github.com/owner/repo"
	params := url.Values{}
	if props != nil {
		for k, v := range props {
			params.Add(k, v)
		}
	}
	// Parse the base URL
	u, err := url.Parse(baseURL)
	if err != nil {
		return result, err
	}

	// Add query parameters
	u.RawQuery = params.Encode()
	u.Fragment = version.String() // Canonical() + Meta

	result.URL = *u
	return result, nil
}

// func NewLockedCoordLocalBlank() LockedCoord {
// 	return LockedCoord{}
// }

// func (o *LockedCoordBlank) Properties() map[string]string {
// 	if o.props == nil {
// 		o.props = make(map[string]string)
// 	}
// 	return o.props
// }
//
// func (o *LockedCoordBlank) Version() Version {
// 	return BlankVersion
// }
// func (o *LockedCoordBlank) CanonicalNameNoVersion() string {
// 	return fmt.Sprintf("blank://")
// }
// func (o *LockedCoordBlank) String() string {
// 	return fmt.Sprintf("blank://@%s", o.Version().Canonical())
// }

// CanonicalNameNoVersion is used to detect circular dependencies
//
// It returns the server and path of the package
// e.g.: github.com/mypackages/python
func (o LockedCoord) CanonicalNameNoVersion() string {
	return fmt.Sprintf("%s/%s", o.URL.Hostname(), strings.Trim(o.URL.Path, "/"))
}

func (o *LockedCoord) String() string {
	return o.URL.String()
}
