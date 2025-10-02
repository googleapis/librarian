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

package config

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"testing"

	"github.com/google/go-cmp/cmp"
	toml "github.com/pelletier/go-toml/v2"
)

const (
	testGitHubDn       = "https://localhost:12345"
	testGitHubApi      = "https://localhost:23456"
	tarballPathTrailer = "/archive/5d5b1bf126485b0e2c972bac41b376438601e266.tar.gz"
)

func TestUpdateRootConfig(t *testing.T) {
	// update() normally writes `.sidekick.toml` to cwd. We need to change to a
	// temporary directory to avoid changing the actual configuration, and any
	// conflicts with other tests running at the same time.
	tempDir := t.TempDir()
	t.Chdir(tempDir)

	const (
		getLatestShaPath      = "/repos/googleapis/googleapis/commits/master"
		latestSha             = "5d5b1bf126485b0e2c972bac41b376438601e266"
		tarballPath           = "/googleapis/googleapis/archive/5d5b1bf126485b0e2c972bac41b376438601e266.tar.gz"
		latestShaContents     = "The quick brown fox jumps over the lazy dog"
		latestShaContentsHash = "d7a8fbb307d7809469ca9abcb0082e4f8d5651e46d3cdb762d02d0bf37c9e592"
	)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case getLatestShaPath:
			got := r.Header.Get("Accept")
			want := "application/vnd.github.VERSION.sha"
			if got != want {
				t.Fatalf("mismatched Accept header for %q, got=%q, want=%s", r.URL.Path, got, want)
			}
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(latestSha))
		case tarballPath:
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(latestShaContents))
		default:
			t.Fatalf("unexpected request path %q", r.URL.Path)
		}
	}))
	defer server.Close()

	rootConfig := &Config{
		General: GeneralConfig{
			Language:            "rust",
			SpecificationFormat: "protobuf",
		},
		Source: map[string]string{
			"github-api":  server.URL,
			"github":      server.URL,
			"test-root":   fmt.Sprintf("%s/googleapis/googleapis/archive/old.tar.gz", server.URL),
			"test-sha256": "old-sha-unused",
		},
		Codec: map[string]string{},
	}
	if err := WriteSidekickToml(".", rootConfig); err != nil {
		t.Fatal(err)
	}

	if err := UpdateRootConfig(rootConfig, "test"); err != nil {
		t.Fatal(err)
	}

	got := &Config{}
	contents, err := os.ReadFile(path.Join(tempDir, configName))
	if err != nil {
		t.Fatal(err)
	}
	if err := toml.Unmarshal(contents, got); err != nil {
		t.Fatal("error reading top-level configuration: %w", err)
	}
	want := &Config{
		General: rootConfig.General,
		Source: map[string]string{
			"github-api":  server.URL,
			"github":      server.URL,
			"test-root":   fmt.Sprintf("%s/googleapis/googleapis/archive/%s.tar.gz", server.URL, latestSha),
			"test-sha256": latestShaContentsHash,
		},
	}

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch in loaded root config (-want, +got)\n:%s", diff)
	}
}

