// SPDX-FileCopyrightText: 2023 RH America LLC <info@rhamerica.com>
// SPDX-License-Identifier: GPL-3.0-only

package lib

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

var (
	AppName            = "bz"
	LockFileName       = fmt.Sprintf(".%s.lock", AppName)
	HomeDir, _         = os.UserHomeDir()
	UserDir            = filepath.Join(HomeDir, fmt.Sprintf(".%s", AppName))
	UserConfigFileName = filepath.Join(UserDir, "config")
	UserCacheDirName   = filepath.Join(UserDir, "cache")
	ConfigFileNames    = []string{
		fmt.Sprintf(".%s.hcl", AppName),
		fmt.Sprintf(".%s.json", AppName),
		fmt.Sprintf(".%s", AppName),
	}
	bzUserConfig *UserConfig
)

type Engine struct {
	configFileNames []string
	resolvers       []Resolver
}

func NewEngine() *Engine {
	e := &Engine{
		configFileNames: ConfigFileNames,
	}

	// create directories if it does not exist
	if err := mkdirIfNotExists(UserCacheDirName); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating create cache dir: %s\n", err)
		os.Exit(1)
	}

	// load user config if it exists
	if fileExists(UserConfigFileName) {
		var err error
		bzUserConfig, err = NewUserConfigFromFile(UserConfigFileName)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading user config (%s): %s\n", UserConfigFileName, err)
			os.Exit(1)
		}
	} else {
		bzUserConfig = &UserConfig{}
	}

	return e
}

func (o *Engine) AddResolver(r Resolver) {
	Debug.Printf("Start AddResolver(%s)", r)
	//defer Debug.Printf("End AddResolver(%s)", r)
	o.resolvers = append(o.resolvers, r)
}

func (o *Engine) ContextFromConfigDir(dir string) (*ResolvedDependency, error) {
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
	lockConfigFileName := filepath.Join(dir, LockFileName)
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

	var updateLockFile bool = false
	var lcc *LockedConfigContent
	if readFuzzy {
		// read from .bz, .bz.hcl, .bz.json
		lcc, err = o.readFuzzyConfigContentFromDir(dir)
		updateLockFile = true
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
			updateLockFile = true
			if err != nil {
				return nil, err
			}
		}
	}

	//
	Debug.Printf("read config: %s", lcc)
	cdd := NewCircularDependencyDetector()
	v := NewVersion("0.0.0")
	c := LockedCoord{
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
	if updateLockFile {
		if e := o.updateLockFile(dir, resolvedDependency); e != nil {
			Warn.Println(e)
		}
	}

	return resolvedDependency, nil
}

