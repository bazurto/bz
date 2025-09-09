// SPDX-FileCopyrightText: 2023 RH America LLC <info@rhamerica.com>
// SPDX-License-Identifier: GPL-3.0-only

package model

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/amzn/ion-go/ion"
	"github.com/bazurto/bz/lib/utils"
	"github.com/spf13/cast"
)

type LoegacyLockedCoord struct {
	Server  string `json:"server" ion:"server"`
	Owner   string `json:"owner" ion:"owner"`
	Repo    string `json:"repo" ion:"repo"`
	Version string `json:"version" ion:"version"`
}

type LockedCoord struct {
	URL url.URL
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
	for k, v := range props {
		params.Add(k, v)
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

// CanonicalNameNoVersion is used to detect circular dependencies
//
// It returns the server and path of the package
// e.g.: github.com/mypackages/python
func (o LockedCoord) CanonicalNameNoVersion() string {
	return fmt.Sprintf("%s/%s", o.URL.Hostname(), strings.Trim(o.URL.Path, "/"))
}

// String returns the URL of the package as a string.
//
// The format of the returned string is the same as the format of the
// URL returned by url.URL.String().
func (o *LockedCoord) String() string {
	return o.URL.String()
}

// UnmarshalJSON unmarshals a JSON byte stream into the object.
//
// The format of the JSON object is the same as the format of the
// object returned by MarshalJSON.
//
// The JSON object can contain the following fields:
//
// - URL: the URL of the package
//
// If the URL field is not present, the package is assumed to be in the
// "legacy" format, which is as follows:
//
// - server: the server of the package
// - owner: the owner of the package
// - repo: the repository of the package
// - version: the version of the package
func (o *LockedCoord) UnmarshalJSON(data []byte) error {
	var tmp map[string]any
	if err := json.Unmarshal(data, &tmp); err != nil {
		return err
	}
	return o.unmarshalFromMap(tmp)
}

// UnmarshalIon unmarshals an Ion byte stream into the object.
// Ion is similar to JSON but more efficient and with more features.
// See https://github.com/amzn/ion-docs for more information.
func (o *LockedCoord) UnmarshalIon(r ion.Reader) error {
	var tmp map[string]any
	if err := ion.UnmarshalFrom(r, &tmp); err != nil {
		return err
	}
	return o.unmarshalFromMap(tmp)
}

func (o *LockedCoord) unmarshalFromMap(tmp map[string]any) error {
	if _, ok := tmp["URL"]; ok {
		// new
		var u url.URL
		if err := utils.JsonConvert(tmp["URL"], &u); err != nil {
			return err
		}
		o.URL = u
		return nil
	} else if _, ok := tmp["server"]; ok {
		// legacy
		lc, err := NewLockedCoord(
			"https",
			cast.ToString(tmp["server"]),
			fmt.Sprintf("%s/%s", cast.ToString(tmp["owner"]), cast.ToString(tmp["repo"])),
			NewVersion(cast.ToString(tmp["version"])),
			nil,
		)
		if err != nil {
			return err
		}
		o.URL = lc.URL
		return nil
	}
	return fmt.Errorf("unable to parse LockedCoord: %s",
		func() string {
			b, _ := json.Marshal(tmp)
			if b != nil {
				return (string(b))
			}
			return ""
		}(),
	)
}