func TestUpdateRootConfigErrors(t *testing.T) {
	const (
		getLatestShaPath      = "/repos/googleapis/googleapis/commits/master"
		latestSha             = "5d5b1bf126485b0e2c972bac41b376438601e266"
		tarballPath           = "/googleapis/googleapis/archive/5d5b1bf126485b0e2c972bac41b376438601e266.tar.gz"
		latestShaContents     = "The quick brown fox jumps over the lazy dog"
		latestShaContentsHash = "d7a8fbb307d7809469ca9abcb0082e4f8d5651e46d3cdb762d02d0bf37c9e592"
	)

	badLatestSha := func() *httptest.Server {
		return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case getLatestShaPath:
				got := r.Header.Get("Accept")
				want := "application/vnd.github.VERSION.sha"
				if got != want {
					t.Fatalf("mismatched Accept header for %q, got=%q, want=%s", r.URL.Path, got, want)
				}
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte("ERROR - bad request"))
			default:
				t.Fatalf("unexpected request path %q", r.URL.Path)
			}
		}))
	}
	badGetSha256 := func() *httptest.Server {
		return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case getLatestShaPath:
				got := r.Header.Get("Accept")
				want := "application/vnd.github.VERSION.sha"
				if got != want {
					t.Fatalf("mismatched Accept header for %q, got=%q, want=%s", r.URL.Path, got, want)
				}
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(latestSha))
			case tarballPath:
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte("ERROR - bad request"))
			default:
				t.Fatalf("unexpected request path %q", r.URL.Path)
			}
		}))
	}
	goodResponses := func() *httptest.Server {
		return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case getLatestShaPath:
				got := r.Header.Get("Accept")
				want := "application/vnd.github.VERSION.sha"
				if got != want {
					t.Fatalf("mismatched Accept header for %q, got=%q, want=%s", r.URL.Path, got, want)
				}
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(latestSha))
			case tarballPath:
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(latestShaContents))
			default:
				t.Fatalf("unexpected request path %q", r.URL.Path)
			}
		}))
	}

	for _, test := range []struct {
		Server       func() *httptest.Server
		ExtraOptions func(*httptest.Server) map[string]string
		Setup        func(*Config)
	}{
		{
			Server: badLatestSha,
			ExtraOptions: func(*httptest.Server) map[string]string {
				return map[string]string{
					"error-reason":    "githubRepoFromTarballLink() fails.",
					"googleapis-root": "--invalid--",
				}
			},
			Setup: func(*Config) {},
		},
		{
			Server: badLatestSha,
			ExtraOptions: func(server *httptest.Server) map[string]string {
				return map[string]string{
					"error-reason":    "getLatestSha() fails.",
					"googleapis-root": fmt.Sprintf("%s/googleapis/googleapis/archive/tarball.tar.gz", server.URL),
				}
			},
			Setup: func(*Config) {},
		},
		{
			Server: badGetSha256,
			ExtraOptions: func(server *httptest.Server) map[string]string {
				return map[string]string{
					"error-reason":    "getSha256() fails.",
					"googleapis-root": fmt.Sprintf("%s/googleapis/googleapis/archive/tarball.tar.gz", server.URL),
				}
			},
			Setup: func(*Config) {},
		},
		{
			Server: goodResponses,
			ExtraOptions: func(server *httptest.Server) map[string]string {
				return map[string]string{
					"error-reason":    "ReadFile() fails.",
					"googleapis-root": fmt.Sprintf("%s/googleapis/googleapis/archive/tarball.tar.gz", server.URL),
				}
			},
			Setup: func(*Config) {},
		},
		{
			Server: goodResponses,
			ExtraOptions: func(server *httptest.Server) map[string]string {
				return map[string]string{
					"error-reason":    "updateRootConfigContents() fails.",
					"googleapis-root": fmt.Sprintf("%s/googleapis/googleapis/archive/tarball.tar.gz", server.URL),
				}
			},
			Setup: func(config *Config) {
				t.Helper()
				if err := WriteSidekickToml(".", config); err != nil {
					t.Fatal(err)
				}
			},
		},
	} {
		// update() normally writes `.sidekick.toml` to cwd. We need to change to a
		// temporary directory to avoid changing the actual configuration, and any
		// conflicts with other tests running at the same time.
		tempDir := t.TempDir()
		t.Chdir(tempDir)
		server := test.Server()
		defer server.Close()

		rootConfig := &Config{
			General: GeneralConfig{
				Language:            "rust",
				SpecificationFormat: "protobuf",
			},
			Source: map[string]string{
				"github-api": server.URL,
				"github":     server.URL,
			},
			Codec: map[string]string{},
		}
		for k, v := range test.ExtraOptions(server) {
			rootConfig.Source[k] = v
		}
		test.Setup(rootConfig)
		if err := UpdateRootConfig(rootConfig, defaultRoot); err == nil {
			t.Errorf("expected an error with configuration %v", rootConfig)
			t.Fatal(err)
		}
	}
}

