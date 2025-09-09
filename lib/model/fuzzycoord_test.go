// SPDX-FileCopyrightText: 2023 RH America LLC <info@rhamerica.com>
// SPDX-License-Identifier: GPL-3.0-only

package model_test

import (
	"net/url"
	"testing"

	"github.com/bazurto/bz/lib/model"
	"github.com/stretchr/testify/assert"
)

func TestNewCoordFromStr(t *testing.T) {
	tests := []struct {
		name     string // description of this test case
		given    string
		expected model.FuzzyCoord
		wantErr  bool
	}{
		{
			name:  "fullUrl With Version",
			given: "https://github.com/bazurto/python#3",
			expected: model.FuzzyCoord{
				OriginalString: "https://github.com/bazurto/python#3",
				URL:            url.URL{Scheme: "https", Host: "github.com", Path: "/bazurto/python", Fragment: "3"},
			},
		},
		{
			name:  "fullUrl Legacy Version",
			given: "https://github.com/bazurto/python@3",
			expected: model.FuzzyCoord{
				OriginalString: "https://github.com/bazurto/python@3",
				URL:            url.URL{Scheme: "https", Host: "github.com", Path: "/bazurto/python", Fragment: "3"},
			},
		},
		{
			name:  "githubUrlWithVersion",
			given: "github.com/bazurto/python#3",
			expected: model.FuzzyCoord{
				OriginalString: "github.com/bazurto/python#3",
				URL:            url.URL{Scheme: "https", Host: "github.com", Path: "/bazurto/python", Fragment: "3"},
			},
		},
		{
			name:  "githubUrl Legacy Version",
			given: "github.com/bazurto/python@3",
			expected: model.FuzzyCoord{
				OriginalString: "github.com/bazurto/python@3",
				URL:            url.URL{Scheme: "https", Host: "github.com", Path: "/bazurto/python", Fragment: "3"},
			},
		},
		{
			name:  "githubUrl no version",
			given: "github.com/bazurto/python",
			expected: model.FuzzyCoord{
				OriginalString: "github.com/bazurto/python",
				URL:            url.URL{Scheme: "https", Host: "github.com", Path: "/bazurto/python"},
			},
		},
		{
			name:  "githubUrl no version",
			given: "file:///tmp",
			expected: model.FuzzyCoord{
				OriginalString: "file:///tmp",
				URL:            url.URL{Scheme: "file", Path: "/tmp"},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, gotErr := model.NewCoordFromStr(tt.given)
			if gotErr != nil {
				if !tt.wantErr {
					t.Errorf("NewCoordFromStr() failed: %v", gotErr)
				}
				return
			}
			if tt.wantErr {
				t.Fatal("NewCoordFromStr() succeeded unexpectedly")
				return
			}
			assert.Equal(t, tt.expected, got)
		})
	}
}
