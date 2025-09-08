// SPDX-FileCopyrightText: 2023 RH America LLC <info@rhamerica.com>
// SPDX-License-Identifier: GPL-3.0-only

package lib

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/bazurto/bz/lib/luautils"
	"github.com/bazurto/bz/lib/model"
	"github.com/bazurto/bz/lib/resolver"
	"github.com/bazurto/bz/lib/utils"
)

type Engine struct {
	appCtx    model.AppContext
	resolvers []resolver.Resolver
}

func NewEngine(appCtx model.AppContext) *Engine {
	e := &Engine{
		appCtx: appCtx,
	}

	// create directories if it does not exist
	if err := utils.MkdirIfNotExists(appCtx.UserCacheDirName); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating create cache dir: %s\n", err)
		os.Exit(1)
	}

	return e
}

func (o *Engine) Execute(execCtx *model.DependencyTree, args []string) int {
	return o.ExecuteWithIO(execCtx, args, os.Stdout, os.Stdin, os.Stderr)
}

func (o *Engine) ExecuteWithIO(
	execCtx *model.DependencyTree,
	args []string,
	stdout io.Writer,
	stderr io.Writer,
	stdin io.Reader,
) int {
	// empty
	if len(args) < 1 {
		return 0
	}

	// env vars
	os.Setenv("BZ_PROJECT_DIR", execCtx.Dir) // also have to be set in lib/model/dependency_tree.go

	ctx := execCtx.Resolve()

	// Keep Original OS Path
	originalPathPathStr := os.Getenv("PATH") // /uar/local/bin:/usr/bin

	// Expand aliases
	args = ctx.ResolveAlias(args)

	// SetEnv
	for k, v := range ctx.Env() {
		os.Setenv(k, v)
	}

	//originalOsPathParts := strings.Split(originalPathPathStr, string([]rune{os.PathListSeparator})) // ["/usr/local/bin", "/usr/bin"]
	//ctx.SetPath(append(ctx.Path(), originalOsPathParts...))
	// Restore OS path
	os.Setenv(
		"PATH",
		strings.Join(
			[]string{os.Getenv("PATH") + originalPathPathStr},
			string([]rune{os.PathListSeparator}),
		),
	)
	defer func() {
		os.Setenv("PATH", originalPathPathStr)
	}()

	prog := args[0]
	progArgs := args[1:]
	cmd := exec.Command(prog, progArgs...)
	cmd.Stdout = stdout
	cmd.Stdin = stdin
	cmd.Stderr = stderr
	err := cmd.Run()
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			return exitError.ExitCode()
		} else {
			fmt.Fprintf(os.Stderr, "`%s`: %s\n", strings.Join(args, " "), err)
			return cmd.ProcessState.ExitCode()
		}
	}
	return 0
}

func (o *Engine) AddResolver(r resolver.Resolver) {
	Debug.Printf("Start AddResolver(%s)", r)
	o.resolvers = append(o.resolvers, r)
}

// ContextFromFuzzyConfig will generate an execution context for a given directory
// without reading the fuzzy config or locked config.  This method is to be used
// for on the fly executions.  It does not update locked config
func (o *Engine) ContextFromFuzzyConfig(dir string, cc *model.FuzzyConfigContent) (*model.DependencyTree, error) {
	lcc, err := o.lockedConfigFromFuzzyConfig(cc)
	if err != nil {
		return nil, err
	}

	return o.ContextFromLockedConfig(dir, lcc)
}

// ContextFromLockedConfig will generate an execution context for a given directory
// without reading the fuzzy config or locked config.  This method is to be used
// for on the fly executions.  It does not update locked config
func (o *Engine) ContextFromLockedConfig(dir string, lcc *model.LockedConfigContent) (*model.DependencyTree, error) {
	//
	Debug.Printf("read config: %v", lcc)
	cdd := utils.NewCircularDependencyDetector()
	c, err := model.NewLockedCoord("file", "", dir, model.NewVersion("0"), nil)
	if err != nil {
		return nil, err
	}

	// resolve dependency
	execCtx, err := o.resolvedDependencyFromConfigContext(dir, &c, lcc, cdd)
	if err != nil {
		return nil, err
	}

	return execCtx, nil
}

