// SPDX-FileCopyrightText: 2023 RH America LLC <info@rhamerica.com>
// SPDX-License-Identifier: GPL-3.0-only

package lib

import (
	"strings"
)

/*
	server "github.com" {
			token: "abcde..."
	}

------------

	{
		server: {
			"github.com": {
				token: "abcde..."
			}
		}
	}
*/
type UserConfig struct {
	Servers []UserConfigServer `ion:"server" hcl:"server,block"`
}

type UserConfigServer struct {
	Name  string `ion:"name" hcl:",label"`
	Token string `ion:"token" hcl:"token,optional"`
}

type UserConfigIon struct {
	Servers map[string]UserConfigServer `ion:"server"`
}

func NewUserConfigFromFile(f string) (*UserConfig, error) {
	var err error
	cfg := UserConfig{}
	if isIonFile(f) {
		uci := UserConfigIon{}
		err = ionLoad(f, &uci)
		if err == nil {
			for serverName, attr := range uci.Servers {
				attr.Name = serverName
				cfg.Servers = append(cfg.Servers, attr)
			}
		}
	} else {
		err = hclLoad(f, &cfg)
	}
	return &cfg, err
}

func (o *UserConfig) GetServerToken(serverName string) string {
	var token string
	for _, server := range o.Servers {
		if strings.ToLower(server.Name) == strings.ToLower(serverName) {
			token = server.Token
		}
	}
	return token
}
