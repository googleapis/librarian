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

	testExtractedDir = "github.com/googleapis/googleapis@abc123/"
	testInfo         = "download/github.com/googleapis/googleapis@abc123.info"
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
			name:    "uses HOME/.librarian when LIBRARIAN_CACHE not set",
			env:     "",
			wantDir: filepath.Join(os.Getenv("HOME"), ".librarian"),
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

func TestArtifactPath(t *testing.T) {
	const cachedir = "/tmp/cache"

	for _, test := range []struct {
		name   string
		suffix string
		want   string
	}{
		{
			name:   "tarball",
			suffix: "tar.gz",
			want:   filepath.Join(cachedir, testTarball),
		},
		{
			name:   "info file",
			suffix: "info",
			want:   filepath.Join(cachedir, testInfo),
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := artifactPath(cachedir, testRepo, testCommit, test.suffix)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestExtractDir(t *testing.T) {
	cachedir := t.TempDir()
	extractedDir := filepath.Join(cachedir, testExtractedDir)
	if err := os.MkdirAll(extractedDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(extractedDir, "test.txt"), []byte("content"), 0644); err != nil {
		t.Fatal(err)
	}

	got, err := extractDir(cachedir, testRepo, testCommit)
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(extractedDir, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestExtractDir_Empty(t *testing.T) {
	cachedir := t.TempDir()
	if _, err := extractDir(cachedir, testRepo, testCommit); err == nil {
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

	got, err := RepoDir(testRepo, testCommit)
	if err != nil {
		t.Fatal(err)
	}

	// Since extracted dir exists, nothing is downloaded.
	if diff := cmp.Diff(extractedDir, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestRepoDir_TarballExists(t *testing.T) {
	cachedir := t.TempDir()
	t.Setenv(envLibrarianCache, cachedir)

	tarballPath := filepath.Join(cachedir, testTarball)
	infoPath := filepath.Join(cachedir, testInfo)

	if err := os.MkdirAll(filepath.Dir(tarballPath), 0755); err != nil {
		t.Fatal(err)
	}

	tarballData := createTestTarball(t, "test-repo-abc123", map[string]string{
		"README.md": "# Test Repo",
		"main.go":   "package main",
	})
	if err := os.WriteFile(tarballPath, tarballData, 0o644); err != nil {
		t.Fatal(err)
	}

	sha := fmt.Sprintf("%x", sha256.Sum256(tarballData))
	infoData, err := json.MarshalIndent(Info{SHA256: sha}, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(infoPath, infoData, 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := RepoDir(testRepo, testCommit)
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
	got, err := RepoDir(repo, testCommit)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(filepath.Join(got, "README.md")); err != nil {
		t.Errorf("expected README.md to exist: %v", err)
	}
	if _, err := os.Stat(filepath.Join(got, "google/api/annotations.proto")); err != nil {
		t.Errorf("expected google/api/annotations.proto to exist: %v", err)
	}

	tarballPath := artifactPath(cachedir, repo, testCommit, "tar.gz")
	if _, err := os.Stat(tarballPath); err != nil {
		t.Errorf("expected tarball to be cached at %q: %v", tarballPath, err)
	}

	infoPath := artifactPath(cachedir, repo, testCommit, "info")
	infoData, err := os.ReadFile(infoPath)
	if err != nil {
		t.Fatalf("expected info file at %q: %v", infoPath, err)
	}

	var i Info
	if err := json.Unmarshal(infoData, &i); err != nil {
		t.Fatal(err)
	}
	wantSHA := fmt.Sprintf("%x", sha256.Sum256(tarballData))
	if diff := cmp.Diff(wantSHA, i.SHA256); diff != "" {
		t.Errorf("SHA256 mismatch (-want +got):\n%s", diff)
	}
}