// ContextFromConfigDir will generate the execution context from the given directory.
// It will try to read fuzzy config first, if it does, then it updates the locked config
// It then will try to read the locked config and resolves execution context
func (o *Engine) ContextFromConfigDir(dir string) (*model.DependencyTree, error) {
	var err error
	// Fuzzy Config Info
	var fuzzyConfigModTime time.Time
	fuzzyConfigFileName, fuzzyConfigFound := o.findFuzzyConfigFile(dir)
	if fuzzyConfigFound {
		if stat, err := os.Stat(fuzzyConfigFileName); err == nil {
			fuzzyConfigModTime = stat.ModTime()
		}
	}

	// Lock Config Info
	var lockConfigModTime time.Time
	lockConfigFileName := filepath.Join(dir, o.appCtx.LockFileName)
	lockConfigStat, err := os.Stat(lockConfigFileName)
	var lockConfigFound bool = true
	if os.IsNotExist(err) {
		lockConfigFound = false
	} else {
		if lockConfigStat == nil {
			return nil, fmt.Errorf("this should not happen, ContextFromConfigDir could not stat file %s", lockConfigFileName)
		}
		lockConfigModTime = lockConfigStat.ModTime()
	}

	//
	// Read
	//
	// if lockFileDoesNotExixts : read from fuzzy file
	// if fuzzyFile newer       : read from fuzzy file
	// if error                 : read from fuzzy file
	// else						: read from lock file
	readFuzzy := false
	if !lockConfigFound {
		// read fuzzy
		readFuzzy = true
		Debug.Print("lock file not found, setting readFuzzy flag to true")
	} else if fuzzyConfigModTime.After(lockConfigModTime) {
		// read fuzzy
		readFuzzy = true
		Debug.Print("fuzzy config file is newer than lock file, string readFuzzy flag to true")
	} else {
		// read lock
		Debug.Print("will read from lock file")
	}

	var shouldUpdateLockFile bool = false
	var lcc *model.LockedConfigContent
	if readFuzzy {
		// read from .bz, .bz.hcl, .bz.json
		lcc, err = o.lockedConfigFromFuzzyConfigContentInDir(dir)
		shouldUpdateLockFile = true
		if err != nil {
			return nil, err
		}
	} else {
		// read lock file from .bz.lock
		lcc, err = o.lockedConfigContentFromDir(dir)
		if err != nil {
			// on error ready fuzzy file
			lcc, err = o.lockedConfigFromFuzzyConfigContentInDir(dir)
			Warn.Printf("Failed reading %s, updating with %s", lockConfigFileName, fuzzyConfigFileName)
			shouldUpdateLockFile = true
			if err != nil {
				return nil, err
			}
		}
	}

	//
	Debug.Printf("read config: %v", lcc)
	cdd := utils.NewCircularDependencyDetector()
	c, err := model.NewLockedCoord("file", "", dir, model.NewVersion("0"), nil)
	if err != nil {
		return nil, err
	}

	// resolve dependency
	resolvedDependency, err := o.resolvedDependencyFromConfigContext(dir, &c, lcc, cdd)
	if err != nil {
		return nil, err
	}

	// update lock file
	if shouldUpdateLockFile {
		if e := o.updateLockFile(dir, resolvedDependency); e != nil {
			Warn.Println(e)
		}
	}

	return resolvedDependency, nil
}

