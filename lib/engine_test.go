package lib

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/bazurto/bz/lib/model"
	"github.com/bazurto/bz/lib/resolver"
	"github.com/stretchr/testify/assert"
)

func TestExecutionContext(t *testing.T) {
	appCtx := model.NewDefaultAppContext()

	// Setup Engine
	engine := NewEngine(*appCtx)
	engine.AddResolver(resolver.NewGithubResolver(appCtx))
	engine.AddResolver(resolver.NewLocalDevResolver(appCtx))

	tmpDir := t.TempDir()
	cc := &model.FuzzyConfigContent{}
	cc.Deps = []string{"github.com/bazurto/python@3.11.1"}
	execCtx, err := engine.ContextFromFuzzyConfig(tmpDir, cc)
	if err != nil {
		t.Fatal(err)
	}

	cmd := []string{"python", "-c", "import os; import sys; [print(f'#{k}={v}$') for k, v in os.environ.items()]; print('#ACTUALPYTHONEXEC=%s$' % sys.executable)"}
	out := &bytes.Buffer{}
	//out := os.Stdout
	exitCode := engine.ExecuteWithIO(execCtx, cmd, out, os.Stderr, os.Stdin)
	if exitCode != 0 {
		t.Fatalf("non zero exit code: %d", exitCode)
	}
	result := out.String()
	assert.Contains(t, result, "#GITHUB_COM_BAZURTO_PYTHON_3_11_1_BINDIR=")
	assert.Contains(t, result, "#BAZURTO_PYTHON_3_11_1_BINDIR=")
	assert.Contains(t, result, "#BAZURTO_PYTHON_3_11_1_BINDIR=")
	assert.Contains(t, result, "#PYTHON_3_11_1_BINDIR=")
	assert.Contains(t, result, "#PYTHON_BINDIR=")

	assert.Contains(t, result, "#GITHUB_COM_BAZURTO_PYTHON_3_11_1_DIR=")
	assert.Contains(t, result, "#BAZURTO_PYTHON_3_11_1_DIR=")
	assert.Contains(t, result, "#BAZURTO_PYTHON_3_11_1_DIR=")
	assert.Contains(t, result, "#PYTHON_3_11_1_DIR=")
	assert.Contains(t, result, "#BAZURTO_PYTHON_DIR=")
	assert.Contains(t, result, "#PYTHON_DIR=")

	assert.Regexp(t, "#BAZURTO_PYTHON_3_11_1_BINDIR=.*extracted[\\\\/]bin\\$", result)
	assert.Regexp(t, "#BAZURTO_PYTHON_3_11_1_DIR=.*extracted\\$", result)

	assert.Contains(t, result, fmt.Sprintf("#BZ_PROJECT_DIR=%s$", tmpDir))
}
func TestRunInstallScript_NoInstallScript(t *testing.T) {
	engine := NewEngine(*model.NewDefaultAppContext())
	dep := model.DependencyTree{
		Triggers: model.Triggers{
			InstallScript: "",
		},
	}
	err := engine.runInstallScript(&dep)
	assert.NoError(t, err)
}

func TestRunInstallScript_InstallScriptAlreadyDone(t *testing.T) {
	engine := NewEngine(*model.NewDefaultAppContext())
	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "install.lua")
	doneFile := scriptPath + ".done"
	scriptContent := []byte(`
	dir = os.getenv("DIR")
	local f = io.open(dir .. "/luaexecuted.txt", "w")
	f:write("this was created by lua")
	f:close()
	`)
	_ = os.WriteFile(scriptPath, scriptContent, 0640)
	// done marker holds the hash of the script content
	_ = os.WriteFile(doneFile, []byte(fmt.Sprintf("%x", sha256.Sum256(scriptContent))), 0640)
	dep := model.DependencyTree{
		Dir: tmpDir,
		Triggers: model.Triggers{
			InstallScript: scriptPath,
		},
	}
	err := engine.runInstallScript(&dep)
	assert.NoError(t, err)
	assert.FileExists(t, doneFile)
	assert.NoFileExists(t, filepath.Join(tmpDir, "luaexecuted.txt"))
}

func TestRunInstallScript_RerunsWhenScriptChanges(t *testing.T) {
	engine := NewEngine(*model.NewDefaultAppContext())
	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "install.lua")
	script := `
	local f = io.open(os.getenv("DIR") .. "/install_count.txt", "a")
	f:write("x")
	f:close()
	`
	_ = os.WriteFile(scriptPath, []byte(script), 0640)

	dep := model.DependencyTree{
		Dir: tmpDir,
		Triggers: model.Triggers{
			InstallScript: scriptPath,
		},
	}

	// first run executes
	assert.NoError(t, engine.runInstallScript(&dep))
	// unchanged script does not run again
	assert.NoError(t, engine.runInstallScript(&dep))
	count, _ := os.ReadFile(filepath.Join(tmpDir, "install_count.txt"))
	assert.Len(t, count, 1)

	// edited script runs again
	_ = os.WriteFile(scriptPath, []byte(script+"\n-- edited\n"), 0640)
	assert.NoError(t, engine.runInstallScript(&dep))
	count, _ = os.ReadFile(filepath.Join(tmpDir, "install_count.txt"))
	assert.Len(t, count, 2)
}

