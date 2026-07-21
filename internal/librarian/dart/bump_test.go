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
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/googleapis/librarian/internal/config"
)

func TestUpdatePubspecDependencyVersions(t *testing.T) {
	tempDir := t.TempDir()
	libDir := filepath.Join(tempDir, "my_library")
	if err := os.MkdirAll(libDir, 0755); err != nil {
		t.Fatal(err)
	}

	pubspecPath := filepath.Join(libDir, "pubspec.yaml")
	initialContent := `name: my_library
version: 1.0.0
dependencies:
  sdk: ">=3.0.0 <4.0.0"
  google_cloud_protobuf: ^0.5.0
  another_dep: ^1.0.0
`
	if err := os.WriteFile(pubspecPath, []byte(initialContent), 0644); err != nil {
		t.Fatal(err)
	}

	lib := &config.Library{
		Name:   "my_library",
		Output: libDir,
	}

	newDeps := map[string]string{
		"package:google_cloud_protobuf": "^0.6.0",
		"package:another_dep":           "^1.2.0",
	}

	if err := updatePubspecDependencyVersions(lib, nil, newDeps); err != nil {
		t.Fatalf("updatePubspecDependencyVersions failed: %v", err)
	}

	content, err := os.ReadFile(pubspecPath)
	if err != nil {
		t.Fatal(err)
	}

	got := string(content)
	want := `name: my_library
version: 1.0.0
dependencies:
  sdk: ">=3.0.0 <4.0.0"
  google_cloud_protobuf: ^0.6.0
  another_dep: ^1.2.0
`
	if got != want {
		t.Errorf("pubspec.yaml content mismatch:\ngot:\n%s\nwant:\n%s", got, want)
	}
}

func TestUpdatePubspecDependencyVersions_Defaults(t *testing.T) {
	tempDir := t.TempDir()

	// Create outputs directory to act as defaults.Output
	outputsDir := filepath.Join(tempDir, "outputs")
	libDir := filepath.Join(outputsDir, "my_library")
	if err := os.MkdirAll(libDir, 0755); err != nil {
		t.Fatal(err)
	}

	pubspecPath := filepath.Join(libDir, "pubspec.yaml")
	initialContent := `name: my_library
version: 1.0.0
dependencies:
  google_cloud_protobuf: ^0.5.0
`
	if err := os.WriteFile(pubspecPath, []byte(initialContent), 0644); err != nil {
		t.Fatal(err)
	}

	lib := &config.Library{
		Name: "my_library",
		// Output is empty, so it should fall back to defaults.Output/lib.Name
	}
	defaults := &config.Default{
		Output: outputsDir,
	}

	newDeps := map[string]string{
		"package:google_cloud_protobuf": "^0.6.0",
	}

	if err := updatePubspecDependencyVersions(lib, defaults, newDeps); err != nil {
		t.Fatalf("updatePubspecDependencyVersions failed: %v", err)
	}

	content, err := os.ReadFile(pubspecPath)
	if err != nil {
		t.Fatal(err)
	}

	got := string(content)
	want := `name: my_library
version: 1.0.0
dependencies:
  google_cloud_protobuf: ^0.6.0
`
	if got != want {
		t.Errorf("pubspec.yaml content mismatch:\ngot:\n%s\nwant:\n%s", got, want)
	}
}

func TestUpdateChangelog_New(t *testing.T) {
	tempDir := t.TempDir()

	err := updateChangelog(tempDir, "1.2.3", []string{"feat: added support for something", "fix: resolved a bug"})
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

- feat: added support for something
- fix: resolved a bug

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

	err := updateChangelog(tempDir, "1.2.3", []string{"feat: added support for something", "fix: resolved a bug"})
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

- feat: added support for something
- fix: resolved a bug

## 1.2.2

- chore: release 1.2.2
`
	if got != want {
		t.Errorf("CHANGELOG.md content mismatch:\ngot:\n%s\nwant:\n%s", got, want)
	}
}

func TestUpdatePubspecDependencyVersions_PreservesComments(t *testing.T) {
	tempDir := t.TempDir()
	libDir := filepath.Join(tempDir, "my_library")
	if err := os.MkdirAll(libDir, 0755); err != nil {
		t.Fatal(err)
	}

	pubspecPath := filepath.Join(libDir, "pubspec.yaml")
	initialContent := `# A top level comment
name: my_library
version: 1.0.0

dependencies:
  # This is the SDK
  sdk: ">=3.0.0 <4.0.0"

  # The main protobuf dependency
  google_cloud_protobuf: ^0.5.0 # inline comment
  another_dep: ^1.0.0
`
	if err := os.WriteFile(pubspecPath, []byte(initialContent), 0644); err != nil {
		t.Fatal(err)
	}

	lib := &config.Library{
		Name:   "my_library",
		Output: libDir,
	}

	newDeps := map[string]string{
		"package:google_cloud_protobuf": "^0.6.0",
		"package:another_dep":           "^1.2.0",
	}

	if err := updatePubspecDependencyVersions(lib, nil, newDeps); err != nil {
		t.Fatalf("updatePubspecDependencyVersions failed: %v", err)
	}

	content, err := os.ReadFile(pubspecPath)
	if err != nil {
		t.Fatal(err)
	}

	got := string(content)
	if !strings.Contains(got, "# A top level comment") {
		t.Errorf("Expected top level comment to be preserved, got:\n%s", got)
	}
	if !strings.Contains(got, "# This is the SDK") {
		t.Errorf("Expected SDK comment to be preserved, got:\n%s", got)
	}
	if !strings.Contains(got, "# The main protobuf dependency") {
		t.Errorf("Expected dependency comment to be preserved, got:\n%s", got)
	}
	if !strings.Contains(got, "# inline comment") {
		t.Errorf("Expected inline comment to be preserved, got:\n%s", got)
	}
	if !strings.Contains(got, "google_cloud_protobuf: ^0.6.0") && !strings.Contains(got, "google_cloud_protobuf: \"^0.6.0\"") {
		t.Errorf("Expected dependency version to be updated, got:\n%s", got)
	}
}

func TestUpdatePubspecVersion_PreservesComments(t *testing.T) {
	tempDir := t.TempDir()
	pubspecPath := filepath.Join(tempDir, "pubspec.yaml")
	initialContent := `# Top level comment
name: my_library
version: 1.0.0 # inline version comment
dependencies:
  sdk: ">=3.0.0 <4.0.0"
`
	if err := os.WriteFile(pubspecPath, []byte(initialContent), 0644); err != nil {
		t.Fatal(err)
	}

	if err := updatePubspecVersion(pubspecPath, "1.1.0"); err != nil {
		t.Fatalf("updatePubspecVersion failed: %v", err)
	}

	content, err := os.ReadFile(pubspecPath)
	if err != nil {
		t.Fatal(err)
	}

	got := string(content)
	if !strings.Contains(got, "# Top level comment") {
		t.Errorf("Expected top level comment to be preserved, got:\n%s", got)
	}
	if !strings.Contains(got, "# inline version comment") {
		t.Errorf("Expected inline version comment to be preserved, got:\n%s", got)
	}
	if !strings.Contains(got, "version: 1.1.0") && !strings.Contains(got, "version: \"1.1.0\"") {
		t.Errorf("Expected version to be updated, got:\n%s", got)
	}
}