func (o *Engine) resolvedDependencyFromConfigContext(
	dir string,
	lockedCoord *model.LockedCoord,
	lockedConfigContent *model.LockedConfigContent,
	cdd *utils.CircularDependencyDetector,
) (*model.DependencyTree, error) {
	Debug.Printf("Start resolvedDependencyFromConfigContext(%s,%v)", dir, lockedCoord)

	// dir, binDir
	dir = utils.FsAbs(dir) // default dir

	// exports
	exports := lockedConfigContent.Export
	if exports == nil {
		exports = make(map[string]string)
	}

	// aliases
	aliases := lockedConfigContent.Alias
	if aliases == nil {
		aliases = make(map[string]string)
	}

	// triggers
	triggers := lockedConfigContent.Triggers

	var subDeps []*model.DependencyTree
	for _, subLockedCoord := range lockedConfigContent.Deps {
		cdd2 := cdd.Clone()
		// Circular depedency protection
		if err := cdd2.Push(subLockedCoord.CanonicalNameNoVersion()); err != nil {
			return nil, fmt.Errorf("resolvedDependencyFromConfigContext: %w", err)
		}

		//
		// Download Dependency if it doesn't exist
		//
		//extractToDir := filepath.Join(o.resolvedCoordToDir(subLockedCoord), "extracted")
		extractToDir, err := o.downloadAndInstallDependencyIfNotExists(subLockedCoord)
		if err != nil {
			return nil, err
		}

		//
		subCc, err := o.lockedConfigContentFromDir(extractToDir)
		if err != nil {
			return nil, fmt.Errorf("load sub dependency error: %w", err)
		}

		subRd, err := o.resolvedDependencyFromConfigContext(extractToDir, subLockedCoord, subCc, cdd2.Clone())
		if err != nil {
			return nil, fmt.Errorf("resole sub dependency error: : %w", err)
		}

		subDeps = append(subDeps, subRd)
	}

	//
	rd := model.DependencyTree{}
	rd.Coord = *lockedCoord
	rd.Dir = dir
	rd.BinDir = lockedConfigContent.BinDir
	rd.Exports = exports
	rd.Alias = aliases
	rd.Triggers = triggers
	rd.Sub = subDeps
	return &rd, nil
}

// lockedConfigContentFromDir takes a directory name `dir` and returns the json from the lock file
func (o *Engine) lockedConfigContentFromDir(extractToDir string) (*model.LockedConfigContent, error) {
	configFile := filepath.Join(extractToDir, o.appCtx.LockFileName)
	if !utils.FileExists(configFile) {
		return nil, fmt.Errorf("%s: %w", configFile, utils.FileNotFoundError)
	}

	lcc, err := model.LockedConfigContentFromFile(configFile)
	if err != nil {
		return nil, fmt.Errorf("error@reading %s: %w", configFile, err)
	}

	return lcc, nil
}

func (o *Engine) findFuzzyConfigFile(dir string) (string, bool) {
	//
	for _, configFileName := range o.appCtx.ConfigFileNames {
		configFile := filepath.Join(dir, configFileName)
		if utils.FileExists(configFile) {
			return configFile, true
		}
	}

	return "", false
}

// loadFuzzyConfigFromDir loads optional fuzzy config from directory
func (o *Engine) loadFuzzyConfigFromDir(dir string) (*model.FuzzyConfigContent, error) {
	var cc *model.FuzzyConfigContent
	var err error

	// if file is in directory, return configuration
	// otherwise, return default
	if configFile, found := o.findFuzzyConfigFile(dir); found {
		cc, err = model.FuzzyConfigContentFromFile(configFile)
		if err != nil {
			return nil, fmt.Errorf("error reading %s: %w", configFile, err)
		}
	}

	// empty
	if cc == nil {
		cc = &model.FuzzyConfigContent{
			BinDir: filepath.Join(dir, "bin"),
			Deps:   nil,
			Export: make(map[string]string),
			Alias:  make(map[string]string),
		}
	}
	return cc, nil
}

// lockedConfigFromFuzzyConfigContentInDir takes a directory name `dir` and returns the json or hcl from the
// configuration file as a struct.
func (o *Engine) lockedConfigFromFuzzyConfigContentInDir(dir string) (*model.LockedConfigContent, error) {
	cc, err := o.loadFuzzyConfigFromDir(dir)
	if err != nil {
		return nil, err
	}
	return o.lockedConfigFromFuzzyConfig(cc)
}

