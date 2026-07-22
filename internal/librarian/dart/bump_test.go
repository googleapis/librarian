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
	"strings"
	"testing"

	"github.com/googleapis/librarian/internal/command"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/git"
	"github.com/googleapis/librarian/internal/testhelper"
	"github.com/googleapis/librarian/internal/yaml"
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

- initial release

`
	if got != want {
		t.Errorf("CHANGELOG.md content mismatch:\ngot:\n%s\nwant:\n%s", got, want)
	}
}

func TestUpdateChangelog_UpdateWithCommits(t *testing.T) {
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

	changelogPath := filepath.Join(tempDir, "CHANGELOG.md")
	initialContent := `# Changelog

## 1.2.2

- chore: release 1.2.2
`
	if err := os.WriteFile(changelogPath, []byte(initialContent), 0644); err != nil {
		t.Fatal(err)
	}

	err = updateChangelog(context.Background(), tempDir, "1.2.3", tagCommit, false)
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

- fix: resolved a bug
- feat: added support for something

## 1.2.2

- chore: release 1.2.2
`
	if got != want {
		t.Errorf("CHANGELOG.md content mismatch:\ngot:\n%s\nwant:\n%s", got, want)
	}
}

func TestUpdateChangelog_UpdateWithDeps(t *testing.T) {
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

	changelogPath := filepath.Join(tempDir, "CHANGELOG.md")
	initialContent := `# Changelog

## 1.2.2

- chore: release 1.2.2
`
	if err := os.WriteFile(changelogPath, []byte(initialContent), 0644); err != nil {
		t.Fatal(err)
	}

	err = updateChangelog(context.Background(), tempDir, "1.2.3", tagCommit, true)
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

func TestBump_NothingChanged(t *testing.T) {
	testhelper.RequireCommand(t, "dart")
	testhelper.RequireCommand(t, "git")

	inputDir, err := filepath.Abs("testdata/bump/input")
	if err != nil {
		t.Fatal(err)
	}
	goldenDir, err := filepath.Abs("testdata/bump/nothing_changed/golden_output")
	if err != nil {
		t.Fatal(err)
	}

	setupRepoFromDir(t, inputDir)
	testhelper.RunGit(t, "tag", "a-v1.0.0")
	testhelper.RunGit(t, "tag", "b-v1.0.0")
	testhelper.RunGit(t, "tag", "c-v1.0.0")

	apiToolResponses := map[string]packageVersion{
		"a": {needed: "1.0.0", old: "1.0.0"},
		"b": {needed: "1.0.0", old: "1.0.0"},
		"c": {needed: "1.0.0", old: "1.0.0"},
	}
	setupFakeApitool(t, apiToolResponses)

	cfg, err := yaml.Read[config.Config]("librarian.yaml")
	if err != nil {
		t.Fatal(err)
	}

	err = Bump(t.Context(), cfg, true, "", "")
	if err != nil {
		t.Fatalf("Bump failed: %v", err)
	}

	if err := yaml.Write("librarian.yaml", cfg); err != nil {
		t.Fatal(err)
	}

	compareDirWithGolden(t, goldenDir)
}

func TestBump_APIChange(t *testing.T) {
	testhelper.RequireCommand(t, "dart")
	testhelper.RequireCommand(t, "git")

	inputDir, err := filepath.Abs("testdata/bump/input")
	if err != nil {
		t.Fatal(err)
	}
	goldenDir, err := filepath.Abs("testdata/bump/api_change/golden_output")
	if err != nil {
		t.Fatal(err)
	}

	setupRepoFromDir(t, inputDir)
	testhelper.RunGit(t, "tag", "a-v1.0.0")
	testhelper.RunGit(t, "tag", "b-v1.0.0")
	testhelper.RunGit(t, "tag", "c-v1.0.0")

	// Now make a commit with changes to package a.
	if err := os.WriteFile("generated/a/lib.dart", []byte("const a = 5;"), 0644); err != nil {
		t.Fatal(err)
	}
	testhelper.RunGit(t, "add", ".")
	testhelper.RunGit(t, "commit", "-m", "feat: added new value", ".")

	// Since the API surfaces didn't change, dart-apitool will report that the versions do not
	// need to be bumped.
	apiToolResponses := map[string]packageVersion{
		"a": {needed: "1.1.0", old: "1.0.0"},
		"b": {needed: "1.0.0", old: "1.0.0"},
		"c": {needed: "1.0.0", old: "1.0.0"},
	}
	setupFakeApitool(t, apiToolResponses)

	cfg, err := yaml.Read[config.Config]("librarian.yaml")
	if err != nil {
		t.Fatal(err)
	}

	err = Bump(t.Context(), cfg, true, "", "")
	if err != nil {
		t.Fatalf("Bump failed: %v", err)
	}

	if err := yaml.Write("librarian.yaml", cfg); err != nil {
		t.Fatal(err)
	}

	compareDirWithGolden(t, goldenDir)
}

func TestBump_FileChanged_APIUnchanged(t *testing.T) {
	testhelper.RequireCommand(t, "dart")
	testhelper.RequireCommand(t, "git")

	inputDir, err := filepath.Abs("testdata/bump/input")
	if err != nil {
		t.Fatal(err)
	}
	goldenDir, err := filepath.Abs("testdata/bump/file_changed_api_unchanged/golden_output")
	if err != nil {
		t.Fatal(err)
	}

	setupRepoFromDir(t, inputDir)
	testhelper.RunGit(t, "tag", "a-v1.0.0")
	testhelper.RunGit(t, "tag", "b-v1.0.0")
	testhelper.RunGit(t, "tag", "c-v1.0.0")

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

	cfg, err := yaml.Read[config.Config]("librarian.yaml")
	if err != nil {
		t.Fatal(err)
	}

	err = Bump(t.Context(), cfg, true, "", "")
	if err != nil {
		t.Fatalf("Bump failed: %v", err)
	}

	if err := yaml.Write("librarian.yaml", cfg); err != nil {
		t.Fatal(err)
	}

	compareDirWithGolden(t, goldenDir)
}

func TestBump_UnpublishedLibrary(t *testing.T) {
	testhelper.RequireCommand(t, "git")
	testhelper.RequireCommand(t, "dart")

	inputDir, err := filepath.Abs("testdata/bump/input")
	if err != nil {
		t.Fatal(err)
	}
	goldenDir, err := filepath.Abs("testdata/bump/unpublished_c/golden_output")
	if err != nil {
		t.Fatal(err)
	}

	setupRepoFromDir(t, inputDir)
	testhelper.RunGit(t, "tag", "a-v1.0.0")
	testhelper.RunGit(t, "tag", "b-v1.0.0")

	// Now make a commit with changes to package c.
	if err := os.WriteFile("generated/c/lib.dart", []byte("// library c: new changes"), 0644); err != nil {
		t.Fatal(err)
	}
	testhelper.RunGit(t, "add", ".")
	testhelper.RunGit(t, "commit", "-m", "feat: added support for c", ".")

	// responses map only has "a" and "b" but NOT "c"!
	apiToolResponses := map[string]packageVersion{
		"a": {needed: "1.0.0", old: "1.0.0"},
		"b": {needed: "1.0.0", old: "1.0.0"},
	}
	setupFakeApitool(t, apiToolResponses)

	cfg, err := yaml.Read[config.Config]("librarian.yaml")
	if err != nil {
		t.Fatal(err)
	}

	err = Bump(t.Context(), cfg, true, "", "")
	if err != nil {
		t.Fatalf("Bump failed: %v", err)
	}

	if err := yaml.Write("librarian.yaml", cfg); err != nil {
		t.Fatal(err)
	}

	compareDirWithGolden(t, goldenDir)
}

func setupRepoFromDir(t *testing.T, sourceDir string) {
	t.Helper()

	absSourceDir, err := filepath.Abs(sourceDir)
	if err != nil {
		t.Fatal(err)
	}

	remoteDir := testhelper.SetupRepoWithChange(t, "release-2001-02-03")
	if err := command.Run(t.Context(), command.Git, "-C", remoteDir, "config", "receive.denyCurrentBranch", "ignore"); err != nil {
		t.Fatal(err)
	}
	testhelper.CloneRepository(t, remoteDir)

	copyDir(t, absSourceDir, ".")

	testhelper.RunGit(t, "add", ".")
	testhelper.RunGit(t, "commit", "-m", "feat: added files from template", ".")
	testhelper.RunGit(t, "push", config.RemoteUpstream, config.BranchMain)
}

func copyDir(t *testing.T, src, dst string) {
	t.Helper()
	err := filepath.WalkDir(src, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0644)
	})
	if err != nil {
		t.Fatal(err)
	}
}

