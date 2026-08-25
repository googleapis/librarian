// Copyright 2026 Google LLC
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

package swift

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/googleapis/librarian/internal/command"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/git"
	"github.com/googleapis/librarian/internal/testhelper"
)

func TestFormatRemoteURL(t *testing.T) {
	for _, tc := range []struct {
		name    string
		format  string
		repo    string
		libName string
		wantURL string
	}{
		{
			name:    "default googleapis",
			format:  "",
			repo:    "googleapis/google-cloud-swift",
			libName: "google-cloud-auth",
			wantURL: "git@github.com:googleapis/google-cloud-auth.git",
		},
		{
			name:    "custom format",
			format:  "git@github.com:test-org/{name}.git",
			repo:    "googleapis/google-cloud-swift",
			libName: "google-cloud-auth",
			wantURL: "git@github.com:test-org/google-cloud-auth.git",
		},
		{
			name:    "custom https format",
			format:  "https://github.com/test-org/{name}.git",
			repo:    "googleapis/google-cloud-swift",
			libName: "google-rpc",
			wantURL: "https://github.com/test-org/google-rpc.git",
		},
		{
			name:    "empty repo fallback",
			format:  "",
			repo:    "",
			libName: "google-cloud-auth",
			wantURL: "git@github.com:googleapis/google-cloud-auth.git",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := FormatRemoteURL(tc.format, tc.repo, tc.libName)
			if got != tc.wantURL {
				t.Errorf("FormatRemoteURL(%q, %q, %q) = %q, want %q", tc.format, tc.repo, tc.libName, got, tc.wantURL)
			}
		})
	}
}

func setupSplitRemoteRepo(t *testing.T, tag string) string {
	t.Helper()
	remoteDir := t.TempDir()
	testhelper.ContinueInNewGitRepository(t, remoteDir)
	if err := os.WriteFile("README.md", []byte("# Remote Repo"), 0o644); err != nil {
		t.Fatal(err)
	}
	testhelper.RunGit(t, "add", ".")
	testhelper.RunGit(t, "commit", "-m", "initial remote commit")
	if tag != "" {
		testhelper.RunGit(t, "tag", tag)
	}
	return remoteDir
}

func TestPublishSuccess(t *testing.T) {
	testhelper.RequireCommand(t, "git")

	// Create remote split repos for two libraries: auth and storage
	authRemote := setupSplitRemoteRepo(t, "1.0.0")
	storageRemote := setupSplitRemoteRepo(t, "")

	// Set up monorepo
	monorepoRemote := setupMonorepoWithRootFiles(t)
	cloneDir := t.TempDir()
	t.Chdir(cloneDir)
	testhelper.RunGit(t, "clone", monorepoRemote, ".")
	testhelper.RunGit(t, "remote", "rename", "origin", config.RemoteUpstream)
	testhelper.ConfigNewGitRepository(t)
	remotesMap := map[string]string{
		"google-cloud-auth":    authRemote,
		"google-cloud-storage": storageRemote,
	}

	cfg := &config.Config{
		Language: config.LanguageSwift,
		Repo:     "googleapis/google-cloud-swift",
		Libraries: []*config.Library{
			{
				Name:    "google-cloud-auth",
				Version: "1.0.0",
				Output:  "packages/auth",
			},
			{
				Name:    "google-cloud-storage",
				Version: "1.1.0",
				Output:  "packages/storage",
			},
			{
				Name:        "skipped-lib",
				Version:     "1.0.0",
				Output:      "packages/auth",
				SkipRelease: true,
			},
			{
				Name:    "unversioned-lib",
				Version: "",
				Output:  "packages/auth",
			},
		},
	}

	// Publish with dynamic remote format using template
	// We can use a custom remote URL format using local paths:
	// For testing, we point FormatRemoteURL to remotesMap entries
	tempDir := t.TempDir()
	for name, remotePath := range remotesMap {
		linkPath := filepath.Join(tempDir, name+".git")
		testhelper.RunGit(t, "clone", "--bare", remotePath, linkPath)
	}

	err := Publish(t.Context(), PublishParams{
		Config:          cfg,
		RemoteURLFormat: filepath.Join(tempDir, "{name}.git"),
		Origin:          "HEAD",
		RemoteBranch:    config.BranchMain,
	})
	if err != nil {
		t.Fatalf("Publish() failed: %v", err)
	}

	// Verify that storage remote now has tag 1.1.0
	storageBareRepo := filepath.Join(tempDir, "google-cloud-storage.git")
	hasTag, err := git.RemoteTagExists(t.Context(), command.Git, storageBareRepo, "1.1.0")
	if err != nil {
		t.Fatal(err)
	}
	if !hasTag {
		t.Errorf("tag 1.1.0 was not pushed to storage remote")
	}
}

