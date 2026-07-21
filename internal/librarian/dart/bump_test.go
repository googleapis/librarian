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

package dart

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/googleapis/librarian/internal/command"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/git"
	"github.com/googleapis/librarian/internal/testhelper"
)

func TestUpdateChangelog_New(t *testing.T) {
	tempDir := t.TempDir()

	err := updateChangelog(context.Background(), tempDir, "1.2.3", "", true)
	if err != nil {
		t.Fatalf("updateChangelog failed: %v", err)
	}

	changelogPath := filepath.Join(tempDir, "CHANGELOG.md")
	content, err := os.ReadFile(changelogPath)
	if err != nil {
		t.Fatal(err)
	}

	got := string(content)
	want := `# Changelog

## 1.2.3

- chore: update cloud dependencies

`
	if got != want {
		t.Errorf("CHANGELOG.md content mismatch:\ngot:\n%s\nwant:\n%s", got, want)
	}
}

func TestUpdateChangelog_Existing(t *testing.T) {
	tempDir := t.TempDir()
	changelogPath := filepath.Join(tempDir, "CHANGELOG.md")
	initialContent := `# Changelog

## 1.2.2

- chore: release 1.2.2
`
	if err := os.WriteFile(changelogPath, []byte(initialContent), 0644); err != nil {
		t.Fatal(err)
	}

	err := updateChangelog(context.Background(), tempDir, "1.2.3", "", true)
	if err != nil {
		t.Fatalf("updateChangelog failed: %v", err)
	}

	content, err := os.ReadFile(changelogPath)
	if err != nil {
		t.Fatal(err)
	}

	got := string(content)
	want := `# Changelog

## 1.2.3

- chore: update cloud dependencies

## 1.2.2

- chore: release 1.2.2
`
	if got != want {
		t.Errorf("CHANGELOG.md content mismatch:\ngot:\n%s\nwant:\n%s", got, want)
	}
}

func TestUpdateChangelog_WithCommits(t *testing.T) {
	tempDir := t.TempDir()

	testhelper.ContinueInNewGitRepository(t, tempDir)
	t.Chdir(tempDir)

	if err := os.WriteFile("file.txt", []byte("init"), 0644); err != nil {
		t.Fatal(err)
	}
	testhelper.RunGit(t, "add", ".")
	testhelper.RunGit(t, "commit", "-m", "Initial release.")

	tagCommit, err := git.GetCommitHash(context.Background(), command.Git, "HEAD")
	if err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile("file.txt", []byte("feat 1"), 0644); err != nil {
		t.Fatal(err)
	}
	testhelper.RunGit(t, "add", ".")
	testhelper.RunGit(t, "commit", "-m", "feat: added support for something")

	if err := os.WriteFile("file.txt", []byte("fix 1"), 0644); err != nil {
		t.Fatal(err)
	}
	testhelper.RunGit(t, "add", ".")
	testhelper.RunGit(t, "commit", "-m", "fix: resolved a bug")

	err = updateChangelog(context.Background(), tempDir, "1.2.3", tagCommit, false)
	if err != nil {
		t.Fatalf("updateChangelog failed: %v", err)
	}

	changelogPath := filepath.Join(tempDir, "CHANGELOG.md")
	content, err := os.ReadFile(changelogPath)
	if err != nil {
		t.Fatal(err)
	}

	got := string(content)
	want := `# Changelog

## 1.2.3

- fix: resolved a bug
- feat: added support for something

`
	if got != want {
		t.Errorf("CHANGELOG.md content mismatch:\ngot:\n%s\nwant:\n%s", got, want)
	}
}

type PackageVersion struct {
	needed string
	old    string
}

func setupFakeApitool(t *testing.T, responses map[string]PackageVersion) {
	t.Helper()

	var script strings.Builder
	script.WriteString(`#!/bin/bash
report_file=""
pkg_name=""
while [ $# -gt 0 ]; do
  if [ "$1" == "--report-file-path" ]; then
    report_file="$2"
    shift
  elif [[ "$1" == pub://* ]]; then
    pkg_name="${1#pub://}"
  fi
  shift
done

if [ -n "$report_file" ]; then
`)

	first := true
	for pkg, res := range responses {
		if first {
			fmt.Fprintf(&script, "  if [ \"$pkg_name\" == %q ]; then\n", pkg)
			first = false
		} else {
			fmt.Fprintf(&script, "  elif [ \"$pkg_name\" == %q ]; then\n", pkg)
		}
		fmt.Fprintf(&script, "    echo '{\"version\": {\"needed\": %q, \"old\": %q}}' > \"$report_file\"\n", res.needed, res.old)
	}
	if !first {
		script.WriteString("  fi\n")
	}
	script.WriteString("fi\n")

	setupFakeScript(t, "dart-apitool", script.String())
}

