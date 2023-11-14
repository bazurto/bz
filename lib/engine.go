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

	"github.com/bazurto/bz/lib/model"
	"github.com/bazurto/bz/lib/resolver"
	"github.com/bazurto/bz/lib/utils"
)

// var (
// 	AppName            = "bz"
// 	LockFileName       = fmt.Sprintf(".%s.lock", AppName)
// 	HomeDir, _         = os.UserHomeDir()
// 	UserDir            = filepath.Join(HomeDir, fmt.Sprintf(".%s", AppName))
// 	UserConfigFileName = filepath.Join(UserDir, "config")
// 	UserCacheDirName   = filepath.Join(UserDir, "cache")
// 	ConfigFileNames    = []string{
// 		fmt.Sprintf(".%s.hcl", AppName),
// 		fmt.Sprintf(".%s.json", AppName),
// 		fmt.Sprintf(".%s", AppName),
// 	}
// 	bzUserConfig *model.UserConfig
// )

type Engine struct {
	//configFileNames []string
	appCtx    model.AppContext
	resolvers []resolver.Resolver
}

func NewEngine(appCtx model.AppContext) *Engine {
	e := &Engine{
		appCtx: appCtx,
		//configFileNames: appCtx.ConfigFileNames,
	}

	// create directories if it does not exist
	if err := utils.MkdirIfNotExists(appCtx.UserCacheDirName); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating create cache dir: %s\n", err)
		os.Exit(1)
	}

	// // load user config if it exists
	// if utils.FileExists(UserConfigFileName) {
	// 	var err error
	// 	bzUserConfig, err = model.NewUserConfigFromFile(UserConfigFileName)
	// 	if err != nil {
	// 		fmt.Fprintf(os.Stderr, "Error reading user config (%s): %s\n", UserConfigFileName, err)
	// 		os.Exit(1)
	// 	}
	// } else {
	// 	bzUserConfig = &model.UserConfig{}
	// }

	return e
}

func (o *Engine) Execute(rdep *model.ResolvedDependency, args []string) int {
	return o.ExecuteWithIO(rdep, args, os.Stdout, os.Stdin, os.Stderr)
}

