// SPDX-FileCopyrightText: 2023 RH America LLC <info@rhamerica.com>
// SPDX-License-Identifier: GPL-3.0-only

package utils

/*
func TestRunLua(t *testing.T) {
	err := RunScriptCode(`
	std.printf("Hello World\n");
	std.printf("Hello World\n");
	std.printf("Hello World\n");
	std.printf("Hello World\n");
	exports.a = function() { console.log("hello world"); };
	exports.b = "B";
	exports.c = "C";
	printf("hello\n")
	try {
		exports.e = file.exists("lua_test.go");
	} catch (e) {
		console.log(e)
	}
	`)
	assert.Nil(t, err)
}

func TestRunLuaMod(t *testing.T) {
	err := RunLua(`
local m = require("file")
local s = m.join("/a", "b", "c", "d")
print(s)
return {}
	`)
	assert.Nil(t, err)
}

func TestRunLuaModTgz(t *testing.T) {
	err := RunLua(`
local m = require("file")
m.tgz("a", "b")
return {}
	`)
	fmt.Printf("%s\n", err)
}
*/