func TestPublishDryRun(t *testing.T) {
	testhelper.RequireCommand(t, "git")

	storageRemote := setupSplitRemoteRepo(t, "")

	monorepoRemote := setupMonorepoWithRootFiles(t)
	cloneDir := t.TempDir()
	t.Chdir(cloneDir)
	testhelper.RunGit(t, "clone", monorepoRemote, ".")
	testhelper.RunGit(t, "remote", "rename", "origin", config.RemoteUpstream)
	testhelper.ConfigNewGitRepository(t)

	tempDir := t.TempDir()
	storageBareRepo := filepath.Join(tempDir, "google-cloud-storage.git")
	testhelper.RunGit(t, "clone", "--bare", storageRemote, storageBareRepo)

	cfg := &config.Config{
		Language: config.LanguageSwift,
		Repo:     "googleapis/google-cloud-swift",
		Libraries: []*config.Library{
			{
				Name:    "google-cloud-storage",
				Version: "1.0.0",
				Output:  "packages/auth",
			},
		},
	}

	err := Publish(t.Context(), PublishParams{
		Config:          cfg,
		RemoteURLFormat: filepath.Join(tempDir, "{name}.git"),
		DryRun:          true,
	})
	if err != nil {
		t.Fatalf("Publish() with DryRun failed: %v", err)
	}

	// Verify tag was NOT pushed
	hasTag, err := git.RemoteTagExists(t.Context(), command.Git, storageBareRepo, "1.0.0")
	if err != nil {
		t.Fatal(err)
	}
	if hasTag {
		t.Errorf("tag 1.0.0 was pushed during dry-run; want not pushed")
	}
}

func TestPublishEmptyOutputDir(t *testing.T) {
	testhelper.RequireCommand(t, "git")

	monorepoRemote := setupMonorepoWithRootFiles(t)
	cloneDir := t.TempDir()
	t.Chdir(cloneDir)
	testhelper.RunGit(t, "clone", monorepoRemote, ".")
	testhelper.RunGit(t, "remote", "rename", "origin", config.RemoteUpstream)
	testhelper.ConfigNewGitRepository(t)

	cfg := &config.Config{
		Language: config.LanguageSwift,
		Repo:     "googleapis/google-cloud-swift",
		Libraries: []*config.Library{
			{
				Name:    "mixed-lib",
				Version: "1.0.0",
				Swift: &config.SwiftPackage{
					Modules: []*config.SwiftModule{{Output: "Tests/generated", APIPath: "google/storage/v2"}},
				},
			},
		},
	}

	err := Publish(t.Context(), PublishParams{
		Config: cfg,
	})
	if err == nil {
		t.Fatal("expected error for library without output directory, got nil")
	}

	// With DryRunKeepGoing, it should log and continue without error
	err = Publish(t.Context(), PublishParams{
		Config:          cfg,
		DryRunKeepGoing: true,
	})
	if err != nil {
		t.Fatalf("Publish() with DryRunKeepGoing failed: %v", err)
	}
}

func TestMatchLibrary(t *testing.T) {
	for _, tc := range []struct {
		name    string
		targets []string
		libName string
		libDir  string
		want    bool
	}{
		{
			name:    "match by name",
			targets: []string{"google-cloud-wkt"},
			libName: "google-cloud-wkt",
			libDir:  "packages/wkt",
			want:    true,
		},
		{
			name:    "match by relative path",
			targets: []string{"packages/wkt"},
			libName: "google-cloud-wkt",
			libDir:  "packages/wkt",
			want:    true,
		},
		{
			name:    "match by path with trailing slash",
			targets: []string{"packages/wkt/"},
			libName: "google-cloud-wkt",
			libDir:  "packages/wkt",
			want:    true,
		},
		{
			name:    "match by path with leading dot-slash",
			targets: []string{"./packages/wkt"},
			libName: "google-cloud-wkt",
			libDir:  "packages/wkt",
			want:    true,
		},
		{
			name:    "match by base directory name",
			targets: []string{"wkt"},
			libName: "google-cloud-wkt",
			libDir:  "packages/wkt",
			want:    true,
		},
		{
			name:    "no match",
			targets: []string{"packages/auth"},
			libName: "google-cloud-wkt",
			libDir:  "packages/wkt",
			want:    false,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := matchLibrary(tc.targets, tc.libName, tc.libDir)
			if got != tc.want {
				t.Errorf("matchLibrary(%v, %q, %q) = %v, want %v", tc.targets, tc.libName, tc.libDir, got, tc.want)
			}
		})
	}
}
