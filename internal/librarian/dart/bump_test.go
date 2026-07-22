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
	"slices"
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

func TestBump_NothingChanged(t *testing.T) {
	testhelper.RequireCommand(t, "dart")
	testhelper.RequireCommand(t, "git")

	repoVersions := map[string]string{
		"a": "1.0.0",
		"b": "1.0.0",
		"c": "1.0.0",
	}
	deps := map[string][]string{
		"b": {"a"},
	}
	setupRepo(t, repoVersions, deps)

	apiToolResponses := map[string]packageVersion{
		"a": {needed: "1.0.0", old: "1.0.0"},
		"b": {needed: "1.0.0", old: "1.0.0"},
		"c": {needed: "1.0.0", old: "1.0.0"},
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
			{Name: "c", Version: "1.0.0"},
		},
	}

	err := Bump(t.Context(), cfg, true, "", "")
	if err != nil {
		t.Fatalf("Bump failed: %v", err)
	}

	if got, want := libraryVersions(cfg.Libraries), map[string]string{
		"a": "1.0.0",
		"b": "1.0.0",
		"c": "1.0.0",
	}; !reflect.DeepEqual(got, want) {
		t.Errorf("library versions = %v; want %v", got, want)
	}

	// Verify cfg.Default.Dart.Packages values:
	if got, want := cfg.Default.Dart.Packages, map[string]string{"package:a": "^1.0.0"}; !reflect.DeepEqual(got, want) {
		t.Errorf("default packages map = %v; want %v", got, want)
	}

	// Verify updated files in directory:
	// a's pubspec should be 1.0.0
	// b's pubspec should be 1.0.0 and depend on a: ^1.0.0
	pubspecA, err := os.ReadFile("generated/a/pubspec.yaml")
	if err != nil {
		t.Fatal(err)
	}
	wantPubspecA := `name: a
version: 1.0.0
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
version: 1.0.0
environment:
  sdk: ^3.9.0
resolution: workspace
dependencies:
  a: ^1.0.0
`
	if got := string(pubspecB); got != wantPubspecB {
		t.Errorf("b/pubspec.yaml content mismatch:\ngot:\n%s\nwant:\n%s", got, wantPubspecB)
	}

	pubspecC, err := os.ReadFile("generated/c/pubspec.yaml")
	if err != nil {
		t.Fatal(err)
	}
	wantPubspecC := `name: c
version: 1.0.0
environment:
  sdk: ^3.9.0
resolution: workspace
`
	if got := string(pubspecC); got != wantPubspecC {
		t.Errorf("c/pubspec.yaml content mismatch:\ngot:\n%s\nwant:\n%s", got, wantPubspecC)
	}
}

