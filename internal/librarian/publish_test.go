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

package librarian

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/testhelper"
	"github.com/googleapis/librarian/internal/yaml"
)

func TestPublishCommand_UnsupportedLanguage(t *testing.T) {
	tempDir := t.TempDir()
	t.Chdir(tempDir)

	cfg := &config.Config{
		Language: "python",
	}
	if err := yaml.Write(config.LibrarianYAML, cfg); err != nil {
		t.Fatal(err)
	}

	err := Run(t.Context(), "librarian", "publish")
	if err == nil {
		t.Fatal("expected error for unsupported language, got nil")
	}
}

func TestPublishCommand_Swift(t *testing.T) {
	testhelper.RequireCommand(t, "git")

	remoteDir := t.TempDir()
	testhelper.ContinueInNewGitRepository(t, remoteDir)
	if err := os.WriteFile("LICENSE", []byte("license"), 0o644); err != nil {
		t.Fatal(err)
	}
	authDir := filepath.Join("packages", "auth")
	testhelper.AddSwiftPackage(t, authDir, "GoogleCloudAuth")
	cfg := &config.Config{
		Language: config.LanguageSwift,
		Repo:     "googleapis/google-cloud-swift",
		Libraries: []*config.Library{
			{
				Name:    "google-cloud-auth",
				Version: "1.0.0",
				Output:  "packages/auth",
			},
		},
	}
	if err := yaml.Write(config.LibrarianYAML, cfg); err != nil {
		t.Fatal(err)
	}
	testhelper.RunGit(t, "add", ".")
	testhelper.RunGit(t, "commit", "-m", "initial commit")

	cloneDir := t.TempDir()
	t.Chdir(cloneDir)
	testhelper.RunGit(t, "clone", remoteDir, ".")
	testhelper.RunGit(t, "remote", "rename", "origin", config.RemoteUpstream)
	testhelper.ConfigNewGitRepository(t)

	splitBareRepo := filepath.Join(t.TempDir(), "swift-auth.git")
	splitRemote := t.TempDir()
	testhelper.ContinueInNewGitRepository(t, splitRemote)
	if err := os.WriteFile("README.md", []byte("# Auth"), 0o644); err != nil {
		t.Fatal(err)
	}
	testhelper.RunGit(t, "add", ".")
	testhelper.RunGit(t, "commit", "-m", "init auth remote")
	testhelper.RunGit(t, "clone", "--bare", splitRemote, splitBareRepo)

	t.Chdir(cloneDir)
	err := Run(t.Context(), "librarian", "publish", "--dry-run", "--remote-url-format", filepath.Dir(splitBareRepo)+"/{name}.git")
	if err != nil {
		t.Fatalf("librarian publish --dry-run failed: %v", err)
	}

	err = Run(t.Context(), "librarian", "publish", "--dry-run", "--remote-url-format", filepath.Dir(splitBareRepo)+"/{name}.git", "google-cloud-auth")
	if err != nil {
		t.Fatalf("librarian publish specific library failed: %v", err)
	}

	err = Run(t.Context(), "librarian", "publish", "--dry-run", "--upstream", config.RemoteUpstream, "--remote-url-format", filepath.Dir(splitBareRepo)+"/{name}.git", "google-cloud-auth")
	if err != nil {
		t.Fatalf("librarian publish with --upstream failed: %v", err)
	}
}
