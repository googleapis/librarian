// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package rust

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/sideflip/config"
)

func TestBumpVersions(t *testing.T) {
	tmpDir := t.TempDir()
	t.Chdir(tmpDir)

	createCrate(t, "src/storage", "google-cloud-storage", "1.0.0")
	createCrate(t, "src/secretmanager", "google-cloud-secretmanager-v1", "1.5.3")

	cfg := &config.Config{
		Version:  "v1",
		Language: "rust",
		Versions: map[string]string{
			"google-cloud-storage":          "1.0.0",
			"google-cloud-secretmanager-v1": "1.5.3",
		},
	}
	configPath := "librarian.yaml"
	if err := cfg.Write(configPath); err != nil {
		t.Fatal(err)
	}

	if err := BumpVersions(t.Context(), cfg, configPath); err != nil {
		t.Fatal(err)
	}

	checkCargoVersion(t, "src/storage/Cargo.toml", "1.1.0")
	checkCargoVersion(t, "src/secretmanager/Cargo.toml", "1.6.0")

	updatedCfg, err := config.Read(configPath)
	if err != nil {
		t.Fatal(err)
	}

	wantVersions := map[string]string{
		"google-cloud-storage":          "1.1.0",
		"google-cloud-secretmanager-v1": "1.6.0",
	}
	if diff := cmp.Diff(wantVersions, updatedCfg.Versions); diff != "" {
		t.Errorf("versions mismatch (-want +got):\n%s", diff)
	}
}

func createCrate(t *testing.T, dir, name, version string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}

	cargo := fmt.Sprintf(`[package]
name                   = "%s"
version                = "%s"
edition                = "2021"
`, name, version)

	if err := os.WriteFile(filepath.Join(dir, "Cargo.toml"), []byte(cargo), 0644); err != nil {
		t.Fatal(err)
	}
}

func checkCargoVersion(t *testing.T, path, wantVersion string) {
	t.Helper()
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	want := fmt.Sprintf(`version = '%s'`, wantVersion)
	if !contains(string(contents), want) {
		t.Errorf("%s does not contain %q\nGot:\n%s", path, want, string(contents))
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && containsInner(s, substr))
}

func containsInner(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
