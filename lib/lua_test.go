package lib

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRunLua(t *testing.T) {
	err := RunLua(`print("hello world")`)
	assert.Nil(t, err)
}

func TestRunLuaMod(t *testing.T) {
	err := RunLua(`
local m = require("file")
local s = m.join("/a", "b", "c", "d")
print(s)
	`)
	assert.Nil(t, err)
}

func TestRunLuaModTgz(t *testing.T) {
	err := RunLua(`
local m = require("file")
m.tgz("a", "b")
	`)
	assert.Nil(t, err)
}