func TestGithubConfig(t *testing.T) {
	got := githubConfig(&Config{})
	want := &githubEndpoints{
		Api:      defaultGitHubApi,
		Download: defaultGitHubDn,
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want, +got):\n%s", diff)
	}

	got = githubConfig(&Config{
		Source: map[string]string{
			"github-api": testGitHubApi,
			"github":     testGitHubDn,
		},
	})
	want = &githubEndpoints{
		Api:      testGitHubApi,
		Download: testGitHubDn,
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want, +got):\n%s", diff)
	}
}

func TestGithubRepoFromTarballLink(t *testing.T) {
	config := &Config{
		Source: map[string]string{
			"github-api": testGitHubApi,
			"github":     testGitHubDn,
			"test-root":  testGitHubDn + "/org-name/repo-name" + tarballPathTrailer,
		},
	}
	got, err := githubRepoFromTarballLink(config, "test")
	if err != nil {
		t.Fatal(err)
	}
	want := &githubRepo{
		Org:  "org-name",
		Repo: "repo-name",
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want, +got):\n%s", diff)
	}
}

func TestGithubRepoFromTarballLinkErrors(t *testing.T) {
	for _, test := range []struct {
		ExtraConfig map[string]string
	}{
		{ExtraConfig: map[string]string{}},
		{ExtraConfig: map[string]string{"test-root": "too-short"}},
	} {
		config := &Config{
			Source: map[string]string{
				"github-api": testGitHubApi,
				"github":     testGitHubDn,
			},
		}
		for k, v := range test.ExtraConfig {
			config.Source[k] = v
		}
		if got, err := githubRepoFromTarballLink(config, "test"); err == nil {
			t.Errorf("expected an error, got=%v", got)
		}
	}
}

func TestUpdateRootConfigContents(t *testing.T) {
	const (
		inputContents = `# Test input for updateRootConfigContents
test-extracted-name    = 'repo-name-old-commit'
test-root              = 'https://github.com/org-name/repo-name/archive/old-commit.tar.gz'
test-sha256            = 'old-sha256'
untouched-extracted-name     = 'untouched-123'
untouched-root               = 'https://github.com/org-name/repo-name/archive/refs/tags/123.gz'
untouched-sha256             = 'untouched-sha256'
`
		output1 = `# Test input for updateRootConfigContents
test-extracted-name    = 'repo-name-new-commit-1'
test-root              = 'https://github.com/org-name/repo-name/archive/new-commit-1.tar.gz'
test-sha256            = 'new-sha256-1'
untouched-extracted-name     = 'untouched-123'
untouched-root               = 'https://github.com/org-name/repo-name/archive/refs/tags/123.gz'
untouched-sha256             = 'untouched-sha256'
`
		output2 = `# Test input for updateRootConfigContents
test-extracted-name    = 'repo-name-new-commit-2'
test-root              = 'https://github.com/org-name/repo-name/archive/new-commit-2.tar.gz'
test-sha256            = 'new-sha256-2'
untouched-extracted-name     = 'untouched-123'
untouched-root               = 'https://github.com/org-name/repo-name/archive/refs/tags/123.gz'
untouched-sha256             = 'untouched-sha256'
`
		inputContentsNoExtractedName = `# Test input for updateRootConfigContents
test-root              = 'https://github.com/org-name/repo-name/archive/old-commit.tar.gz'
test-sha256            = 'old-sha256'
untouched-extracted-name     = 'untouched-123'
untouched-root               = 'https://github.com/org-name/repo-name/archive/refs/tags/123.gz'
untouched-sha256             = 'untouched-sha256'
`
		output3 = `# Test input for updateRootConfigContents
test-root              = 'https://github.com/org-name/repo-name/archive/new-commit-3.tar.gz'
test-sha256            = 'new-sha256-3'
untouched-extracted-name     = 'untouched-123'
untouched-root               = 'https://github.com/org-name/repo-name/archive/refs/tags/123.gz'
untouched-sha256             = 'untouched-sha256'
`
		inputContentsNoNewline = `# Test input for updateRootConfigContents
test-root              = 'https://github.com/org-name/repo-name/archive/old-commit.tar.gz'
test-sha256            = 'old-sha256'
# No newline at EOF`
		output4 = `# Test input for updateRootConfigContents
test-root              = 'https://github.com/org-name/repo-name/archive/new-commit-4.tar.gz'
test-sha256            = 'new-sha256-4'
# No newline at EOF`
	)
	for _, test := range []struct {
		RootName  string
		Input     string
		LatestSha string
		NewSha256 string
		Want      string
	}{
		{
			RootName:  "test",
			Input:     inputContents,
			LatestSha: "new-commit-1",
			NewSha256: "new-sha256-1",
			Want:      output1,
		},
		{
			RootName:  "test",
			Input:     inputContents,
			LatestSha: "new-commit-2",
			NewSha256: "new-sha256-2",
			Want:      output2,
		},
		{
			RootName:  "googleapis",
			Input:     inputContents,
			LatestSha: "new-commit-2",
			NewSha256: "new-sha256-2",
			Want:      inputContents,
		},
		{
			RootName:  "test",
			Input:     inputContentsNoExtractedName,
			LatestSha: "new-commit-3",
			NewSha256: "new-sha256-3",
			Want:      output3,
		},
		{
			RootName:  "test",
			Input:     inputContentsNoNewline,
			LatestSha: "new-commit-4",
			NewSha256: "new-sha256-4",
			Want:      output4,
		},
	} {
		endpoints := &githubEndpoints{
			Api:      defaultGitHubApi,
			Download: defaultGitHubDn,
		}
		repo := &githubRepo{
			Org:  "org-name",
			Repo: "repo-name",
		}
		got, err := updateRootConfigContents(test.RootName, []byte(test.Input), endpoints, repo, test.LatestSha, test.NewSha256)
		if err != nil {
			t.Error(err)
			continue
		}
		if diff := cmp.Diff(test.Want, string(got)); diff != "" {
			t.Errorf("mismatch (-want, +got):\n%s", diff)
		}
	}
}

