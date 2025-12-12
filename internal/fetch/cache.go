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

package fetch

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

const envLibrarianCache = "LIBRARIAN_CACHE"

// RepoDir downloads a repository tarball and returns the path to the extracted
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
func RepoDir(ctx context.Context, repo, commit, expectedSHA256 string) (string, error) {
	cacheDir, err := cacheDir()
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
				if err := ExtractTarball(tgz, outDir); err == nil {
					return outDir, nil
				}
			}
			if err := os.Remove(tgz); err != nil {
				slog.Debug("failed to remove tarball", "path", tgz, "err", err)
			}
		}
	}

	// Step 3: Download tarball, compute SHA256, verify against expected, extract.
	sourceURL := ""
	if isVersionTag(commit) {
		sourceURL = fmt.Sprintf("https://%s/archive/refs/tags/%s.tar.gz", repo, commit)
	}
	sourceURL = fmt.Sprintf("https://%s/archive/%s.tar.gz", repo, commit)
	if err := os.MkdirAll(filepath.Dir(tgz), 0755); err != nil {
		return "", fmt.Errorf("failed creating %q: %w", filepath.Dir(tgz), err)
	}
	if err := os.MkdirAll(outDir, 0755); err != nil {
		return "", fmt.Errorf("failed creating %q: %w", outDir, err)
	}
	if err := DownloadTarball(ctx, tgz, sourceURL, expectedSHA256); err != nil {
		return "", err
	}
	if err := ExtractTarball(tgz, outDir); err != nil {
		return "", fmt.Errorf("failed to extract tarball: %w", err)
	}
	return outDir, nil
}

// cacheDir returns the root cache directory for librarian operations. It
// checks the $LIBRARIAN_CACHE environment variable, falling back to
// $HOME/.cache/librarian if not set.
func cacheDir() (string, error) {
	if cache := os.Getenv(envLibrarianCache); cache != "" {
		return cache, nil
	}

	home, err := os.UserCacheDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user cache directory: %w", err)
	}
	return filepath.Join(home, "librarian"), nil
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

// isVersionTag returns true if the commit string starts with 'v' and is followed by digits and dots.
func isVersionTag(commit string) bool {
	if !strings.HasPrefix(commit, "v") {
		return false
	}
	version := strings.TrimPrefix(commit, "v")
	for _, r := range version {
		if !('0' <= r && r <= '9' || r == '.') {
			return false
		}
	}
	return true
}
