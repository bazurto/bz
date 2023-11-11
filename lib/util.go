// SPDX-FileCopyrightText: 2023 RH America LLC <info@rhamerica.com>
// SPDX-License-Identifier: GPL-3.0-only

package lib

import (
	"archive/tar"
	"archive/zip"
	"bufio"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/amzn/ion-go/ion"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/vibrantbyte/go-antpath/antpath"
)

var (
	FileNotFoundError = fmt.Errorf("file not found")
	antMatcher        = antpath.New()
)

func mkdir(d string) error {
	if err := os.MkdirAll(d, 0770); err != nil {
		return err
	}
	return nil
}

func fileExists(f string) bool {
	_, err := os.Stat(f)
	return !os.IsNotExist(err)
}

func mkdirIfNotExists(d string) error {
	if fileExists(d) {
		return nil
	}
	return mkdir(d)
}

func Uncompress(archiveFileName, dstDirName string) error {
	if strings.HasSuffix(archiveFileName, ".zip") {
		return Unzip(archiveFileName, dstDirName)
	} else if strings.HasSuffix(archiveFileName, ".tgz") {
		return Untgz(archiveFileName, dstDirName)
	} else if strings.HasSuffix(archiveFileName, ".tar.gz") {
		return Untgz(archiveFileName, dstDirName)
	}
	return fmt.Errorf("archive file extension not supported: %s", archiveFileName)
}

func Unzip(zipFileName, dstDirName string) error {
	reader, err := zip.OpenReader(zipFileName)
	if err != nil {
		return err
	}
	defer reader.Close()

	if err := os.MkdirAll(dstDirName, 0755); err != nil {
		return err
	}

	// valid prefix to avoid out of ZipSlip hack
	validPrefix := filepath.Clean(dstDirName) + string(os.PathSeparator)

	// Closure to address file descriptors issue with all the deferred .Close() methods
	extractAndWriteFile := func(zipFile *zip.File) error {
		rc, err := zipFile.Open()
		if err != nil {
			return err
		}
		defer rc.Close()

		path := filepath.Clean(filepath.Join(dstDirName, filepath.FromSlash(zipFile.Name)))

		// Check for ZipSlip (Directory traversal)
		if !strings.HasPrefix(path, validPrefix) {
			return fmt.Errorf("illegal file path: %s", path)
		}

		if zipFile.FileInfo().IsDir() {
			// Dir
			os.MkdirAll(path, zipFile.Mode())
		} else {
			// File
			os.MkdirAll(filepath.Dir(path), zipFile.Mode())
			f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, zipFile.Mode())
			if err != nil {
				return err
			}
			defer f.Close()

			_, err = io.Copy(f, rc)
			if err != nil {
				return err
			}
		}
		return nil
	}

	for _, zipFile := range reader.File {
		err := extractAndWriteFile(zipFile)
		if err != nil {
			return err
		}
	}

	return nil
}

