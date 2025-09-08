// SPDX-FileCopyrightText: 2023 RH America LLC <info@rhamerica.com>
// SPDX-License-Identifier: GPL-3.0-only

package model_test

import (
	"net/url"
	"testing"

	"github.com/bazurto/bz/lib/model"
)

func TestNewCoordFromStr(t *testing.T) {
	tests := []struct {
		name string // description of this test case
		// Named input parameters for target function.
		depStrArg string
		want      model.FuzzyCoord
		wantErr   bool
	}{
		{
			name:      "fullUrlWithVersion",
			depStrArg: "https://github.com/bazurto/python#3",
			want: model.FuzzyCoord{
				OriginalString: "https://github.com/bazurto/python#3",
				URL: url.URL{
					Scheme:   "https",
					Host:     "github.com",
					Path:     "/bazurto/python",
					Fragment: "3",
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, gotErr := model.NewCoordFromStr(tt.depStrArg)
			if gotErr != nil {
				if !tt.wantErr {
					t.Errorf("NewCoordFromStr() failed: %v", gotErr)
				}
				return
			}
			if tt.wantErr {
				t.Fatal("NewCoordFromStr() succeeded unexpectedly")
			}
			// TODO: update the condition below to compare got with tt.want.
			if true {
				t.Errorf("NewCoordFromStr() = %v, want %v", got, tt.want)
			}
		})
	}
}
