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