func TestRunInstallScript_InstallScriptRunsLua(t *testing.T) {
	engine := NewEngine(*model.NewDefaultAppContext())
	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "install.lua")
	_ = os.WriteFile(scriptPath, []byte(`
	dir = os.getenv("DIR") 
	local f = io.open(dir .. "/luaexecuted.txt", "w")
	f:write("this was created by lua")
	f:close()
	`), 0640)

	dep := model.DependencyTree{
		Dir: tmpDir,
		Triggers: model.Triggers{
			InstallScript: scriptPath,
		},
	}

	err := engine.runInstallScript(&dep)
	assert.NoError(t, err)
	assert.FileExists(t, scriptPath+".done")
	assert.FileExists(t, filepath.Join(tmpDir, "luaexecuted.txt"))
}

func TestRunInstallScript_LuaScriptError(t *testing.T) {
	engine := NewEngine(*model.NewDefaultAppContext())
	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "install.lua")
	_ = os.WriteFile(scriptPath, []byte(`
	error("this is a lua error")
	`), 0640)

	dep := model.DependencyTree{
		Dir: tmpDir,
		Triggers: model.Triggers{
			InstallScript: scriptPath,
		},
	}

	err := engine.runInstallScript(&dep)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "lua error")

	// no done marker is written when the script fails
	assert.NoFileExists(t, scriptPath+".done")
}

// TestMetaPackageTriggers exercises the meta-package pattern end to end: a
// local dependency whose installScript performs the installation once and
// whose preRunScript sets up variables before every command.
func TestMetaPackageTriggers(t *testing.T) {
	appCtx := model.NewDefaultAppContext()
	appCtx.UserCacheDirName = t.TempDir()
	engine := NewEngine(*appCtx)
	engine.AddResolver(resolver.NewLocalDevResolver(appCtx))

	// meta-package with install and pre-run triggers
	depDir := t.TempDir()
	_ = os.WriteFile(filepath.Join(depDir, "install.lua"), []byte(`
	local f = io.open(os.getenv("DIR") .. "/install_count.txt", "a")
	f:write("x")
	f:close()
	`), 0640)
	_ = os.WriteFile(filepath.Join(depDir, "prerun.lua"), []byte(`
	local f = io.open(os.getenv("DIR") .. "/prerun_count.txt", "a")
	f:write("x")
	f:close()
	return { env = { META_VAR = "from_prerun" } }
	`), 0640)
	_ = os.WriteFile(filepath.Join(depDir, appCtx.LockFileName), []byte(`{
		"env": {"META_HOME": "$DIR"},
		"triggers": {
			"installScript": "$DIR/install.lua",
			"preRunScript": "$DIR/prerun.lua"
		}
	}`), 0640)

	// project depending on the meta-package
	projDir := t.TempDir()
	cc := &model.FuzzyConfigContent{Deps: []string{"file://" + depDir}}
	tree, err := engine.ContextFromFuzzyConfig(projDir, cc)
	if err != nil {
		t.Fatal(err)
	}

	out := &bytes.Buffer{}
	exitCode := engine.ExecuteWithIO(tree, []string{"env"}, out, os.Stderr, os.Stdin)
	assert.Equal(t, 0, exitCode)

	// pre-run env overrides and exports reach the executed command
	assert.Contains(t, out.String(), "META_VAR=from_prerun")
	assert.Contains(t, out.String(), fmt.Sprintf("META_HOME=%s", depDir))

	// install script ran once, pre-run script ran exactly once (not once
	// per Resolve call)
	installCount, _ := os.ReadFile(filepath.Join(depDir, "install_count.txt"))
	assert.Len(t, installCount, 1)
	preRunCount, _ := os.ReadFile(filepath.Join(depDir, "prerun_count.txt"))
	assert.Len(t, preRunCount, 1)

	// a second invocation re-runs the pre-run script but not the install script
	tree2, err := engine.ContextFromFuzzyConfig(projDir, cc)
	if err != nil {
		t.Fatal(err)
	}
	exitCode = engine.ExecuteWithIO(tree2, []string{"env"}, &bytes.Buffer{}, os.Stderr, os.Stdin)
	assert.Equal(t, 0, exitCode)
	installCount, _ = os.ReadFile(filepath.Join(depDir, "install_count.txt"))
	assert.Len(t, installCount, 1)
	preRunCount, _ = os.ReadFile(filepath.Join(depDir, "prerun_count.txt"))
	assert.Len(t, preRunCount, 2)
}
