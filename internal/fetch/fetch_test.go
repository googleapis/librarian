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
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"testing"

	"github.com/google/go-cmp/cmp"
)

const (
	testGitHubDn       = "https://localhost:12345"
	tarballPathTrailer = "/archive/5d5b1bf126485b0e2c972bac41b376438601e266.tar.gz"
)

func TestRepoFromTarballLink(t *testing.T) {
	got, err := RepoFromTarballLink(testGitHubDn, testGitHubDn+"/org-name/repo-name"+tarballPathTrailer)
	if err != nil {
		t.Fatal(err)
	}
	want := &Repo{
		Org:  "org-name",
		Repo: "repo-name",
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestRepoFromTarballLinkErrors(t *testing.T) {
	for _, test := range []struct {
		tarballLink string
	}{
		{tarballLink: "too-short"},
	} {
		if got, err := RepoFromTarballLink(testGitHubDn, test.tarballLink); err == nil {
			t.Errorf("expected an error, got=%v", got)
		}
	}
}

func TestSha256(t *testing.T) {
	const (
		tarballPath           = "/googleapis/googleapis/archive/5d5b1bf126485b0e2c972bac41b376438601e266.tar.gz"
		latestShaContents     = "The quick brown fox jumps over the lazy dog"
		latestShaContentsHash = "d7a8fbb307d7809469ca9abcb0082e4f8d5651e46d3cdb762d02d0bf37c9e592"
	)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != tarballPath {
			t.Fatalf("unexpected request path %q", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(latestShaContents))
	}))
	defer server.Close()

	got, err := Sha256(server.URL + tarballPath)
	if err != nil {
		t.Fatal(err)
	}
	if got != latestShaContentsHash {
		t.Errorf("Sha256() = %q, want %q", got, latestShaContentsHash)
	}
}

func TestSha256Error(t *testing.T) {
	for _, test := range []struct {
		name string
		url  string
	}{
		{
			name: "http status error",
			url: func() string {
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusBadRequest)
					w.Write([]byte("ERROR - bad request"))
				}))
				t.Cleanup(server.Close)
				return server.URL + "/test"
			}(),
		},
		{
			name: "invalid url",
			url:  "http://invalid-url-that-does-not-exist-12345.local",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			if _, err := Sha256(test.url); err == nil {
				t.Error("expected an error from Sha256()")
			}
		})
	}
}

func TestLatestSha(t *testing.T) {
	const (
		getLatestShaPath = "/repos/googleapis/googleapis/commits/master"
		latestSha        = "5d5b1bf126485b0e2c972bac41b376438601e266"
	)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != getLatestShaPath {
			t.Fatalf("unexpected request path %q", r.URL.Path)
		}
		got := r.Header.Get("Accept")
		want := "application/vnd.github.VERSION.sha"
		if got != want {
			t.Fatalf("mismatched Accept header for %q, got=%q, want=%s", r.URL.Path, got, want)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(latestSha))
	}))
	defer server.Close()

	got, err := LatestSha(server.URL + getLatestShaPath)
	if err != nil {
		t.Fatal(err)
	}
	if got != latestSha {
		t.Errorf("LatestSha() = %q, want %q", got, latestSha)
	}
}

func TestLatestShaError(t *testing.T) {
	for _, test := range []struct {
		name string
		url  string
	}{
		{
			name: "http status error",
			url: func() string {
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusBadRequest)
					w.Write([]byte("ERROR - bad request"))
				}))
				t.Cleanup(server.Close)
				return server.URL + "/test"
			}(),
		},
		{
			name: "invalid url",
			url:  "http://invalid-url-that-does-not-exist-12345.local",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			if _, err := LatestSha(test.url); err == nil {
				t.Error("expected an error from LatestSha()")
			}
		})
	}
}

