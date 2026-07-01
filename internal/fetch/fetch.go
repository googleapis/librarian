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
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/googleapis/librarian/internal/cache"
)

const (
	// DefaultBranchMaster represents the default git branch "master".
	DefaultBranchMaster = "master"
	maxDownloadRetries  = 3
)

var (
	errAbsSymlinks         = errors.New("absolute symlinks are not allowed")
	errChecksumMismatch    = errors.New("checksum mismatch")
	errMissingSHA256       = errors.New("must provide expected SHA256")
	errSymlinkEscape       = errors.New("symlinks are not allowed to escape destination")
	errUnsupportedFileType = errors.New("unsupported file type")
	defaultBackoff         = 10 * time.Second
)

// Endpoints defines the endpoints used to access GitHub.
type Endpoints struct {
	// API defines the endpoint used to make API calls.
	API string

	// Download defines the endpoint to download tarballs.
	Download string
}

// RepoRef represents a GitHub repository name.
type RepoRef struct {
	// Branch is the name of the repository branch, such as `master` or `preview`.
	Branch string

	// Org defines the GitHub organization (or user), that owns the repository.
	Org string

	// Name is the name of the repository, such as `googleapis` or `google-cloud-rust`.
	Name string
}

// Repo downloads a repository tarball and returns the path to the extracted
// directory.
//
// The cache directory is determined by LIBRARIAN_CACHE environment variable,
// or defaults to $HOME/.cache/librarian if not set.
//
// The diagrams below explains the structure of the librarian cache. For each
// path, $repo is a repository path (i.e. github.com/googleapis/googleapis),
// and $commit is a commit hash in that repository.
//
// Cache structure:
//
//	$LIBRARIAN_CACHE/
//	├── download/                    # Downloaded artifacts
//	│   └── $repo@$commit.tar.gz     # Source tarball (kept for re-extraction)
//	└── $repo@$commit/               # Extracted source files
//	    └── {files...}
//
// Example for github.com/googleapis/googleapis at commit abc123:
//
//	$HOME/.cache/librarian/
//	├── download/
//	│   └── github.com/googleapis/googleapis@abc123.tar.gz
//	└── github.com/googleapis/googleapis@abc123/
//	    └── google/
//	        └── api/
//	            └── annotations.proto
//
// Cache lookup order:
//  1. Check if extracted directory exists and contains files. If so, return it.
//  2. Check if tarball exists. Verify its SHA256 matches expectedSHA256. If yes,
//     extract tarball and return the directory. If the hash mismatches, fall
//     through to step 3.
//  3. Download tarball, compute SHA256, verify it matches expectedSHA256 from
//     librarian.yaml, extract, and return the path.
func Repo(ctx context.Context, repo, commit, expectedSHA256 string) (string, error) {
	cacheDir, err := cache.Directory()
	if err != nil {
		return "", err
	}

	tgz := tarballPath(cacheDir, repo, commit)
	outDir := filepath.Join(cacheDir, fmt.Sprintf("%s@%s", repo, commit))

	// Step 1: Check if extracted directory exists and contains files.
	if cached, err := extractedDir(cacheDir, repo, commit); err == nil {
		return cached, nil
	}

	// Step 2: Check if tarball exists. Verify its SHA256 matches expectedSHA256.
	// If hash doesn't match or any error happens during the extraction, delete
	// the tarball and fall through to re-download.
	if _, err := os.Stat(tgz); err == nil {
		sha, err := computeSHA256(tgz)
		if err == nil {
			if sha == expectedSHA256 {
				if err := os.MkdirAll(outDir, 0755); err != nil {
					return "", fmt.Errorf("failed creating %q: %w", outDir, err)
				}
				if err := ExtractTarball(tgz, outDir, stripTopLevelDir); err == nil {
					return outDir, nil
				}
			}
			if err := os.Remove(tgz); err != nil {
				return "", fmt.Errorf("failed to remove %q: %w", tgz, err)
			}
		}
	}

	// Step 3: Download tarball, compute SHA256, verify against expected, extract.
	sourceURL := fmt.Sprintf("https://%s/archive/%s.tar.gz", repo, commit)
	if err := os.MkdirAll(filepath.Dir(tgz), 0755); err != nil {
		return "", fmt.Errorf("failed creating %q: %w", filepath.Dir(tgz), err)
	}
	if err := os.MkdirAll(outDir, 0755); err != nil {
		return "", fmt.Errorf("failed creating %q: %w", outDir, err)
	}
	if err := Download(ctx, tgz, sourceURL, expectedSHA256); err != nil {
		return "", err
	}
	if err := ExtractTarball(tgz, outDir, stripTopLevelDir); err != nil {
		return "", fmt.Errorf("failed to extract tarball: %w", err)
	}
	return outDir, nil
}

// tarballPath returns the path to a cached tarball for the given repo and
// commit.
//
// The returned path has the format
// $LIBRARIAN_CACHE/download/$repo@$commit.tar.gz.
func tarballPath(cacheDir, repo, commit string) string {
	downloadDir := filepath.Join(cacheDir, "download", filepath.Dir(repo))
	return filepath.Join(downloadDir, fmt.Sprintf("%s@%s.tar.gz", filepath.Base(repo), commit))
}

// extractedDir returns the directory containing the extracted files for the
// given repo and commit. It validates that the directory exists and contains
// files.
//
// The returned path has the format $LIBRARIAN_CACHE/$repo@$commit/.
func extractedDir(cacheDir, repo, commit string) (string, error) {
	dir := filepath.Join(cacheDir, fmt.Sprintf("%s@%s", repo, commit))
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", fmt.Errorf("directory %q does not exist or is empty: %w", dir, err)
	}
	if len(entries) == 0 {
		return "", fmt.Errorf("directory %q does not exist or is empty", dir)
	}
	return dir, nil
}