// Zip zips up files
func Zip(srcDir string, writer io.Writer, include []string) error {
	var err error
	srcDir, err = filepath.Abs(srcDir) // clean path
	if err != nil {
		return fmt.Errorf("Zip() filepath.Abs: %w", err)
	}

	// fixed patterns by appendding /** to dir include names
	// - dirToIncude/ => dirToInclude/**
	// - dirToIncude => dirToInclude/**
	var fixedPatterns []string
	for _, pattern := range include {
		stat, err := os.Stat(pattern)
		if err == nil {
			if stat.IsDir() {
				if strings.HasSuffix(pattern, "/") { //pattern ends with /, then append **
					pattern = pattern + "**"
				} else { // does not end in /, append /**
					pattern = pattern + "/**"
				}
			}
		}
		fixedPatterns = append(fixedPatterns, pattern)
	}
	fmt.Printf("%v\n", fixedPatterns)

	tw := zip.NewWriter(writer)
	defer tw.Close()

	err = RecurseDir(srcDir, func(fullFilename string, file *os.File) error {
		filename := strings.Replace(fullFilename, srcDir, "", 1) // /path/to/srcDir/dir/file => /dir/file
		filename = strings.TrimLeft(filename, "/")               // dir/file
		filename = filepath.ToSlash(filename)                    // replace windows filenames to *nix filenames: dir\file => dir/file

		includeFile := false
		for _, pattern := range fixedPatterns {
			includeFile = antMatcher.Match(pattern, filename)
			//includeFile, _ = path.Match(pattern, filename)
			if includeFile {
				break
			}
		}
		if !includeFile {
			return nil
		}
		fmt.Printf("- %s\n", filename)

		// Get FileInfo about our file providing file size, mode, etc.
		info, err := file.Stat()
		if err != nil {
			return fmt.Errorf("file.Stat(%s): %w", fullFilename, err)
		}

		// Create a tar Header from the FileInfo data
		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return fmt.Errorf("zip.FileInfoHeader(%s): %w", fullFilename, err)
		}
		header.Name = filename

		// Write file header to the tar archive
		w, err := tw.CreateHeader(header)
		if err != nil {
			return fmt.Errorf("zip.CreateHeader(%s): %w", filename, err)
		}

		// Copy contents if it is regular file
		if info.Mode().IsRegular() {
			_, err = io.Copy(w, file)
			if err != nil {
				return fmt.Errorf("io.Copy(%s, %s): %w", filename, fullFilename, err)
			}
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("Zip(): %w", err)
	}
	return nil
}

// RecurseDir loop throguh all files and directories recursively.  cb is called back
// with the name and the open file
func RecurseDir(absDir string, cb func(absName string, file *os.File) error) error {
	//
	diSlice, err := os.ReadDir(absDir)
	if err != nil {
		return fmt.Errorf("RecurseDir(): %w", err)
	}
	for _, di := range diSlice {
		var err error
		if di.IsDir() {
			// func for defer to work
			err = func() error {
				absName := filepath.Join(absDir, di.Name())
				file, err := os.Open(absName)
				if err != nil {
					return fmt.Errorf("RecurseDir() os.Open: %w", err)
				}
				defer file.Close()
				if err := cb(absName, file); err != nil {
					return fmt.Errorf("RecurseDir(): %w", err)
				}
				if err := RecurseDir(absName, cb); err != nil {
					return err
				}
				return nil
			}()
		} else {
			err = func() error {
				absName := filepath.Join(absDir, di.Name())
				file, err := os.Open(absName)
				if err != nil {
					return fmt.Errorf("RecurseDir() os.Open: %w", err)
				}
				defer file.Close()
				if err := cb(absName, file); err != nil {
					return err
				}
				return nil
			}()
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func Untgz(fileName, dir string) error {
	gzipStream, err := os.Open(fileName)
	if err != nil {
		return fmt.Errorf("Untgz: Opening file (%s): %w", fileName, err)
	}

	uncompressedStream, err := gzip.NewReader(gzipStream)
	if err != nil {
		return fmt.Errorf("Untgz: NewReader failed: %w", err)
	}

	tarReader := tar.NewReader(uncompressedStream)

	if !fileExists(dir) {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}

	for true {
		header, err := tarReader.Next()

		if err == io.EOF {
			break
		}

		if err != nil {
			return fmt.Errorf("Untgz: Next() failed: %w", err)
		}

		switch header.Typeflag {
		case tar.TypeDir:
			//fmt.Printf("%d %s\n", header.Mode, header.Name)
			realName, err := uncompressActualPath(dir, header.Name)
			if err != nil {
				return fmt.Errorf("untgz: filepath.abs() failed: %w", err)
			}

			//os.MkdirAll(path, zipFile.Mode())
			if err := os.MkdirAll(realName, os.FileMode(header.Mode)); err != nil {
				return fmt.Errorf("Untgz: Mkdir(%s) failed: %w", realName, err)
			}
		case tar.TypeReg:
			//fmt.Printf("%d %s\n", header.Mode, header.Name)
			realName, err := uncompressActualPath(dir, header.Name)
			if err != nil {
				return fmt.Errorf("untgz: filepath.abs() failed: %w", err)
			}

			outFile, err := os.OpenFile(realName, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, os.FileMode(header.Mode))
			if err != nil {
				return fmt.Errorf("Untgz: Create() failed: %w", err)
			}
			if _, err := io.Copy(outFile, tarReader); err != nil {
				return fmt.Errorf("Untgz: Copy() failed: %w", err)
			}
			outFile.Close()
		case tar.TypeSymlink:
			realName, err := uncompressActualPath(dir, header.Name)
			if err != nil {
				return fmt.Errorf("untgz: filepath.abs() failed: %w", err)
			}
			err = os.Symlink(header.Linkname, realName)
			if err != nil {
				return fmt.Errorf("Untgz: symlink failed: %w", err)
			}

		default:
			/*
				fmt.Fprintf(os.Stderr, "tar.TypeReg #%b#\n", tar.TypeReg)
				fmt.Fprintf(os.Stderr, "tar.TypeRegA #%b#\n", tar.TypeRegA)
				fmt.Fprintf(os.Stderr, "tar.TypeLink #%b#\n", tar.TypeLink)
				fmt.Fprintf(os.Stderr, "tar.TypeSymlink #%b#\n", tar.TypeSymlink)
				fmt.Fprintf(os.Stderr, "tar.TypeChar #%b#\n", tar.TypeChar)
				fmt.Fprintf(os.Stderr, "tar.TypeBlock #%b#\n", tar.TypeBlock)
				fmt.Fprintf(os.Stderr, "tar.TypeDir #%b#\n", tar.TypeDir)
				fmt.Fprintf(os.Stderr, "tar.TypeFifo #%b#\n", tar.TypeFifo)
				fmt.Fprintf(os.Stderr, "tar.TypeCont #%b#\n", tar.TypeCont)
				fmt.Fprintf(os.Stderr, "tar.TypeXHeader #%b#\n", tar.TypeXHeader)
				fmt.Fprintf(os.Stderr, "tar.TypeXGlobalHeader #%b#\n", tar.TypeXGlobalHeader)
				fmt.Fprintf(os.Stderr, "tar.TypeGNUSparse #%b#\n", tar.TypeGNUSparse)
				fmt.Fprintf(os.Stderr, "tar.TypeGNULongName #%b#\n", tar.TypeGNULongName)
				fmt.Fprintf(os.Stderr, "tar.TypeGNULongLink #%b#\n", tar.TypeGNULongLink)
			*/

			fmt.Fprintf(os.Stderr, "#%b#\n", tar.TypeDir)
			return fmt.Errorf("Untgz: unknown type: %b in %s ", header.Typeflag, header.Name)
		}

	}
	return nil
}

func uncompressActualPath(dir, path string) (string, error) {
	var err error
	realName := filepath.Clean(filepath.Join(dir, filepath.FromSlash(path)))
	if err != nil {
		return "", fmt.Errorf("Uncompress: filepath.Abs() failed: %w", err)
	}
	if !strings.HasPrefix(realName, dir) {
		return "", fmt.Errorf("Uncompress: path(%s) not contained within path(%s)", realName, dir)
	}
	return realName, nil
}

// isIonFile detects if file is ion
func isIonFile(f string) bool {
	ext := filepath.Ext(f)
	if strings.EqualFold(ext, ".ion") {
		return true
	} else if strings.EqualFold(ext, ".json") {
		return true
	} else if strings.EqualFold(ext, ".hcl") {
		return false
	}

	// try to get by detecting first character. if it starts with '{', then it is json
	if isIonContent(f) {
		return true
	}
	return false
}

func jsonLoad(f string, cfg any) error {
	b, err := os.ReadFile(f)
	if err != nil {
		return fmt.Errorf("unable to load JSON|ION file %s: %w", f, err)
	}
	return json.Unmarshal(b, cfg)
}

func ionLoad(f string, cfg any) error {
	b, err := os.ReadFile(f)
	if err != nil {
		return fmt.Errorf("unable to load JSON|ION file %s: %w", f, err)
	}
	return ion.Unmarshal(b, cfg)
}

func hclLoad(f string, cfg any) error {
	b, err := os.ReadFile(f)
	if err != nil {
		return fmt.Errorf("unable to load HCL file %s: %w", f, err)
	}

	var ctx *hcl.EvalContext //nil
	var file *hcl.File
	var diags hcl.Diagnostics

	file, diags = hclsyntax.ParseConfig(b, f, hcl.Pos{Line: 1, Column: 1})
	if diags.HasErrors() {
		return fmt.Errorf("Unable to parse HCL file %s: %w", f, diags)
	}

	diags = gohcl.DecodeBody(file.Body, ctx, cfg)
	if diags.HasErrors() {
		return fmt.Errorf("Unable to decode HCL file %s: %s", f, diags.Error())
	}

	return nil
}

func isIonContent(l string) bool {
	f, err := os.Open(l)
	if err != nil {
		return false
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	for sc.Scan() {
		txt := sc.Text()
		txt = strings.TrimSpace(txt)
		if txt == "" {
			continue
		}
		if strings.HasPrefix(txt, "//") {
			continue
		}

		// first line found that is not a space
		// or a commment, evaluate
		if strings.HasPrefix(txt, "{") {
			return true
		}
		return false
	}
	return false
}

type CircularDependencyDetector struct {
	stack []string
}

func NewCircularDependencyDetector() *CircularDependencyDetector {
	return &CircularDependencyDetector{
		stack: make([]string, 0, 10),
	}
}

func (c *CircularDependencyDetector) Push(item string) error {
	// Circular depedency protection
	for _, v := range c.stack {
		if item == v {
			return fmt.Errorf("Detected circular dependency: %s->%s\n", strings.Join(c.stack, "->"), item)
		}
	}
	c.stack = append(c.stack, item)
	return nil
}

func (c *CircularDependencyDetector) Clone() *CircularDependencyDetector {
	n := CircularDependencyDetector{}
	n.stack = make([]string, 0, cap(c.stack))
	for _, v := range c.stack {
		n.stack = append(n.stack, v)
	}
	return &n
}

// mapMerge merges m1 with the value of m2 without modifying the
// original map.   The values of m2 will override any duplicate
// values of m1
func mapMerge(m1, m2 map[string]string) map[string]string {
	m := make(map[string]string)
	for k, v := range m1 {
		m[k] = v
	}
	for k, v := range m2 {
		m[k] = v
	}
	return m
}

func mapJoin(m map[string]string, kvGlue, glue string) string {
	s := ""
	for k, v := range m {
		s += fmt.Sprintf("%s%s%s%s", k, kvGlue, v, glue)
	}
	return s
}

func toEnvKey(k string) string {
	k = strings.ToUpper(k)
	k = strings.ReplaceAll(k, ".", "_")
	k = strings.ReplaceAll(k, "-", "_")
	k = strings.ReplaceAll(k, "/", "_")
	return k
}
func toPropKey(k string) string {
	k = strings.ToLower(k)
	k = strings.ReplaceAll(k, "_", ".")
	k = strings.ReplaceAll(k, "-", ".")
	k = strings.ReplaceAll(k, "/", ".")
	return k
}

func execCommand(ed *ResolvedDependency, args []string) int {
	// env vars
	newPath, newEnv := ed.Resolve()

	// Set the path
	path := os.Getenv("PATH") // get original path
	pathParts := strings.Split(path, string([]rune{os.PathListSeparator}))
	newPath = append(newPath, pathParts...)
	newEnv["PATH"] = strings.Join(newPath, string([]rune{os.PathListSeparator}))

	// Args
	args, err := ed.ExpandCommand(args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}

	//executeCommand(args, newEnv)
	//debug.Printf("env: \n%s\n", mapJoin(env, "=", "\n"))
	Debug.Printf("command: %s", strings.Join(args, " "))

	// Set environment
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

// jsonDecode decodes json and returns pointer of R type passed
// e.g.1:
//
//	myTypePtr, err := jsonDecode(`{"fld1": "Val1"}`, MyType{})
//
// e.g.2:
//
//	mapPtr, err := jsonDecode(`{"fld1": "Val1"}`, make(map[string]string))
//	m := *mapPtr
//	fmt.Println(m["fld1"])
func jsonDecode[T string | []byte](b T, v any) error {
	switch tmp := any(b).(type) {
	case string:
		//try(json.Unmarshal([]byte(tmp), &v))
		err := json.Unmarshal([]byte(tmp), &v)
		return err
	case []byte:
		err := json.Unmarshal(tmp, &v)
		return err
	}
	return fmt.Errorf("Unknown type parameter")
}
