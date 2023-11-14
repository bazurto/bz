// SPDX-FileCopyrightText: 2023 RH America LLC <info@rhamerica.com>
// SPDX-License-Identifier: GPL-3.0-only

package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestVersionEqual(t *testing.T) {
	v1 := NewVersion("0.0.0.1+asdfasdf")
	v2 := NewVersion("0.0.0.1")

	assert.Equal(t, v1.Compare(v2), 0)
}

func TestVersionEqualWithV(t *testing.T) {
	v1 := NewVersion("v0.0.0.1+asdfasdf")
	v2 := NewVersion("0.0.0.1")

	assert.Equal(t, v1.Compare(v2), 0)
}
func TestVersionGrater(t *testing.T) {
	v1 := NewVersion("0.0.2")
	v2 := NewVersion("0.0.0.1")

	assert.Greater(t, v1.Compare(v2), 0)
}
func TestVersionLessThan(t *testing.T) {
	v1 := NewVersion("0.0.2")
	v2 := NewVersion("0.2")

	assert.Less(t, v1.Compare(v2), 0)
}

func TestVersionEmpty(t *testing.T) {
	v1 := NewVersion("") // 0.0.0
	v2 := NewVersion("0.2")

	assert.Less(t, v1.Compare(v2), 0)
}

func TestVersionPre(t *testing.T) {
	v1 := NewVersion("1.2.3-pre")
	v2 := NewVersion("1.2.3")

	assert.Less(t, v1.Compare(v2), 0)
}

func TestVersionPattern(t *testing.T) {
	d := map[string][]string{
		"*": {
			"1",
			"1.2",
			"1.2.3",
			"1.2.3.4",
			"1-pre",
			"1.2-pre",
			"1.2.3-pre",
			"1.2.3.4-pre",
			"1+extra",
			"1.2+extra",
			"1.2.3+extra",
			"1.2.3.4+extra",
			"1-pre+extra",
			"1.2-pre+extra",
			"1.2.3-pre+extra",
			"1.2.3.4-pre+extra",
		},
		"1.*": {
			"1",
			"1.2",
			"1.2.3",
			"1.2.3.4",
			"1-pre",
			"1.2-pre",
			"1.2.3-pre",
			"1.2.3.4-pre",
			"1+extra",
			"1.2+extra",
			"1.2.3+extra",
			"1.2.3.4+extra",
			"1-pre+extra",
			"1.2-pre+extra",
			"1.2.3-pre+extra",
			"1.2.3.4-pre+extra",
		},
		"1.2.*": {
			//"1",
			"1.2",
			"1.2.3",
			"1.2.3.4",
			//"1-pre",
			"1.2-pre",
			"1.2.3-pre",
			"1.2.3.4-pre",
			//"1+extra",
			"1.2+extra",
			"1.2.3+extra",
			"1.2.3.4+extra",
			//"1-pre+extra",
			"1.2-pre+extra",
			"1.2.3-pre+extra",
			"1.2.3.4-pre+extra",
		},
	}

	for patternStr, versions := range d {
		for _, versionStr := range versions {
			p := NewVersionPattern(patternStr)
			v := NewVersion(versionStr)
			assert.True(t, p.Matches(v), "%s should match %s", patternStr, versionStr)
		}

	}
}

func TestVersionPatternDoesNotMatch(t *testing.T) {
	d := map[string][]string{
		"1.*": {
			"2",
			"0.2",
			"2.2.3",
			"3.2.3.4",
			"2-pre",
			"0.2-pre",
			"2.2.3-pre",
			"3.2.3.4-pre",
			"2+extra",
			"0.2+extra",
			"2.2.3+extra",
			"3.2.3.4+extra",
			"2-pre+extra",
			"0.2-pre+extra",
			"2.2.3-pre+extra",
			"3.2.3.4-pre+extra",
		},
		"1.2.*": {
			"0.1",
			"2.1",
			"1",
			"0.2",
			"2.2.3",
			"3.2.3.4",
			"1-pre",
			"2.2-pre",
			"3.2.3-pre",
			"4.2.3.4-pre",
			"1+extra",
			"2.2+extra",
			"3.2.3+extra",
			"4.2.3.4+extra",
			"1-pre+extra",
			"2.2-pre+extra",
			"3.2.3-pre+extra",
			"4.2.3.4-pre+extra",
		},
	}

	for patternStr, versions := range d {
		for _, versionStr := range versions {
			p := NewVersionPattern(patternStr)
			v := NewVersion(versionStr)
			assert.False(t, p.Matches(v), "%s should not match %s", patternStr, versionStr)
		}

	}
}