func TestBump_Leaf(t *testing.T) {
	testhelper.RequireCommand(t, "git")

	t.Helper()
	remoteDir := testhelper.SetupRepoWithChange(t, "release-2001-02-03")
	if err := command.Run(t.Context(), command.Git, "-C", remoteDir, "config", "receive.denyCurrentBranch", "ignore"); err != nil {
		t.Fatal(err)
	}
	testhelper.CloneRepository(t, remoteDir)

	if err := os.MkdirAll("generated/a", 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll("generated/b", 0755); err != nil {
		t.Fatal(err)
	}

	workspacePubspec := `name: pkg_workspace
publish_to: none

environment:
  sdk: ^3.9.0

workspace:
  - generated/a
  - generated/b
`
	packageAPubspec := `name: a
version: 1.0.0
environment:
  sdk: ^3.9.0
resolution: workspace
`
	packageBPubspec := `name: b
version: 1.0.0
environment:
  sdk: ^3.9.0
resolution: workspace
dependencies:
  a: ^1.0.0
`

	if err := os.WriteFile("pubspec.yaml", []byte(workspacePubspec), 0644); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile("generated/a/pubspec.yaml", []byte(packageAPubspec), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile("generated/b/pubspec.yaml", []byte(packageBPubspec), 0644); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile("generated/a/lib.dart", []byte("// library a"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile("generated/b/lib.dart", []byte("// library b"), 0644); err != nil {
		t.Fatal(err)
	}

	testhelper.RunGit(t, "add", ".")
	testhelper.RunGit(t, "commit", "-m", "feat: added pubspec files", ".")
	testhelper.RunGit(t, "push", config.RemoteUpstream, config.BranchMain)

	// Tag the initial 1.0.0 release.
	testhelper.RunGit(t, "tag", "a-v1.0.0")
	testhelper.RunGit(t, "tag", "b-v1.0.0")

	// Now make a commit with changes to package a.
	if err := os.WriteFile("generated/a/lib.dart", []byte("// library a: new feature"), 0644); err != nil {
		t.Fatal(err)
	}
	testhelper.RunGit(t, "add", ".")
	testhelper.RunGit(t, "commit", "-m", "feat: added support for something new in a", ".")

	apiToolResponses := map[string]PackageVersion{
		"a": {needed: "1.1.0", old: "1.0.0"},
		"b": {needed: "1.0.0", old: "1.0.0"},
	}
	setupFakeApitool(t, apiToolResponses)

	cfg := &config.Config{
		Default: &config.Default{
			Output:    "generated",
			TagFormat: "{name}-v{version}",
			Dart: &config.DartPackage{
				Packages: map[string]string{
					"package:a": "^1.0.0",
				},
			},
		},
		Libraries: []*config.Library{
			{Name: "a", Version: "1.0.0"},
			{Name: "b", Version: "1.0.0"},
		},
	}

	err := Bump(t.Context(), cfg, true, "", "")
	if err != nil {
		t.Fatalf("Bump failed: %v", err)
	}

	// Verify versions in config:
	// a should be bumped to 1.1.0
	// b should be bumped to 1.0.1 (patch bump because its dependency "a" was updated)
	if got, want := cfg.Libraries[0].Version, "1.1.0"; got != want {
		t.Errorf("library a version = %q; want %q", got, want)
	}
	if got, want := cfg.Libraries[1].Version, "1.0.1"; got != want {
		t.Errorf("library b version = %q; want %q", got, want)
	}

	// Verify cfg.Default.Dart.Packages values:
	if got, want := cfg.Default.Dart.Packages, map[string]string{"package:a": "^1.1.0"}; !reflect.DeepEqual(got, want) {
		t.Errorf("default packages map = %v; want %v", got, want)
	}

	// Verify updated files in directory:
	// a's pubspec should be 1.1.0
	// b's pubspec should be 1.0.1 and depend on a: ^1.1.0
	pubspecA, err := os.ReadFile("generated/a/pubspec.yaml")
	if err != nil {
		t.Fatal(err)
	}
	wantPubspecA := `name: a
version: 1.1.0
environment:
  sdk: ^3.9.0
resolution: workspace
`
	if got := string(pubspecA); got != wantPubspecA {
		t.Errorf("a/pubspec.yaml content mismatch:\ngot:\n%s\nwant:\n%s", got, wantPubspecA)
	}

	pubspecB, err := os.ReadFile("generated/b/pubspec.yaml")
	if err != nil {
		t.Fatal(err)
	}
	wantPubspecB := `name: b
version: 1.0.1
environment:
  sdk: ^3.9.0
resolution: workspace
dependencies:
  a: ^1.1.0
`
	if got := string(pubspecB); got != wantPubspecB {
		t.Errorf("b/pubspec.yaml content mismatch:\ngot:\n%s\nwant:\n%s", got, wantPubspecB)
	}
}