func compareDirWithGolden(t *testing.T, goldenDir string) {
	t.Helper()

	absGoldenDir, err := filepath.Abs(goldenDir)
	if err != nil {
		t.Fatal(err)
	}

	err = filepath.WalkDir(absGoldenDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if d.Name() == ".dart_tool" {
				return filepath.SkipDir
			}
			return nil
		}

		relPath, err := filepath.Rel(absGoldenDir, path)
		if err != nil {
			return err
		}

		gotBytes, err := os.ReadFile(relPath)
		if err != nil {
			t.Errorf("expected file %s is missing in output: %v", relPath, err)
			return nil
		}

		wantBytes, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		if got, want := strings.TrimSpace(string(gotBytes)), strings.TrimSpace(string(wantBytes)); got != want {
			t.Errorf("file %s content mismatch:\ngot:\n%s\nwant:\n%s", relPath, got, want)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
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
		script.WriteString("  else\n")
		script.WriteString("    echo \"Package not available\" >&2\n")
		script.WriteString("    exit 1\n")
		script.WriteString("  fi\n")
	} else {
		script.WriteString("  echo \"Package not available\" >&2\n")
		script.WriteString("  exit 1\n")
	}
	script.WriteString("fi\n")

	setupFakeScript(t, "dart-apitool", script.String())
}
