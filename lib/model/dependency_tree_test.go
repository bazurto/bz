package model

import (
	"net/url"
	"os"
	"path/filepath"
	"testing"
)

type dummyCoord struct {
	URL *url.URL
}

func TestDependencyTree_resolveLocalEnvVars_Basic(t *testing.T) {
	dir := t.TempDir()
	lua := `
	local i = 0
	local calc = 0
	for i = 1,10 do
		calc = i + calc
	end
	return {
		env = {
			CALC_VAR1 = "testprerun",
			CALC_VAR2 = tostring(calc),
		},
	}
	`
	luaPath := filepath.Join(dir, "prerun.lua")
	os.WriteFile(luaPath, []byte(lua), 0644)
	c, err := NewLockedCoord("file", "", dir, NewVersion("0"), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Setup
	os.Setenv("BZ_PROJECT_DIR", "/tmp/project")
	dt := &DependencyTree{
		Coord: c,
		Dir:   dir,
		Exports: map[string]string{
			"GIVEN_VAR1": "$DIR/some/path",
		},
		Triggers: Triggers{
			PreRunScript: "$DIR/prerun.lua",
		},
	}

	// Act
	ctx, err := dt.resolveLocalEnvVars(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Assert
	env := ctx.Env()
	if env["CALC_VAR1"] != "testprerun" {
		t.Errorf("CALC_VAR1 not set correctly, got %s", env["CALC_VAR1"])
	}

	if env["GIVEN_VAR1"] != dir+"/some/path" {
		t.Errorf("GIVEN_VAR1 not set correctly, got %s", env["GIVEN_VAR1"])
	}

	if env["DIR"] != dir {
		t.Errorf("DIR not set correctly, got %s", env["DIR"])
	}

	if env["CALC_VAR2"] != "55" {
		t.Errorf("CALC_VAR2 not set correctly, got %s", env["CALC_VAR2"])
	}
}
