// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package fetch

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

const envLibrarianCache = "LIBRARIAN_CACHE"

// Info contains metadata about a downloaded tarball.
type Info struct {
	SHA256 string `json:"sha256"`
}

// RepoDir downloads a repository tarball and returns the path to the extracted
// directory.
//
// The cache directory is determined by LIBRARIAN_CACHE environment variable,
// or defaults to $HOME/.librarian if not set.
//
// The diagrams below explains the structure of the librarian cache. For each
// path, $repo is a repository path (i.e. github.com/googleapis/googleapis),
// and $commit is a commit hash in that repository.
//
// Cache structure:
//
//	$LIBRARIAN_CACHE/
//	├── download/                  # Downloaded artifacts
//	│   └── $repo@$commit.tar.gz   # Source tarball
//	│   └── $repo@$commit.info     # Metadata (SHA256)
//	└── $repo@$commit/             # Extracted source files
//	    └── {files...}
//
// Example for github.com/googleapis/googleapis at commit abc123:
//
//	$HOME/.librarian/
//	├── download/
//	│   └── github.com/googleapis/googleapis@abc123.tar.gz
//	│   └── github.com/googleapis/googleapis@abc123.info
//	└── github.com/googleapis/googleapis@abc123/
//	    └── google/
//	        └── api/
//	            └── annotations.proto
//
// When downloading a repository, the cache is checked in this order:
//  1. Check if extracted directory exists and contains files. If so, return
//     that path.
//  2. Check if tarball exists with valid SHA256 in .info file. If so, extract
//     and return the path.
//  3. Download tarball, compute SHA256, save .info, extract, and return the
//     path.
func RepoDir(repo, commit string) (string, error) {
	cacheDir, err := cacheDir()
	if err != nil {
		return "", err
	}

	// Step 1: Check if extracted directory exists and contains files.
	extractedDir, err := extractDir(cacheDir, repo, commit)
	if err == nil {
		return extractedDir, nil
	}

	// Step 2: Check if tarball exists with valid SHA256, extract if valid.
	tarballPath := artifactPath(cacheDir, repo, commit, "tar.gz")
	infoPath := artifactPath(cacheDir, repo, commit, "info")
	extractedDir = filepath.Join(cacheDir, fmt.Sprintf("%s@%s", repo, commit))

	if _, err := os.Stat(tarballPath); err == nil {
		if err := extractTarball(extractedDir, tarballPath, infoPath); err == nil {
			return extractedDir, nil
		}
	}

	// Step 3: Download tarball, compute SHA256, save .info, extract.
	sourceURL := fmt.Sprintf("https://%s/archive/%s.tar.gz", repo, commit)
	if err := os.MkdirAll(filepath.Dir(tarballPath), 0755); err != nil {
		return "", fmt.Errorf("failed creating %q: %w", filepath.Dir(tarballPath), err)
	}
	if err := os.MkdirAll(extractedDir, 0755); err != nil {
		return "", fmt.Errorf("failed creating %q: %w", extractedDir, err)
	}
	sha, err := downloadTarball(tarballPath, sourceURL)
	if err != nil {
		return "", err
	}

	i := Info{SHA256: sha}
	b, err := json.MarshalIndent(i, "", "  ")
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(infoPath), 0755); err != nil {
		return "", fmt.Errorf("failed to create info dir: %w", err)
	}
	if err := os.WriteFile(infoPath, b, 0644); err != nil {
		return "", fmt.Errorf("failed to write .info file: %w", err)
	}
	if err := ExtractTarball(tarballPath, extractedDir); err != nil {
		return "", fmt.Errorf("failed to extract tarball: %w", err)
	}
	return extractedDir, nil
}

// extractTarball validates and extracts a cached tarball if it exists with a
// valid SHA256. It returns an error if validation fails or extraction errors.
func extractTarball(extractedDir, tarballPath, infoPath string) error {
	b, err := os.ReadFile(infoPath)
	if err != nil {
		return err
	}

	var i Info
	if err := json.Unmarshal(b, &i); err != nil {
		return err
	}

	sum, err := computeSHA256(tarballPath)
	if err != nil {
		return err
	}
	if sum != i.SHA256 {
		return fmt.Errorf("SHA256 mismatch: expected %s, got %s", i.SHA256, sum)
	}

	if err := os.MkdirAll(extractedDir, 0755); err != nil {
		return fmt.Errorf("failed creating %q: %w", extractedDir, err)
	}
	if err := ExtractTarball(tarballPath, extractedDir); err != nil {
		return fmt.Errorf("failed to extract tarball: %w", err)
	}
	return nil
}

func computeSHA256(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

// cacheDir returns the root cache directory for librarian operations.
// It checks the $LIBRARIAN_CACHE environment variable, falling back to $HOME/.librarian/
// if not set.
func cacheDir() (string, error) {
	if cache := os.Getenv(envLibrarianCache); cache != "" {
		return cache, nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user home directory: %w", err)
	}
	return filepath.Join(home, ".librarian"), nil
}

// artifactPath returns the path to a cached download artifact for the given repo and commit.
// The suffix specifies the file type (e.g., "tar.gz", "info").
//
// The returned path has the format $LIBRARIAN_CACHE/download/$repo@$commit.$suffix.
func artifactPath(cacheDir, repo, commit, suffix string) string {
	downloadDir := filepath.Join(cacheDir, "download", filepath.Dir(repo))
	return filepath.Join(downloadDir, fmt.Sprintf("%s@%s.%s", filepath.Base(repo), commit, suffix))
}

// extractDir returns the directory containing the extracted files for the
// given repo and commit. It validates that the directory exists and contains
// files.
//
// The returned path has the format $LIBRARIAN_CACHE/$repo@$commit/.
func extractDir(cacheDir, repo, commit string) (string, error) {
	extractedDir := filepath.Join(cacheDir, fmt.Sprintf("%s@%s", repo, commit))

	entries, err := os.ReadDir(extractedDir)
	if err != nil {
		return "", fmt.Errorf("directory %q does not exist or is empty: %w", extractedDir, err)
	}
	if len(entries) == 0 {
		return "", fmt.Errorf("directory %q does not exist or is empty", extractedDir)
	}
	return extractedDir, nil
}
