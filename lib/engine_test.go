package lib

import (
	"bytes"
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
	err := engine.runInstallScript(dep)
	assert.NoError(t, err)
}

func TestRunInstallScript_InstallScriptAlreadyDone(t *testing.T) {
	engine := NewEngine(*model.NewDefaultAppContext())
	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "install.lua")
	doneFile := scriptPath + ".done"
	_ = os.WriteFile(doneFile, []byte("done"), 0640)
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
	err := engine.runInstallScript(dep)
	assert.NoError(t, err)
	assert.FileExists(t, doneFile)
	assert.NoFileExists(t, filepath.Join(tmpDir, "luaexecuted.txt"))
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

	err := engine.runInstallScript(dep)
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

	err := engine.runInstallScript(dep)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "lua error")
}