// computeSHA256 computes the SHA256 checksum of a file and returns it as a hex
// string.
func computeSHA256(filePath string) (string, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, f); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", hasher.Sum(nil)), nil
}

// repoFromArchiveLink extracts the GitHub account and repository (such as
// `googleapis/googleapis`, or `googleapis/google-cloud-rust`) from an archive
// link.
// Note: This does **not** set [RepoRef.Branch] as it is not derivable from a
// commit-based archive URL.
func repoFromArchiveLink(githubDownload, archiveLink string) (*RepoRef, error) {
	urlPath := strings.TrimPrefix(archiveLink, githubDownload)
	urlPath = strings.TrimPrefix(urlPath, "/")
	components := strings.Split(urlPath, "/")
	if len(components) < 2 {
		return nil, fmt.Errorf("invalid archive URL %q", archiveLink)
	}
	repo := &RepoRef{
		Org:  components[0],
		Name: components[1],
	}
	return repo, nil
}

// urlSha256 downloads the content from the given URL and returns its SHA256
// checksum as a hex string.
func urlSha256(query string) (string, error) {
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

// latestSha fetches the latest commit SHA from the GitHub API for the given
// repository URL.
func latestSha(query string) (string, error) {
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

// LatestCommitAndChecksum fetches the latest commit SHA and the SHA256 of the tarball for that
// commit from the GitHub API for the given repository.
func LatestCommitAndChecksum(endpoints *Endpoints, repo *RepoRef) (commit, sha256 string, err error) {
	apiURL := fmt.Sprintf("%s/repos/%s/%s/commits/%s", endpoints.API, repo.Org, repo.Name, repo.Branch)
	commit, err = latestSha(apiURL)
	if err != nil {
		return "", "", err
	}

	tarballURL := tarballLink(endpoints.Download, repo, commit)
	sha256, err = urlSha256(tarballURL)
	if err != nil {
		return "", "", err
	}
	return commit, sha256, nil
}

// tarballLink constructs a GitHub tarball download URL for the given
// repository and commit SHA.
// Note: This does **not** incorporate the [RepoRef.Branch] as this produces a
// commit-based archive URL.
func tarballLink(githubDownload string, repo *RepoRef, sha string) string {
	return fmt.Sprintf("%s/%s/%s/archive/%s.tar.gz", githubDownload, repo.Org, repo.Name, sha)
}

// Download downloads a file from the given url to the target path, verifying
// its SHA256 checksum matches expectedSHA256. It retries up to
// maxDownloadRetries times with exponential backoff on failure.
func Download(ctx context.Context, target, url, expectedSHA256 string) error {
	if fileExists(target) {
		return nil
	}
	if expectedSHA256 == "" {
		return errMissingSHA256
	}
	if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
		return err
	}
	tempFile, err := os.CreateTemp(filepath.Dir(target), "temp-")
	if err != nil {
		return err
	}
	tempPath := tempFile.Name()
	_ = tempFile.Close()
	defer func() {
		cerr := os.Remove(tempPath)
		if err == nil && cerr != nil && !errors.Is(cerr, fs.ErrNotExist) {
			err = cerr
		}
	}()

	if err := downloadFile(ctx, tempPath, url); err != nil {
		return err
	}
	sha, err := computeSHA256(tempPath)
	if err != nil {
		return err
	}
	if sha != expectedSHA256 {
		return fmt.Errorf("%w: expected=%s, got=%s", errChecksumMismatch, expectedSHA256, sha)
	}
	return os.Rename(tempPath, target)
}

// downloadFile downloads a file from the given source URL to the target path.
// It retries up to maxDownloadRetries times with exponential backoff on failure.
func downloadFile(ctx context.Context, target, source string) error {
	var err error
	for i := range maxDownloadRetries {
		if i > 0 {
			select {
			case <-time.After(defaultBackoff):
				defaultBackoff *= 2
			case <-ctx.Done():
				return ctx.Err()
			}
		}

		if err = downloadAttempt(ctx, target, source); err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				return err
			}
			continue
		}
		return nil
	}
	return fmt.Errorf("download failed after %d attempts, last error=%w", maxDownloadRetries, err)
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

	client := http.Client{Timeout: 5 * time.Minute}
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

// stripTopLevelDir removes the top-level directory prefix (such as "{repo}-{commit}/")
// that GitHub automatically adds to repository archive entries.
func stripTopLevelDir(name string) (string, bool) {
	parts := strings.SplitN(name, "/", 2)
	if len(parts) == 2 {
		return parts[1], true
	}
	return "", false
}

// ExtractTarball extracts a gzipped tarball to the specified directory.
func ExtractTarball(tarballPath, destDir string, filter func(string) (string, bool)) error {
	if filter == nil {
		filter = func(name string) (string, bool) { return name, true }
	}
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

		name, ok := filter(hdr.Name)
		if !ok {
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
		case tar.TypeSymlink:
			// Ensure the symlink target does not escape the destination directory.
			linkTarget := hdr.Linkname
			if filepath.IsAbs(linkTarget) {
				return fmt.Errorf("%w: %s", errAbsSymlinks, linkTarget)
			}
			var resolvedTarget string
			resolvedTarget = filepath.Join(filepath.Dir(target), linkTarget)
			relLink, err := filepath.Rel(destDir, resolvedTarget)
			if err != nil || strings.Contains(relLink, "..") {
				return fmt.Errorf("%w: symlink target %q escapes destination directory %q", errSymlinkEscape, linkTarget, destDir)
			}
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return err
			}
			if err := os.Symlink(linkTarget, target); err != nil {
				return err
			}
		default:
			return fmt.Errorf("%w: %v", errUnsupportedFileType, hdr.Typeflag)
		}
	}
}
