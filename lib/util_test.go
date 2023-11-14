// SPDX-FileCopyrightText: 2023 RH America LLC <info@rhamerica.com>
// SPDX-License-Identifier: GPL-3.0-only

package lib

import (
	"testing"

	"github.com/bazurto/bz/lib/utils"
	"github.com/stretchr/testify/assert"
)

func TestJsonDecode(t *testing.T) {
	m := make(map[string]string)
	err := utils.JsonDecode(`{"a":"b"}`, m)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(m))
	assert.Equal(t, "b", m["a"])
}