func TestTarballLink(t *testing.T) {
	for _, test := range []struct {
		githubDownload string
		repo           *Repo
		sha            string
		want           string
	}{
		{
			githubDownload: "https://github.com",
			repo:           &Repo{Org: "googleapis", Repo: "googleapis"},
			sha:            "abc123",
			want:           "https://github.com/googleapis/googleapis/archive/abc123.tar.gz",
		},
		{
			githubDownload: "https://test.example.com",
			repo:           &Repo{Org: "my-org", Repo: "my-repo"},
			sha:            "def456",
			want:           "https://test.example.com/my-org/my-repo/archive/def456.tar.gz",
		},
	} {
		got := TarballLink(test.githubDownload, test.repo, test.sha)
		if got != test.want {
			t.Errorf("TarballLink() = %q, want %q", got, test.want)
		}
	}
}

func TestDownloadTarballTgzExists(t *testing.T) {
	testDir := t.TempDir()
	tarball := makeTestContents(t)
	target := path.Join(testDir, "existing-file")
	if err := os.WriteFile(target, tarball.Contents, 0644); err != nil {
		t.Fatal(err)
	}
	if err := DownloadTarball(target, "https://unused/placeholder.tar.gz", tarball.Sha256); err != nil {
		t.Fatal(err)
	}
}

func TestDownloadTarballNeedsDownload(t *testing.T) {
	testDir := t.TempDir()
	tarball := makeTestContents(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/placeholder.tar.gz" {
			t.Errorf("Expected to request '/placeholder.tar.gz', got: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		w.Write(tarball.Contents)
	}))
	defer server.Close()

	expected := path.Join(testDir, "new-file")
	if err := DownloadTarball(expected, server.URL+"/placeholder.tar.gz", tarball.Sha256); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(expected)
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(tarball.Contents, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestDownloadTarballChecksumMismatch(t *testing.T) {
	testDir := t.TempDir()
	tarball := makeTestContents(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write(tarball.Contents)
	}))
	defer server.Close()

	target := path.Join(testDir, "target-file")
	wrongSha := "0000000000000000000000000000000000000000000000000000000000000000"

	err := DownloadTarball(target, server.URL+"/test.tar.gz", wrongSha)
	if !errors.Is(err, errChecksumMismatch) {
		t.Fatalf("expected errChecksumMismatch, got: %v", err)
	}
	if _, err := os.Stat(target); !os.IsNotExist(err) {
		t.Errorf("target file should not exist after checksum failure: %v", err)
	}
}

type contents struct {
	Sha256   string
	Contents []byte
}

func makeTestContents(t *testing.T) *contents {
	t.Helper()

	hasher := sha256.New()
	var data []byte
	for i := range 10 {
		line := []byte(fmt.Sprintf("%08d the quick brown fox jumps over the lazy dog\n", i))
		data = append(data, line...)
		hasher.Write(line)
	}

	return &contents{
		Sha256:   fmt.Sprintf("%x", hasher.Sum(nil)),
		Contents: data,
	}
}

func TestExtractTarball(t *testing.T) {
	tarballData := createTestTarball(t, "repo-abc123", map[string]string{
		"README.md":     "# Test Repo",
		"src/main.go":   "package main",
		"docs/guide.md": "# Guide",
	})

	tarballPath := path.Join(t.TempDir(), "test.tar.gz")
	if err := os.WriteFile(tarballPath, tarballData, 0644); err != nil {
		t.Fatal(err)
	}

	destDir := t.TempDir()
	if err := ExtractTarball(tarballPath, destDir); err != nil {
		t.Fatal(err)
	}

	for _, test := range []struct {
		name string
		path string
		want string
	}{
		{"README", "README.md", "# Test Repo"},
		{"main.go", "src/main.go", "package main"},
		{"guide", "docs/guide.md", "# Guide"},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := os.ReadFile(path.Join(destDir, test.path))
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(test.want, string(got)); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}

	// Verify that the top-level directory itself was not created.
	if _, err := os.Stat(path.Join(destDir, "repo-abc123")); err == nil {
		t.Error("top-level directory should not be created")
	}
}

func createTestTarball(t *testing.T, topLevelDir string, files map[string]string) []byte {
	t.Helper()

	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)

	for filePath, content := range files {
		fullPath := topLevelDir + "/" + filePath
		hdr := &tar.Header{
			Name: fullPath,
			Mode: 0644,
			Size: int64(len(content)),
		}
		if err := tw.WriteHeader(hdr); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write([]byte(content)); err != nil {
			t.Fatal(err)
		}
	}

	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}
