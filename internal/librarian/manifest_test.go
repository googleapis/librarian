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

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
)

func TestLoadReleasePleaseManifest(t *testing.T) {
	t.Run("missing file returns empty map", func(t *testing.T) {
		got, err := loadReleasePleaseManifest(filepath.Join(t.TempDir(), "nonexistent.json"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(got) != 0 {
			t.Errorf("got %v, want empty map", got)
		}
	})

	t.Run("valid manifest file", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, config.ReleasePleaseManifest)
		content := `{"packages/google-cloud-memorystore": "0.7.0", ".": "1.2.3"}`
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
		got, err := loadReleasePleaseManifest(path)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		want := map[string]string{
			"packages/google-cloud-memorystore": "0.7.0",
			".":                                 "1.2.3",
		}
		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("manifest mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("invalid json returns error", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, config.ReleasePleaseManifest)
		if err := os.WriteFile(path, []byte("{invalid json}"), 0644); err != nil {
			t.Fatal(err)
		}
		_, err := loadReleasePleaseManifest(path)
		if err == nil {
			t.Error("expected error for invalid json, got nil")
		}
	})
}

func TestResolveVersion(t *testing.T) {
	manifest := map[string]string{
		"packages/google-cloud-memorystore": "0.7.0",
		"google-cloud-redis-cluster":        "0.12.0",
		".":                                 "1.0.0",
	}

	tests := []struct {
		name     string
		lib      *config.Library
		manifest map[string]string
		want     string
	}{
		{
			name: "matches lib output path",
			lib: &config.Library{
				Name:    "google-cloud-memorystore",
				Output:  "packages/google-cloud-memorystore",
				Version: "0.6.0",
			},
			manifest: manifest,
			want:     "0.7.0",
		},
		{
			name: "matches lib name",
			lib: &config.Library{
				Name:    "google-cloud-redis-cluster",
				Output:  "other/output/dir",
				Version: "0.11.0",
			},
			manifest: manifest,
			want:     "0.12.0",
		},
		{
			name: "matches single component root dot",
			lib: &config.Library{
				Name:    "single-pkg",
				Output:  "custom/output",
				Version: "0.5.0",
			},
			manifest: manifest,
			want:     "1.0.0",
		},
		{
			name: "falls back to lib version when manifest does not match",
			lib: &config.Library{
				Name:    "unknown-pkg",
				Output:  "unknown/output",
				Version: "2.0.0",
			},
			manifest: map[string]string{
				"packages/foo": "1.0.0",
			},
			want: "2.0.0",
		},
		{
			name: "falls back to lib version when manifest is empty",
			lib: &config.Library{
				Name:    "some-pkg",
				Output:  "some/output",
				Version: "3.1.4",
			},
			manifest: map[string]string{},
			want:     "3.1.4",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolveVersion(tt.lib, tt.manifest)
			if got != tt.want {
				t.Errorf("resolveVersion() = %q, want %q", got, tt.want)
			}
		})
	}
}
