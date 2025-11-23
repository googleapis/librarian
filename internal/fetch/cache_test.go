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
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

const (
	testCommit = "abc123"
	testRepo   = "github.com/googleapis/googleapis"
	testSHA256 = "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"

	testExtractedDir = "github.com/googleapis/googleapis@abc123/"
	testTarball      = "download/github.com/googleapis/googleapis@abc123.tar.gz"
)

func TestCacheDir(t *testing.T) {
	for _, test := range []struct {
		name    string
		env     string
		wantDir string
	}{
		{
			name:    "uses LIBRARIAN_CACHE when set",
			env:     "/custom/cache",
			wantDir: "/custom/cache",
		},
		{
			name: "uses UserCacheDir/librarian when LIBRARIAN_CACHE not set",
			env:  "",
			wantDir: func() string {
				cache, _ := os.UserCacheDir()
				return filepath.Join(cache, "librarian")
			}(),
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			if test.env != "" {
				t.Setenv("LIBRARIAN_CACHE", test.env)
			}
			got, err := cacheDir()
			if err != nil {
				t.Fatal(err)
			}

			if diff := cmp.Diff(test.wantDir, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestTarballPath(t *testing.T) {
	const cachedir = "/tmp/cache"

	got := tarballPath(cachedir, testRepo, testCommit)
	want := filepath.Join(cachedir, testTarball)
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestExtractedDir(t *testing.T) {
	cachedir := t.TempDir()
	want := filepath.Join(cachedir, testExtractedDir)
	if err := os.MkdirAll(want, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(want, "test.txt"), []byte("content"), 0644); err != nil {
		t.Fatal(err)
	}

	got, err := extractedDir(cachedir, testRepo, testCommit)
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestExtractDir_Empty(t *testing.T) {
	cachedir := t.TempDir()
	if _, err := extractedDir(cachedir, testRepo, testCommit); err == nil {
		t.Fatal("expected error for non-existent directory")
	}
}

func TestRepoDir_ExtractedDirExists(t *testing.T) {
	cachedir := t.TempDir()
	t.Setenv(envLibrarianCache, cachedir)

	extractedDir := filepath.Join(cachedir, testExtractedDir)
	if err := os.MkdirAll(extractedDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(extractedDir, "test.txt"), []byte("content"), 0644); err != nil {
		t.Fatal(err)
	}

	got, err := RepoDir(testRepo, testCommit, testSHA256)
	if err != nil {
		t.Fatal(err)
	}

	if diff := cmp.Diff(extractedDir, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestRepoDir_TarballExists(t *testing.T) {
	cachedir := t.TempDir()
	t.Setenv(envLibrarianCache, cachedir)

	tarballPath := filepath.Join(cachedir, testTarball)
	if err := os.MkdirAll(filepath.Dir(tarballPath), 0755); err != nil {
		t.Fatal(err)
	}

	tarballData := createTestTarball(t, "test-repo-abc123", map[string]string{
		"README.md": "# Test Repo",
		"main.go":   "package main",
	})
	if err := os.WriteFile(tarballPath, tarballData, 0644); err != nil {
		t.Fatal(err)
	}

	sha := fmt.Sprintf("%x", sha256.Sum256(tarballData))
	got, err := RepoDir(testRepo, testCommit, sha)
	if err != nil {
		t.Fatal(err)
	}

	extractedDir := filepath.Join(cachedir, testExtractedDir)
	if diff := cmp.Diff(extractedDir, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
	if _, err := os.Stat(filepath.Join(got, "README.md")); err != nil {
		t.Errorf("expected README.md to exist: %v", err)
	}
	if _, err := os.Stat(filepath.Join(got, "main.go")); err != nil {
		t.Errorf("expected main.go to exist: %v", err)
	}
}

func TestRepoDir_Download(t *testing.T) {
	cachedir := t.TempDir()
	t.Setenv(envLibrarianCache, cachedir)

	tarballData := createTestTarball(t, "googleapis-"+testCommit, map[string]string{
		"README.md":                    "# googleapis",
		"google/api/annotations.proto": "syntax = \"proto3\";",
	})

	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/archive/"+testCommit+".tar.gz") {
			http.NotFound(w, r)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write(tarballData)
	}))
	defer server.Close()

	defer func(t http.RoundTripper) { http.DefaultTransport = t }(http.DefaultTransport)
	http.DefaultTransport = server.Client().Transport

	repo := strings.TrimPrefix(server.URL, "https://")
	expectedSHA := fmt.Sprintf("%x", sha256.Sum256(tarballData))
	got, err := RepoDir(repo, testCommit, expectedSHA)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(filepath.Join(got, "README.md")); err != nil {
		t.Errorf("expected README.md to exist: %v", err)
	}
	if _, err := os.Stat(filepath.Join(got, "google/api/annotations.proto")); err != nil {
		t.Errorf("expected google/api/annotations.proto to exist: %v", err)
	}

	tarballPath := tarballPath(cachedir, repo, testCommit)
	if _, err := os.Stat(tarballPath); err != nil {
		t.Errorf("expected tarball to be cached at %q: %v", tarballPath, err)
	}
}
