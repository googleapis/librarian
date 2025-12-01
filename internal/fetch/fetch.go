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
	"archive/tar"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var (
	errChecksumMismatch = errors.New("checksum mismatch")
	defaultBackoff      = 10 * time.Second
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

// DownloadTarball downloads a tarball from the given url to the target
// path, verifying its SHA256 checksum matches expectedSha256. It retries up to
// 3 times with exponential backoff on failure.
func DownloadTarball(ctx context.Context, target, url, expectedSha256 string) error {
	if fileExists(target) {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
		return err
	}
	tempFile, err := os.CreateTemp(filepath.Dir(target), "temp-")
	if err != nil {
		return err
	}
	tempPath := tempFile.Name()
	defer func() {
		tempFile.Close()
		cerr := os.Remove(tempPath)
		if err == nil && cerr != nil && !os.IsNotExist(cerr) {
			err = cerr
		}
	}()

	if err := downloadTarball(ctx, tempPath, url); err != nil {
		return err
	}

	sha, err := computeSHA256(tempPath)
	if err != nil {
		return err
	}
	if sha != expectedSha256 {
		return fmt.Errorf("%w: expected=%s, got=%s", errChecksumMismatch, expectedSha256, sha)
	}
	if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
		return err
	}
	return os.Rename(tempPath, target)
}

// downloadTarball downloads a tarball from the given source URL to the target
// path. It retries up to 3 times with exponential backoff on failure.
func downloadTarball(ctx context.Context, target, source string) error {
	var err error
	for i := range 3 {
		if i > 0 {
			select {
			case <-time.After(defaultBackoff):
				defaultBackoff *= 2
			case <-ctx.Done():
				return ctx.Err()
			}
		}

		if err := downloadAttempt(ctx, target, source); err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				return err
			}
			continue
		}
		return nil

	}
	return fmt.Errorf("download failed after 3 attempts, last error=%w", err)
}

func downloadAttempt(ctx context.Context, target, source string) (err error) {
	file, err := os.Create(target)
	if err != nil {
		return err
	}
	defer func() {
		cerr := file.Close()
		if err == nil {
			err = cerr
		}
		if err != nil {
			os.Remove(target)
		}
	}()

	client := http.Client{Timeout: 60 * time.Second}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, source, nil)
	if err != nil {
		return err
	}
	response, err := client.Do(req)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode >= 300 {
		return fmt.Errorf("http error in download %s", response.Status)
	}

	if _, err := io.Copy(file, response.Body); err != nil {
		return err
	}

	return nil
}

func fileExists(name string) bool {
	stat, err := os.Stat(name)
	if err != nil {
		return false
	}
	return stat.Mode().IsRegular()
}

// ExtractTarball extracts a gzipped tarball to the specified directory,
// stripping the top-level directory prefix that GitHub adds to tarballs.
func ExtractTarball(tarballPath, destDir string) error {
	f, err := os.Open(tarballPath)
	if err != nil {
		return err
	}
	defer f.Close()

	gzr, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}

		// When GitHub creates a tarball archive of a repository, it wraps all
		// the files in a top-level directory named in the format
		// "{repo}-{commit}/". Remove the GitHub top-level "repo-<commit>/"
		// prefix.
		name := hdr.Name
		parts := strings.SplitN(name, "/", 2)
		if len(parts) == 2 {
			name = parts[1]
		} else {
			continue
		}

		target := filepath.Join(destDir, name)
		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0755); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return err
			}

			out, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR|os.O_TRUNC, os.FileMode(hdr.Mode))
			if err != nil {
				return err
			}
			if _, err := io.Copy(out, tr); err != nil {
				out.Close()
				return err
			}
			out.Close()
		}
	}
}