func (o *Engine) resolvedDependencyFromConfigContext(
	dir string,
	rcoord *LockedCoord,
	bzContent *LockedConfigContent,
	cdd *CircularDependencyDetector,
) (*ResolvedDependency, error) {
	Debug.Printf("Start resolvedDependencyFromConfigContext(%s,%v)", dir, rcoord)

	// dir, binDir
	dir = fsAbs(dir) // default dir

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

	var subDeps []*ResolvedDependency
	for _, subLockedCoord := range bzContent.Deps {
		cdd2 := cdd.Clone()
		// Circular depedency protection
		if err := cdd2.Push(subLockedCoord.CanonicalNameNoVersion()); err != nil {
			return nil, fmt.Errorf("resolvedDependencyFromConfigContext: %w", err)
		}

		//
		extractToDir := filepath.Join(o.resolvedCoordToDir(subLockedCoord), "extracted")
		if !fileExists(extractToDir) {
			if err := o.downloadDependencyToDir(subLockedCoord, extractToDir); err != nil {
				return nil, err
			}
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
	rd := ResolvedDependency{}
	rd.Coord = *rcoord
	rd.Dir = dir
	rd.BinDir = bzContent.BinDir
	rd.Exports = exports
	rd.Alias = aliases
	rd.Sub = subDeps
	return &rd, nil
}

func (o *Engine) resolvedCoordToDir(rcoord *LockedCoord) string {
	dir := filepath.Join(
		UserCacheDirName,
		"deps",
		rcoord.Server,
		rcoord.Owner,
		rcoord.Repo,
		fmt.Sprintf("v%s", rcoord.Version.Canonical()),
	)
	return dir
}

/*
func (o *Engine) resolvedCoordToAssetFile(rcoord *LockCoord) string {
	dir := o.resolvedCoordToDir(rcoord)
	if err := mkdirIfNotExists(dir); err != nil {
		Warn.Println(err)
	}
	return filepath.Join(
		dir,
		fmt.Sprintf("%s-%s.zip", rcoord.Repo, rcoord.Version.Canonical()),
	)
}
*/

func (o *Engine) extractDependency(rcoord *LockedCoord, file string, extractToDir string) error {
	var err error
	ext := filepath.Ext(file)
	if ext == ".zip" {
		err = Unzip(file, extractToDir)
	} else if ext == ".tgz" {
		err = Untgz(file, extractToDir)
	}
	if err != nil {
		return err
	}
	return nil
}

// lockedConfigContentFromDir takes a directory name `dir` and returns the json from the lock file
func (o *Engine) lockedConfigContentFromDir(extractToDir string) (*LockedConfigContent, error) {
	configFile := filepath.Join(extractToDir, LockFileName)
	if !fileExists(configFile) {
		return nil, fmt.Errorf("%s: %w", configFile, FileNotFoundError)
	}

	lcc, err := LockedConfigContentFromFile(configFile)
	if err != nil {
		return nil, fmt.Errorf("error reading %s: %w", configFile, err)
	}

	return lcc, nil
}

func (o *Engine) findFuzzyConfigFile(dir string) (string, bool) {
	//
	for _, configFileName := range o.configFileNames {
		configFile := filepath.Join(dir, configFileName)
		if fileExists(configFile) {
			return configFile, true
		}
	}

	return "", false
}

// readFuzzyConfigContentFromDir takes a directory name `dir` and returns the json or hcl from the
// configuration file as a struct.
func (o *Engine) readFuzzyConfigContentFromDir(extractToDir string) (*LockedConfigContent, error) {
	var cc *FuzzyConfigContent
	var err error

	// if file is in directory, return configuration
	// otherwise, return default
	if configFile, found := o.findFuzzyConfigFile(extractToDir); found {
		cc, err = FuzzyConfigContentFromFile(configFile)
		if err != nil {
			return nil, fmt.Errorf("error reading %s: %w", configFile, err)
		}
	}

	// empty
	if cc == nil {
		cc = &FuzzyConfigContent{
			BinDir: filepath.Join(extractToDir, "bin"),
			Deps:   nil,
			Export: make(map[string]string),
			Alias:  make(map[string]string),
		}
	}

	var lockedCoords []*LockedCoord
	for _, dep := range cc.Deps {
		fuzzyCoord, err := NewCoordFromStr(dep)
		if err != nil {
			return nil, err

		}

		//
		var lockCoord *LockedCoord
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
	lcc := LockedConfigContent{}
	lcc.BinDir = cc.BinDir
	lcc.Alias = cc.Alias
	lcc.Export = cc.Export
	lcc.Deps = lockedCoords

	return &lcc, nil
}

func (o *Engine) updateLockFile(dir string, rd *ResolvedDependency) error {
	lockFileName := filepath.Join(dir, LockFileName)

	cc := LockedConfigContent{}
	cc.Alias = rd.Alias
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

func (o *Engine) downloadDependencyToDir(lockCoord *LockedCoord, extractToDir string) error {
	// download if it does not exists
	var file string
	var err error
	subDir := o.resolvedCoordToDir(lockCoord)
	if !fileExists(subDir) {
		for _, resolver := range o.resolvers {
			Info.Printf("calling %v.DownloadResolvedCoord(%s)", resolver, lockCoord)
			file, err = resolver.DownloadResolvedCoord(lockCoord, subDir)
			if err != nil {
				return fmt.Errorf("download coord: %w", err)
			}
			if fileExists(file) {
				break
			}
		}
	}

	err = o.extractDependency(lockCoord, file, extractToDir)
	if err != nil {
		return fmt.Errorf("unable to extract dependency: %w", err)
	}
	return nil
}
