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
	"runtime"
	"strings"

	"github.com/Masterminds/semver"
	"github.com/google/go-github/v47/github"

	"golang.org/x/oauth2"
)

var (
	githubClientMap map[string]*github.Client = make(map[string]*github.Client)
)

type GithubResolver struct {
}

func NewGithubResolver() *GithubResolver {
	return &GithubResolver{}
}

func (o *GithubResolver) String() string {
	return "GithubResolver{}"
}

func (o *GithubResolver) ResolveCoord(c *FuzzyCoord) (*LockedCoord, error) {
	Debug.Printf("Start GithubResolver.ResolveCoord(%s)", c)

	//
	ctx := context.Background()
	client := newGithubClient(c.Server)

	//
	// query github
	// latest if set to latest vLATEST
	// resolve for precise tag v1.2.3.4 -> v1.2.3.4
	// resolve for v1.2 -> v.1.2.3.4
	var release *github.RepositoryRelease
	var err error
	if c.Version == "" || c.Version == "0" {
		Debug.Printf(" | call client.Repositories.GetLatestRelease(%s, %s)", c.Owner, c.Repo)
		release, _, err = client.Repositories.GetLatestRelease(ctx, c.Owner, c.Repo)
	} else {
		// try to get exact tag
		var r *github.Response
		Debug.Printf(" | call client.Repositories.GetReleaseByTag (%s, %s, %s)", c.Owner, c.Repo, fmt.Sprintf("v%s", c.Version))
		release, r, err = client.Repositories.GetReleaseByTag(ctx, c.Owner, c.Repo, fmt.Sprintf("v%s", c.Version))
		if err != nil && r.StatusCode == http.StatusNotFound {
			release, err = ghFindReleaseByPattern(client, c.Owner, c.Repo, fmt.Sprintf("v%s", c.Version))
		}
	}
	if err != nil {
		return nil, fmt.Errorf("GithubResolver.ResolveCoord(): %w", err)
	}

	//
	version := NewVersion(release.GetName())
	if err != nil {
		return nil, fmt.Errorf("GithubResolver.ResolveCoord() NewVersion: %w", err)
	}

	return &LockedCoord{
		Server:  c.Server,
		Owner:   c.Owner,
		Repo:    c.Repo,
		Version: version,
	}, nil
}

func (o *GithubResolver) DownloadResolvedCoord(rc *LockedCoord, dir string) (string, error) {
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
		return "", fmt.Errorf("GithubResolver.DownloadResolvedCoord(): %w", err)
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

func (o *GithubResolver) getAssetFromRelease(c *LockedCoord, release *github.RepositoryRelease) (*github.ReleaseAsset, error) {
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

func ghFindReleaseByPattern(client *github.Client, owner, repo, patternStr string) (*github.RepositoryRelease, error) {
	Debug.Printf("ghFindReleaseByPattern(%s, %s, %s)", owner, repo, patternStr)
	// if not found search for it
	perPage := 30
	page := 1
	var latest *github.RepositoryRelease
	var releases []*github.RepositoryRelease
	var resp *github.Response
	var err error

	pattern := NewVersionPattern(patternStr)

	for resp == nil || resp.NextPage != 0 {
		Debug.Printf(" | call client.Repositories.ListReleases %s/%s/%d/%d", owner, repo, page, perPage)
		releases, resp, err = client.Repositories.ListReleases(
			context.Background(),
			owner,
			repo,
			&github.ListOptions{Page: page, PerPage: perPage},
		)
		if err != nil {
			return nil, fmt.Errorf("ghFindReleaseByPattern(): %w", err)
		}
		for _, release := range releases {
			if release.GetDraft() {
				continue
			}
			Debug.Printf(" || '%s'.matches(%s)", patternStr, release.GetName())
			if pattern.Matches(NewVersion(release.GetName())) &&
				(latest == nil || versionCompare(release.GetName(), latest.GetName()) > 1) {
				Debug.Printf(" || found %s", release.GetName())
				latest = release
			}
		}

		page = resp.NextPage
	}

	if latest != nil {
		Debug.Println(" | returning ", latest.GetName())
		return latest, nil
	}
	return nil, fmt.Errorf("dependency %s/%s-%s not found", owner, repo, patternStr)
}

func possibleAssetNames(c *LockedCoord) []BzAsset {
	osArch := fmt.Sprintf("%s-%s", runtime.GOOS, runtime.GOARCH)

	extensions := []string{"zip", "tgz", "tar.gz"} // possible extensions

	var res []BzAsset
	for _, ext := range extensions {
		res = append(res,
			BzAsset{Canonical: fmt.Sprintf("%s-%s-v%s", c.Repo, osArch, c.Version.Canonical()), Ext: ext}, // openjdk-linux-amd64-v1.2.3.zip
			BzAsset{Canonical: fmt.Sprintf("%s-v%s", c.Repo, c.Version.Canonical()), Ext: ext},            // openjdk-v1.2.3.zip
			BzAsset{Canonical: c.Repo, Ext: ext},                                                          // openjdk.zip
		)
	}
	return res
}

type BzAsset struct {
	Ext       string // zip
	Canonical string // project-name-linux-amd64-v1.2.3
}

func (a *BzAsset) NameWithExt() string {
	return fmt.Sprintf("%s.%s", a.Canonical, a.Ext)
}

type BzAssetArrHelper []BzAsset

func (o BzAssetArrHelper) CollectNames() []string {
	var names []string
	for _, i := range o {
		names = append(names, i.NameWithExt())
	}
	return names
}

func versionCompare(a, b string) int {
	v1, err := semver.NewVersion(a)
	if err != nil {
		Warn.Printf("invalid semver `%s`", a)
		return 0
	}
	v2, err := semver.NewVersion(b)
	if err != nil {
		Warn.Printf("invalid semver `%s`", b)
		return 0
	}

	return v1.Compare(v2)
}

func newGithubClient(server string) *github.Client {
	if client, ok := githubClientMap[server]; ok {
		return client
	}

	// github client
	ctx := context.Background()
	githubAccessToken := bzUserConfig.GetServerToken(server)
	var tc *http.Client = nil
	if githubAccessToken != "" {
		ts := oauth2.StaticTokenSource(
			&oauth2.Token{AccessToken: githubAccessToken},
		)
		tc = oauth2.NewClient(ctx, ts)
	}
	client := github.NewClient(tc)
	githubClientMap[server] = client
	return client
}