func TestUpdateRootConfigContentsErrors(t *testing.T) {
	const (
		badRoot = `# Test input for updateRootConfigContents
test-extracted-name    = 'repo-name-old-commit'
test-root # Missing separator
test-sha256            = 'old-sha256'
`
		badSha256 = `# Test input for updateRootConfigContents
test-extracted-name    = 'repo-name-old-commit'
test-root              = 'https://github.com/org-name/repo-name/archive/old-commit.tar.gz'
test-sha256 # Missing separator
`

		badExtractedName = `# Test input for updateRootConfigContents
test-extracted-name # Missing separator
test-root              = 'https://github.com/org-name/repo-name/archive/old-commit.tar.gz'
test-sha256            = 'old-sha256'
`

		tooManyRoots = `# Test input for updateRootConfigContents
test-extracted-name    = 'repo-name-old-commit'
test-root              = 'https://github.com/org-name/repo-name/archive/old-commit.tar.gz'
test-root              = 'https://github.com/org-name/repo-name/archive/old-commit.tar.gz'
test-sha256            = 'old-sha256'
`
		tooManySha256 = `# Test input for updateRootConfigContents
test-extracted-name    = 'repo-name-old-commit'
test-root              = 'https://github.com/org-name/repo-name/archive/old-commit.tar.gz'
test-sha256            = 'old-sha256'
test-sha256            = 'old-sha256'
`
		tooManyExtractedNames = `# Test input for updateRootConfigContents
test-extracted-name    = 'repo-name-old-commit'
test-extracted-name    = 'repo-name-old-commit'
test-root              = 'https://github.com/org-name/repo-name/archive/old-commit.tar.gz'
test-sha256            = 'old-sha256'
`
	)
	for idx, test := range []string{badRoot, badSha256, badExtractedName, tooManyRoots, tooManySha256, tooManyExtractedNames} {
		endpoints := &githubEndpoints{
			Api:      defaultGitHubApi,
			Download: defaultGitHubDn,
		}
		repo := &githubRepo{
			Org:  "org-name",
			Repo: "repo-name",
		}
		if got, err := updateRootConfigContents("test", []byte(test), endpoints, repo, "unused", "unused"); err == nil {
			t.Errorf("expected an error in updateRootConfigContents[%d], got=%q", idx, got)
		}
	}
}
