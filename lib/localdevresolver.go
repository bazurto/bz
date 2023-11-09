// SPDX-FileCopyrightText: 2023 RH America LLC <info@rhamerica.com>
// SPDX-License-Identifier: GPL-3.0-only

package lib

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/go-github/v47/github"
)

type LocalDevResolver struct {
}

func NewLocalDevResolver() *LocalDevResolver {
	return &LocalDevResolver{}
}

func (o *LocalDevResolver) String() string {
	return "LocalDevResolver{}"
}

func (o *LocalDevResolver) ResolveCoord(c *FuzzyCoord) (*LockedCoord, error) {
	Debug.Printf("Start LocalDevResolver.ResolveCoord(%s)", c)

	return &LockedCoord{
		Server:  c.Server,
		Owner:   c.Owner,
		Repo:    c.Repo,
		Version: version,
	}, nil
}

func (o *LocalDevResolver) DownloadResolvedCoord(rc *LockedCoord, dir string) (string, error) {
	Debug.Printf("Start DownloadResolvedCoord(%v, %s)", rc, dir)
	//
	ctx := context.Background()
	client := newGithubClient(rc.Server)

	//
	githubVersion := fmt.Sprintf("v%s", rc.Version.Canonical())
	release, _, err := client.Repositories.GetReleaseByTag(ctx, rc.Owner, rc.Repo, githubVersion)
	if err != nil {
		return "", err
	}

	// Get the asset name that we should download in the priority order of possible asset names function
	asset, err := o.getAssetFromRelease(rc, release)
	if err != nil {
		return "", fmt.Errorf("LocalDevResolver.DownloadResolvedCoord(): %w", err)
	}

	// download file
	if err := mkdirIfNotExists(dir); err != nil {
		return "", err
	} else {
		Debug.Printf("dir already exists: %s", dir)
	}

	file := filepath.Join(dir, asset.GetName())
	downloadFileTmp := fmt.Sprintf("%s.tmp", file)
	w, err := os.Create(downloadFileTmp)
	if err != nil {
		return "", err
	}
	defer w.Close()

	//
	readCloser, _, err := client.Repositories.DownloadReleaseAsset(ctx, rc.Owner, rc.Repo, asset.GetID(), http.DefaultClient)
	if err != nil {
		return "", err
	}
	defer readCloser.Close()
	Info.Printf("Downloading file %s ...", file)
	if _, err := io.Copy(w, readCloser); err != nil {
		return "", err
	}

	// rename tmp download file to downloadFile
	if err := os.Rename(downloadFileTmp, file); err != nil {
		return "", err
	} else {
		Info.Printf("Downloading file %s DONE", file)
	}

	return file, nil
}

func (o *LocalDevResolver) getAssetFromRelease(c *LockedCoord, release *github.RepositoryRelease) (*github.ReleaseAsset, error) {
	var asset *github.ReleaseAsset
	expectedNames := possibleAssetNames(c)
	for _, expected := range expectedNames {
		for _, a := range release.Assets {
			//Debug.Printf(" | is %s == %s", expected.NameWithExt(), a.GetName())
			if expected.NameWithExt() == a.GetName() {
				Debug.Printf("found asset : %s", a.GetName())
				asset = a
				break
			}
		}
	}
	if asset == nil {
		return nil, fmt.Errorf(
			"could not find asset %s in depedency [%s]",
			strings.Join(BzAssetArrHelper(expectedNames).CollectNames(), ","),
			c.String(),
		)
	}
	return asset, nil
}
