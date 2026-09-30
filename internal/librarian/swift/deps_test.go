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
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
)

func TestExtractInternalPackageDependencies(t *testing.T) {
	manifest := `
// swift-tools-version: 6.2
import PackageDescription

let package = Package(
  name: "GoogleCloudStorage",
  dependencies: [
    localOrRemotePackage(
      url: "https://github.com/googleapis/swift-google-auth",
      path: "pkgs/swift-google-auth",
      from: "0.4.0"
    ),
    localOrRemotePackage(url: "https://github.com/googleapis/swift-google-gax", path: "pkgs/swift-google-gax", from: "0.4.0"),
    .package(url: "https://github.com/apple/swift-log", from: "1.14.0"),
    localOrRemotePackage(
      path: "generated/swift-google-iam-v1",
      url: "https://github.com/googleapis/swift-google-iam-v1",
      from: "0.4.0"
    ),
  ]
)
`
	got := extractInternalPackageDependencies(manifest)
	want := []string{"generated/swift-google-iam-v1", "pkgs/swift-google-auth", "pkgs/swift-google-gax"}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("extractInternalPackageDependencies() mismatch (-want +got):\n%s", diff)
	}
}

func TestTopologicalLevelsIndependent(t *testing.T) {
	libs := []*config.Library{
		{Name: "auth"},
		{Name: "wkt"},
		{Name: "storage"},
	}
	deps := map[string][]string{
		"auth":    nil,
		"wkt":     nil,
		"storage": nil,
	}

	levels, err := topologicalLevels(libs, deps)
	if err != nil {
		t.Fatalf("topologicalLevels failed: %v", err)
	}
	if len(levels) != 1 {
		t.Fatalf("got %d levels, want 1", len(levels))
	}
	var gotNames []string
	for _, l := range levels[0] {
		gotNames = append(gotNames, l.Name)
	}
	wantNames := []string{"auth", "storage", "wkt"}
	if diff := cmp.Diff(wantNames, gotNames); diff != "" {
		t.Errorf("level 0 mismatch (-want +got):\n%s", diff)
	}
}

func TestTopologicalLevelsChain(t *testing.T) {
	libs := []*config.Library{
		{Name: "auth"},
		{Name: "gax"},
		{Name: "iam"},
		{Name: "secretmanager"},
	}
	deps := map[string][]string{
		"auth":          nil,
		"gax":           {"auth"},
		"iam":           {"gax"},
		"secretmanager": {"iam"},
	}

	levels, err := topologicalLevels(libs, deps)
	if err != nil {
		t.Fatalf("topologicalLevels failed: %v", err)
	}
	if len(levels) != 4 {
		t.Fatalf("got %d levels, want 4", len(levels))
	}
	checkLevel(t, levels[0], []string{"auth"})
	checkLevel(t, levels[1], []string{"gax"})
	checkLevel(t, levels[2], []string{"iam"})
	checkLevel(t, levels[3], []string{"secretmanager"})
}

func TestTopologicalLevelsDiamond(t *testing.T) {
	libs := []*config.Library{
		{Name: "a"},
		{Name: "b"},
		{Name: "c"},
		{Name: "d"},
	}
	deps := map[string][]string{
		"a": nil,
		"b": {"a"},
		"c": {"a"},
		"d": {"b", "c"},
	}

	levels, err := topologicalLevels(libs, deps)
	if err != nil {
		t.Fatalf("topologicalLevels failed: %v", err)
	}
	if len(levels) != 3 {
		t.Fatalf("got %d levels, want 3", len(levels))
	}
	checkLevel(t, levels[0], []string{"a"})
	checkLevel(t, levels[1], []string{"b", "c"})
	checkLevel(t, levels[2], []string{"d"})
}

func TestTopologicalLevelsCycleDetection(t *testing.T) {
	libs := []*config.Library{
		{Name: "a"},
		{Name: "b"},
		{Name: "c"},
	}
	deps := map[string][]string{
		"a": {"b"},
		"b": {"c"},
		"c": {"a"},
	}

	_, err := topologicalLevels(libs, deps)
	if err == nil {
		t.Fatal("expected cycle detection error, got nil")
	}
}

func TestBuildDependencyGraph(t *testing.T) {
	t.Chdir(t.TempDir())

	authDir := filepath.Join("pkgs", "swift-google-auth")
	gaxDir := filepath.Join("pkgs", "swift-google-gax")
	if err := os.MkdirAll(authDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(gaxDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(authDir, "Package.swift"), []byte(`// Package.swift for auth`), 0o644); err != nil {
		t.Fatal(err)
	}
	gaxManifest := `
localOrRemotePackage(
  url: "https://github.com/googleapis/swift-google-auth",
  path: "pkgs/swift-google-auth",
  from: "0.4.0"
)
`
	if err := os.WriteFile(filepath.Join(gaxDir, "Package.swift"), []byte(gaxManifest), 0o644); err != nil {
		t.Fatal(err)
	}

	authLib := &config.Library{Name: "auth", Output: authDir}
	gaxLib := &config.Library{Name: "gax", Output: gaxDir}
	cfg := &config.Config{
		Libraries: []*config.Library{authLib, gaxLib},
	}

	deps, err := buildDependencyGraph(t.Context(), cfg, []*config.Library{authLib, gaxLib}, "")
	if err != nil {
		t.Fatalf("buildDependencyGraph failed: %v", err)
	}

	if len(deps["auth"]) != 0 {
		t.Errorf("auth deps = %v, want empty", deps["auth"])
	}
	wantGaxDeps := []string{"auth"}
	if !slices.Equal(deps["gax"], wantGaxDeps) {
		t.Errorf("gax deps = %v, want %v", deps["gax"], wantGaxDeps)
	}
}

func TestDumpPackageParsing(t *testing.T) {
	rawJSON := `{
  "dependencies": [
    {
      "fileSystem": [
        {
          "identity": "swift-google-auth",
          "path": "/path/to/pkgs/swift-google-auth"
        }
      ]
    },
    {
      "fileSystem": [
        {
          "identity": "swift-google-wkt",
          "path": "/path/to/pkgs/swift-google-wkt"
        }
      ]
    },
    {
      "sourceControl": [
        {
          "identity": "swift-log"
        }
      ]
    }
  ]
}`
	var dump struct {
		Dependencies []struct {
			FileSystem []struct {
				Identity string `json:"identity"`
				Path     string `json:"path"`
			} `json:"fileSystem"`
		} `json:"dependencies"`
	}
	if err := json.Unmarshal([]byte(rawJSON), &dump); err != nil {
		t.Fatalf("json.Unmarshal failed: %v", err)
	}
	var paths []string
	for _, dep := range dump.Dependencies {
		for _, fs := range dep.FileSystem {
			if fs.Path != "" {
				paths = append(paths, fs.Path)
			} else if fs.Identity != "" {
				paths = append(paths, fs.Identity)
			}
		}
	}
	want := []string{"/path/to/pkgs/swift-google-auth", "/path/to/pkgs/swift-google-wkt"}
	if diff := cmp.Diff(want, paths); diff != "" {
		t.Errorf("dump paths mismatch (-want +got):\n%s", diff)
	}
}

func checkLevel(t *testing.T, level []*config.Library, wantNames []string) {
	t.Helper()
	var got []string
	for _, l := range level {
		got = append(got, l.Name)
	}
	if diff := cmp.Diff(wantNames, got); diff != "" {
		t.Errorf("level mismatch (-want +got):\n%s", diff)
	}
}
