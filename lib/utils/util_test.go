// SPDX-FileCopyrightText: 2023 RH America LLC <info@rhamerica.com>
// SPDX-License-Identifier: GPL-3.0-only

package utils

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUnzip_Basic(t *testing.T) {
	// Create a valid zip file
	tmpDir := t.TempDir()
	zipPath := filepath.Join(tmpDir, "test.zip")
	extractDir := filepath.Join(tmpDir, "extracted")

	f, err := os.Create(zipPath)
	require.NoError(t, err)

	w := zip.NewWriter(f)
	fw, err := w.Create("hello.txt")
	require.NoError(t, err)
	_, err = fw.Write([]byte("hello world"))
	require.NoError(t, err)
	require.NoError(t, w.Close())
	require.NoError(t, f.Close())

	err = Unzip(zipPath, extractDir)
	require.NoError(t, err)

	content, err := os.ReadFile(filepath.Join(extractDir, "hello.txt"))
	require.NoError(t, err)
	assert.Equal(t, "hello world", string(content))
}

func TestUnzip_ZipSlipPrevented(t *testing.T) {
	tmpDir := t.TempDir()
	zipPath := filepath.Join(tmpDir, "malicious.zip")
	extractDir := filepath.Join(tmpDir, "extracted")

	f, err := os.Create(zipPath)
	require.NoError(t, err)

	w := zip.NewWriter(f)
	// Create a file with path traversal
	fw, err := w.Create("../../etc/passwd")
	require.NoError(t, err)
	_, err = fw.Write([]byte("malicious content"))
	require.NoError(t, err)
	require.NoError(t, w.Close())
	require.NoError(t, f.Close())

	err = Unzip(zipPath, extractDir)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "illegal file path")
}

func TestUntgz_Basic(t *testing.T) {
	tmpDir := t.TempDir()
	tgzPath := filepath.Join(tmpDir, "test.tgz")
	extractDir := filepath.Join(tmpDir, "extracted")

	f, err := os.Create(tgzPath)
	require.NoError(t, err)

	gw := gzip.NewWriter(f)
	tw := tar.NewWriter(gw)

	content := []byte("hello from tgz")
	hdr := &tar.Header{
		Name: "hello.txt",
		Mode: 0644,
		Size: int64(len(content)),
	}
	require.NoError(t, tw.WriteHeader(hdr))
	_, err = tw.Write(content)
	require.NoError(t, err)

	require.NoError(t, tw.Close())
	require.NoError(t, gw.Close())
	require.NoError(t, f.Close())

	err = Untgz(tgzPath, extractDir)
	require.NoError(t, err)

	data, err := os.ReadFile(filepath.Join(extractDir, "hello.txt"))
	require.NoError(t, err)
	assert.Equal(t, "hello from tgz", string(data))
}

func TestUntgz_PathTraversalPrevented(t *testing.T) {
	tmpDir := t.TempDir()
	tgzPath := filepath.Join(tmpDir, "malicious.tgz")
	extractDir := filepath.Join(tmpDir, "extracted")

	f, err := os.Create(tgzPath)
	require.NoError(t, err)

	gw := gzip.NewWriter(f)
	tw := tar.NewWriter(gw)

	content := []byte("malicious content")
	hdr := &tar.Header{
		Name: "../../etc/passwd",
		Mode: 0644,
		Size: int64(len(content)),
	}
	require.NoError(t, tw.WriteHeader(hdr))
	_, err = tw.Write(content)
	require.NoError(t, err)

	require.NoError(t, tw.Close())
	require.NoError(t, gw.Close())
	require.NoError(t, f.Close())

	err = Untgz(tgzPath, extractDir)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not contained within")
}

func TestUncompressActualPath_Safe(t *testing.T) {
	dir := "/tmp/extract"
	path, err := uncompressActualPath(dir, "subdir/file.txt")
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(dir, "subdir/file.txt"), path)
}

func TestUncompressActualPath_Traversal(t *testing.T) {
	dir := "/tmp/extract"
	_, err := uncompressActualPath(dir, "../../etc/passwd")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not contained within")
}

func TestUntgz_WithSubdirectories(t *testing.T) {
	tmpDir := t.TempDir()
	tgzPath := filepath.Join(tmpDir, "test.tgz")
	extractDir := filepath.Join(tmpDir, "extracted")

	f, err := os.Create(tgzPath)
	require.NoError(t, err)

	gw := gzip.NewWriter(f)
	tw := tar.NewWriter(gw)

	// Add directory
	require.NoError(t, tw.WriteHeader(&tar.Header{
		Name:     "subdir/",
		Mode:     0755,
		Typeflag: tar.TypeDir,
	}))

	// Add file in subdirectory
	content := []byte("nested file")
	require.NoError(t, tw.WriteHeader(&tar.Header{
		Name: "subdir/nested.txt",
		Mode: 0644,
		Size: int64(len(content)),
	}))
	_, err = tw.Write(content)
	require.NoError(t, err)

	require.NoError(t, tw.Close())
	require.NoError(t, gw.Close())
	require.NoError(t, f.Close())

	err = Untgz(tgzPath, extractDir)
	require.NoError(t, err)

	data, err := os.ReadFile(filepath.Join(extractDir, "subdir", "nested.txt"))
	require.NoError(t, err)
	assert.Equal(t, "nested file", string(data))
}

func TestUncompress_UnsupportedExtension(t *testing.T) {
	err := Uncompress("file.rar", "/tmp/out")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not supported")
}

func TestCircularDependencyDetector(t *testing.T) {
	cdd := NewCircularDependencyDetector()
	assert.NoError(t, cdd.Push("a"))
	assert.NoError(t, cdd.Push("b"))
	assert.NoError(t, cdd.Push("c"))

	err := cdd.Push("a")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "circular dependency")
}

func TestCircularDependencyDetector_Clone(t *testing.T) {
	cdd := NewCircularDependencyDetector()
	assert.NoError(t, cdd.Push("a"))
	assert.NoError(t, cdd.Push("b"))

	clone := cdd.Clone()
	assert.NoError(t, clone.Push("c"))

	// Original should not have "c"
	assert.NoError(t, cdd.Push("c"))
}

func TestFileExists(t *testing.T) {
	tmpDir := t.TempDir()

	f := filepath.Join(tmpDir, "exists.txt")
	require.NoError(t, os.WriteFile(f, []byte("hi"), 0644))

	assert.True(t, FileExists(f))
	assert.False(t, FileExists(filepath.Join(tmpDir, "nope.txt")))
}

func TestMkdirIfNotExists(t *testing.T) {
	tmpDir := t.TempDir()
	newDir := filepath.Join(tmpDir, "newdir")

	assert.False(t, FileExists(newDir))
	require.NoError(t, MkdirIfNotExists(newDir))
	assert.True(t, FileExists(newDir))

	// Calling again should be a no-op
	require.NoError(t, MkdirIfNotExists(newDir))
}

func TestToEnvKey(t *testing.T) {
	assert.Equal(t, "GITHUB_COM_BAZURTO_PYTHON", ToEnvKey("github.com/bazurto/python"))
	assert.Equal(t, "MY_VAR", ToEnvKey("my-var"))
}

func TestMapMerge(t *testing.T) {
	m1 := map[string]string{"a": "1", "b": "2"}
	m2 := map[string]string{"b": "3", "c": "4"}
	result := MapMerge(m1, m2)

	assert.Equal(t, "1", result["a"])
	assert.Equal(t, "3", result["b"]) // m2 overrides m1
	assert.Equal(t, "4", result["c"])

	// Originals unchanged
	assert.Equal(t, "2", m1["b"])
}