// lockedConfigFromFuzzyConfig
func (o *Engine) lockedConfigFromFuzzyConfig(cc *model.FuzzyConfigContent) (*model.LockedConfigContent, error) {
	if len(o.resolvers) < 1 {
		return nil, fmt.Errorf("no resolvers for added to engine")
	}

	var lockedCoords []*model.LockedCoord
	for _, dep := range cc.Deps {
		fuzzyCoord, err := model.NewCoordFromStr(dep)
		if err != nil {
			return nil, err
		}

		//
		var lockCoord *model.LockedCoord
		for _, reslvr := range o.resolvers {
			lockCoord, err = reslvr.ResolveCoord(fuzzyCoord)
			if err != nil {
				return nil, fmt.Errorf("resolvedDependencyFromConfigContext: ResolveCoord: %w", err)
			}
			if lockCoord != nil {
				break
			}
		}
		if lockCoord == nil {
			return nil, fmt.Errorf("no resolver for dependency `%s`", dep)
		}

		lockedCoords = append(lockedCoords, lockCoord)

	}

	// return locked config content
	lcc := model.LockedConfigContent{}
	lcc.BinDir = cc.BinDir
	lcc.Alias = cc.Alias
	lcc.Export = cc.Export
	if cc.Triggers != nil {
		lcc.Triggers = *cc.Triggers
	}
	lcc.Deps = lockedCoords

	return &lcc, nil
}

func (o *Engine) updateLockFile(dir string, rd *model.DependencyTree) error {
	lockFileName := filepath.Join(dir, o.appCtx.LockFileName)

	cc := model.LockedConfigContent{}
	cc.Alias = rd.Alias
	cc.Triggers = rd.Triggers
	cc.Export = rd.Exports
	cc.BinDir = rd.BinDir
	for _, r := range rd.Sub {
		cc.Deps = append(cc.Deps, &r.Coord)
	}

	f, err := os.Create(lockFileName)
	if err != nil {
		return err
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")

	return enc.Encode(cc)
}

// downloadAndInstallDependencyIfNotExists does the actual work of installing
// the dependency.  It loops through all resolvers
// and unzips the dependency
func (o *Engine) downloadAndInstallDependencyIfNotExists(lockCoord *model.LockedCoord) (string, error) {
	// download if it does not exist
	var extractToDir string
	var err error
	var resolved bool
	for _, rslver := range o.resolvers {
		Debug.Printf("calling %v.DownloadResolvedCoord(%s)", rslver, lockCoord)
		extractToDir, err, resolved = rslver.DownloadResolvedCoord(*lockCoord)
		if err != nil {
			return "", fmt.Errorf("download coord: %w", err)
		}
		if resolved {
			break
		}
	}

	lc, err := o.lockedConfigContentFromDir(extractToDir)
	if err != nil {
		return "", fmt.Errorf("error loading dependency `%s`: %w", lockCoord.String(), err)
	}

	if err := o.runInstallScript(extractToDir, lc); err != nil {
		return "", fmt.Errorf("install script: %w", err)
	}

	return extractToDir, nil
}

func (o *Engine) runInstallScript(dir string, lc *model.LockedConfigContent) error {
	//
	if lc.Triggers.InstallScript == "" {
		return nil
	}

	installScript := lc.Triggers.InstallScript
	lc.Triggers.InstallScript = "" // to avoid running again

	//
	depCtx, err := o.ContextFromLockedConfig(dir, lc)
	if err != nil {
		return err
	}

	execCtx := depCtx.Resolve()
	installScriptFinal, err := execCtx.Expand(installScript)
	if err != nil {
		return err
	}

	err = luautils.RunLuaInstallScript(installScriptFinal, execCtx, nil)
	if err != nil {
		return err
	}
	return nil
}
