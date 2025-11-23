// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package fetch provides utilities for fetching GitHub repository metadata and computing checksums.
package fetch

import (
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Endpoints defines the endpoints used to access GitHub.
type Endpoints struct {
	// API defines the endpoint used to make API calls.
	API string

	// Download defines the endpoint to download tarballs.
	Download string
}

// Repo represents a GitHub repository name.
type Repo struct {
	// Org defines the GitHub organization (or user), that owns the repository.
	Org string

	// Repo is the name of the repository, such as `googleapis` or `google-cloud-rust`.
	Repo string
}

// RepoFromTarballLink extracts the gitHub account and repository (such as
// `googleapis/googleapis`, or `googleapis/google-cloud-rust`) from the tarball
// link.
func RepoFromTarballLink(githubDownload, tarballLink string) (*Repo, error) {
	urlPath := strings.TrimPrefix(tarballLink, githubDownload)
	urlPath = strings.TrimPrefix(urlPath, "/")
	components := strings.Split(urlPath, "/")
	if len(components) < 2 {
		return nil, fmt.Errorf("url path for tarball link is missing components")
	}
	repo := &Repo{
		Org:  components[0],
		Repo: components[1],
	}
	return repo, nil
}

// Sha256 downloads the content from the given URL and returns its SHA256
// checksum as a hex string.
func Sha256(query string) (string, error) {
	response, err := http.Get(query)
	if err != nil {
		return "", err
	}
	if response.StatusCode >= 300 {
		return "", fmt.Errorf("http error in download %s", response.Status)
	}
	defer response.Body.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, response.Body); err != nil {
		return "", err
	}
	got := fmt.Sprintf("%x", hasher.Sum(nil))
	return got, nil
}

// LatestSha fetches the latest commit SHA from the GitHub API for the given
// repository URL.
func LatestSha(query string) (string, error) {
	client := &http.Client{}
	request, err := http.NewRequest(http.MethodGet, query, nil)
	if err != nil {
		return "", err
	}
	request.Header.Set("Accept", "application/vnd.github.VERSION.sha")
	response, err := client.Do(request)
	if err != nil {
		return "", err
	}
	if response.StatusCode >= 300 {
		return "", fmt.Errorf("http error in download %s", response.Status)
	}
	defer response.Body.Close()
	contents, err := io.ReadAll(response.Body)
	if err != nil {
		return "", err
	}
	return string(contents), nil
}

// TarballLink constructs a GitHub tarball download URL for the given
// repository and commit SHA.
func TarballLink(githubDownload string, repo *Repo, sha string) string {
	return fmt.Sprintf("%s/%s/%s/archive/%s.tar.gz", githubDownload, repo.Org, repo.Repo, sha)
}

// DownloadTarball downloads a tarball from the given source URL to the target path,
// verifying its SHA256 checksum matches expectedSha256. It retries up to 3 times
// with exponential backoff on failure.
func DownloadTarball(target, source, expectedSha256 string) error {
	if fileExists(target) {
		return nil
	}
	var err error
	backoff := 10 * time.Second
	for i := range 3 {
		if i != 0 {
			time.Sleep(backoff)
			backoff = 2 * backoff
		}
		if err = downloadAttempt(target, source, expectedSha256); err == nil {
			return nil
		}
	}
	return fmt.Errorf("download failed after 3 attempts, last error=%w", err)
}

func downloadAttempt(target, source, expectedSha256 string) error {
	if err := os.MkdirAll(filepath.Dir(target), 0777); err != nil {
		return err
	}
	tempFile, err := os.CreateTemp(filepath.Dir(target), "temp-")
	if err != nil {
		return err
	}
	defer func() {
		tempFile.Close()
		os.Remove(tempFile.Name())
	}()

	hasher := sha256.New()
	writer := io.MultiWriter(tempFile, hasher)

	response, err := http.Get(source)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode >= 300 {
		return fmt.Errorf("http error in download %s", response.Status)
	}

	if _, err := io.Copy(writer, response.Body); err != nil {
		return err
	}
	if err := tempFile.Close(); err != nil {
		return err
	}

	got := fmt.Sprintf("%x", hasher.Sum(nil))
	if expectedSha256 != got {
		return fmt.Errorf("mismatched hash on download, expected=%s, got=%s", expectedSha256, got)
	}
	return os.Rename(tempFile.Name(), target)
}

func fileExists(name string) bool {
	stat, err := os.Stat(name)
	if err != nil {
		return false
	}
	return stat.Mode().IsRegular()
}