func TestBump_APIChange(t *testing.T) {
	testhelper.RequireCommand(t, "git")

	t.Helper()
	repoVersions := map[string]string{
		"a": "1.0.0",
		"b": "1.0.0",
	}
	deps := map[string][]string{
		"b": {"a"},
	}
	setupRepo(t, repoVersions, deps)

	// Now make a commit with changes to package a.
	if err := os.WriteFile("generated/a/lib.dart", []byte("const a = 5"), 0644); err != nil {
		t.Fatal(err)
	}
	testhelper.RunGit(t, "add", ".")
	testhelper.RunGit(t, "commit", "-m", "feat: added new value", ".")

	// Since the API surfaces didn't change, dart-apitool will report that the versions do not
	// need to be bumped.
	apiToolResponses := map[string]packageVersion{
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
	if got, want := libraryVersions(cfg.Libraries), map[string]string{
		"a": "1.1.0",
		"b": "1.0.1",
	}; !reflect.DeepEqual(got, want) {
		t.Errorf("library versions = %v; want %v", got, want)
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

func libraryVersions(libaries []*config.Library) map[string]string {
	m := make(map[string]string)
	for _, l := range libaries {
		m[l.Name] = l.Version
	}
	return m
}

func TestBump_FileChanged_APIUnchanged(t *testing.T) {
	testhelper.RequireCommand(t, "git")
	testhelper.RequireCommand(t, "dart")

	t.Helper()

	repoVersions := map[string]string{
		"a": "1.0.0",
		"b": "1.0.0",
		"c": "1.0.0",
	}
	deps := map[string][]string{
		"b": {"a"},
	}
	setupRepo(t, repoVersions, deps)

	// Now make a commit with changes to package a.
	if err := os.WriteFile("generated/a/lib.dart", []byte("// library a: new fix"), 0644); err != nil {
		t.Fatal(err)
	}
	testhelper.RunGit(t, "add", ".")
	testhelper.RunGit(t, "commit", "-m", "fix: generator bug", ".")

	// Since the API surfaces didn't change, dart-apitool will report that the versions do not
	// need to be bumped.
	apiToolResponses := map[string]packageVersion{
		"a": {needed: "1.0.0", old: "1.0.0"},
		"b": {needed: "1.0.0", old: "1.0.0"},
		"c": {needed: "1.0.0", old: "1.0.0"},
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
			{Name: "c", Version: "1.0.0"},
		},
	}

	err := Bump(t.Context(), cfg, true, "", "")
	if err != nil {
		t.Fatalf("Bump failed: %v", err)
	}

	// Verify versions in config:
	// a should be bumped to 1.0.1
	// b should be bumped to 1.0.1 (patch bump because its dependency "a" was updated)

	if got, want := libraryVersions(cfg.Libraries), map[string]string{
		"a": "1.0.1",
		"b": "1.0.1",
		"c": "1.0.0",
	}; !reflect.DeepEqual(got, want) {
		t.Errorf("library versions = %v; want %v", got, want)
	}

	// Verify cfg.Default.Dart.Packages values:
	if got, want := cfg.Default.Dart.Packages, map[string]string{"package:a": "^1.0.1"}; !reflect.DeepEqual(got, want) {
		t.Errorf("default packages map = %v; want %v", got, want)
	}

	// Verify updated files in directory:
	// a's pubspec should be 1.0.1
	// b's pubspec should be 1.0.1 and depend on a: ^1.0.1
	pubspecA, err := os.ReadFile("generated/a/pubspec.yaml")
	if err != nil {
		t.Fatal(err)
	}
	wantPubspecA := `name: a
version: 1.0.1
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
  a: ^1.0.1
`
	if got := string(pubspecB); got != wantPubspecB {
		t.Errorf("b/pubspec.yaml content mismatch:\ngot:\n%s\nwant:\n%s", got, wantPubspecB)
	}
}

// repoVersions: {"a": "1.0.0", "b": "1.0.0", "c": "1.0.0"}
// deps: {"a": ["b", "c"]}
func setupRepo(t *testing.T, repoVersions map[string]string, deps map[string][]string) {
	t.Helper()

	remoteDir := testhelper.SetupRepoWithChange(t, "release-2001-02-03")
	if err := command.Run(t.Context(), command.Git, "-C", remoteDir, "config", "receive.denyCurrentBranch", "ignore"); err != nil {
		t.Fatal(err)
	}
	testhelper.CloneRepository(t, remoteDir)

	var workspaceLines []string
	workspaceLines = append(workspaceLines, "name: pkg_workspace", "publish_to: none", "", "environment:", "  sdk: ^3.9.0", "", "workspace:")

	var pkgNames []string
	for name := range repoVersions {
		pkgNames = append(pkgNames, name)
	}
	slices.Sort(pkgNames)
	for _, name := range pkgNames {
		workspaceLines = append(workspaceLines, fmt.Sprintf("  - generated/%s", name))
	}
	workspacePubspec := strings.Join(workspaceLines, "\n") + "\n"

	if err := os.WriteFile("pubspec.yaml", []byte(workspacePubspec), 0644); err != nil {
		t.Fatal(err)
	}

	for _, name := range pkgNames {
		version := repoVersions[name]
		if err := os.MkdirAll("generated/"+name, 0755); err != nil {
			t.Fatal(err)
		}

		var dependencies string
		if len(deps[name]) > 0 {
			dependencies = "\ndependencies:\n"
			for _, depName := range deps[name] {
				dependencies += fmt.Sprintf("  %s: ^%s\n", depName, repoVersions[depName])
			}
		}

		pubspec := fmt.Sprintf(`name: %s
version: %s
environment:
  sdk: ^3.9.0
resolution: workspace%s`, name, version, dependencies)
		pubspec = strings.TrimSuffix(pubspec, "\n") + "\n"
		if err := os.WriteFile("generated/"+name+"/pubspec.yaml", []byte(pubspec), 0644); err != nil {
			t.Fatal(err)
		}

		if err := os.WriteFile("generated/"+name+"/lib.dart", []byte("// library "+name), 0644); err != nil {
			t.Fatal(err)
		}
	}

	testhelper.RunGit(t, "add", ".")
	testhelper.RunGit(t, "commit", "-m", "feat: added pubspec files", ".")
	testhelper.RunGit(t, "push", config.RemoteUpstream, config.BranchMain)

	// Tag the initial releases
	for name, version := range repoVersions {
		testhelper.RunGit(t, "tag", fmt.Sprintf("%s-v%s", name, version))
	}
}

type packageVersion struct {
	needed string
	old    string
}

func setupFakeApitool(t *testing.T, responses map[string]packageVersion) {
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