func (o *Engine) ExecuteWithIO(
	rdep *model.ResolvedDependency,
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
	ctx := model.ExecContext{}
	rdep.Resolve(&ctx)

	//
	path := os.Getenv("PATH")
	pathParts := strings.Split(path, string([]rune{os.PathListSeparator}))
	newPath = append(newPath, pathParts...)
	os.Setenv("PATH", strings.Join(newPath, string([]rune{os.PathListSeparator})))

	// Args
	args, err := ed.ExpandCommand(args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}

	//executeCommand(args, newEnv)
	//debug.Printf("env: \n%s\n", mapJoin(env, "=", "\n"))
	//Debug.Printf("command: %s", strings.Join(args, " "))
	// SetEnv
	for k, v := range newEnv {
		os.Setenv(k, v)
	}

	prog := args[0]
	progArgs := args[1:]
	cmd := exec.Command(prog, progArgs...)
	cmd.Stdout = os.Stdout
	cmd.Stdin = os.Stdin
	cmd.Stderr = os.Stderr
	err = cmd.Run()
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

func (o *Engine) ContextFromConfigDir(dir string) (*model.ResolvedDependency, error) {
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
		Debug.Print("fuzzy config file is newer thatn lock file, stting readFuzzy flag to true")
	} else {
		// read lock
		Debug.Print("will read from lock file")
	}

	var shouldUpdateLockFile bool = false
	var lcc *model.LockedConfigContent
	if readFuzzy {
		// read from .bz, .bz.hcl, .bz.json
		lcc, err = o.readFuzzyConfigContentFromDir(dir)
		shouldUpdateLockFile = true
		if err != nil {
			return nil, err
		}
	} else {
		// read lock file from .bz.lock
		lcc, err = o.lockedConfigContentFromDir(dir)
		if err != nil {
			// on error ready fuzzy file
			lcc, err = o.readFuzzyConfigContentFromDir(dir)
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
	v := model.NewVersion("0.0.0")
	c := model.LockedCoord{
		Server:  "localhost",
		Owner:   "local",
		Repo:    "local",
		Version: v,
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
	rcoord *model.LockedCoord,
	bzContent *model.LockedConfigContent,
	cdd *utils.CircularDependencyDetector,
) (*model.ResolvedDependency, error) {
	Debug.Printf("Start resolvedDependencyFromConfigContext(%s,%v)", dir, rcoord)

	// dir, binDir
	dir = utils.FsAbs(dir) // default dir

	// exports
	exports := bzContent.Export
	if exports == nil {
		exports = make(map[string]string)
	}

	// aliases
	aliases := bzContent.Alias
	if aliases == nil {
		aliases = make(map[string]string)
	}

	// triggers
	triggers := bzContent.Triggers

	var subDeps []*model.ResolvedDependency
	for _, subLockedCoord := range bzContent.Deps {
		cdd2 := cdd.Clone()
		// Circular depedency protection
		if err := cdd2.Push(subLockedCoord.CanonicalNameNoVersion()); err != nil {
			return nil, fmt.Errorf("resolvedDependencyFromConfigContext: %w", err)
		}

		//
		// Download Dependency if it doesn't exist
		//
		extractToDir := filepath.Join(o.resolvedCoordToDir(subLockedCoord), "extracted")
		if err := o.downloadAndInstallDependencyIfNotExists(subLockedCoord, extractToDir); err != nil {
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
	rd := model.ResolvedDependency{}
	rd.Coord = *rcoord
	rd.Dir = dir
	rd.BinDir = bzContent.BinDir
	rd.Exports = exports
	rd.Alias = aliases
	rd.Triggers = triggers
	rd.Sub = subDeps
	return &rd, nil
}

func (o *Engine) resolvedCoordToDir(rcoord *model.LockedCoord) string {
	dir := filepath.Join(
		o.appCtx.UserCacheDirName,
		"deps",
		rcoord.Server,
		rcoord.Owner,
		rcoord.Repo,
		fmt.Sprintf("v%s", rcoord.Version.Canonical()),
	)
	return dir
}

// lockedConfigContentFromDir takes a directory name `dir` and returns the json from the lock file
func (o *Engine) lockedConfigContentFromDir(extractToDir string) (*model.LockedConfigContent, error) {
	configFile := filepath.Join(extractToDir, o.appCtx.LockFileName)
	if !utils.FileExists(configFile) {
		return nil, fmt.Errorf("%s: %w", configFile, utils.FileNotFoundError)
	}

	lcc, err := model.LockedConfigContentFromFile(configFile)
	if err != nil {
		return nil, fmt.Errorf("error reading %s: %w", configFile, err)
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

// readFuzzyConfigContentFromDir takes a directory name `dir` and returns the json or hcl from the
// configuration file as a struct.
func (o *Engine) readFuzzyConfigContentFromDir(extractToDir string) (*model.LockedConfigContent, error) {
	var cc *model.FuzzyConfigContent
	var err error

	// if file is in directory, return configuration
	// otherwise, return default
	if configFile, found := o.findFuzzyConfigFile(extractToDir); found {
		cc, err = model.FuzzyConfigContentFromFile(configFile)
		if err != nil {
			return nil, fmt.Errorf("error reading %s: %w", configFile, err)
		}
	}

	// empty
	if cc == nil {
		cc = &model.FuzzyConfigContent{
			BinDir: filepath.Join(extractToDir, "bin"),
			Deps:   nil,
			Export: make(map[string]string),
			Alias:  make(map[string]string),
		}
	}

	var lockedCoords []*model.LockedCoord
	for _, dep := range cc.Deps {
		fuzzyCoord, err := model.NewCoordFromStr(dep)
		if err != nil {
			return nil, err

		}

		//
		var lockCoord *model.LockedCoord
		for _, resolver := range o.resolvers {
			//Info.Printf("calling %v.ResoveCoord(%v)", resolver, fuzzyCoord)
			lockCoord, err = resolver.ResolveCoord(fuzzyCoord)
			if err != nil {
				return nil, fmt.Errorf("resolvedDependencyFromConfigContext: ResolveCoord: %w", err)
			}
			if lockCoord != nil {
				break
			}
		}
		if lockCoord == nil {
			return nil, fmt.Errorf("resolvedDependencyFromConfigContext: unable to resolve `%s`", lockCoord)
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

func (o *Engine) updateLockFile(dir string, rd *model.ResolvedDependency) error {
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
func (o *Engine) downloadAndInstallDependencyIfNotExists(lockCoord *model.LockedCoord, extractToDir string) error {
	// nothing to do... already installed
	if utils.FileExists(extractToDir) {
		return nil
	}

	// download if it does not exists
	var file string
	var err error
	subDir := o.resolvedCoordToDir(lockCoord)
	if !utils.FileExists(subDir) {
		for _, resolver := range o.resolvers {
			Info.Printf("calling %v.DownloadResolvedCoord(%s)", resolver, lockCoord)
			file, err = resolver.DownloadResolvedCoord(lockCoord, subDir)
			if err != nil {
				return fmt.Errorf("download coord: %w", err)
			}
			if utils.FileExists(file) {
				break
			}
		}
	}

	err = o.extractDependency(lockCoord, file, extractToDir)
	if err != nil {
		return fmt.Errorf("unable to extract dependency: %w", err)
	}

	lc, err := o.lockedConfigContentFromDir(extractToDir)
	if err != nil {
		return fmt.Errorf("load config content from dir: %w", err)
	}

	if err := lc.Triggers.RunInstallScript(lc); err != nil {
		return fmt.Errorf("install script: %w", err)
	}

	return nil
}

func (o *Engine) extractDependency(rcoord *model.LockedCoord, file string, extractToDir string) error {
	var err error
	ext := filepath.Ext(file)
	if ext == ".zip" {
		err = utils.Unzip(file, extractToDir)
	} else if ext == ".tgz" {
		err = utils.Untgz(file, extractToDir)
	}
	if err != nil {
		return err
	}
	return nil
}
