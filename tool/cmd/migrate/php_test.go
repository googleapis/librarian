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

package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/yaml"
)

func TestRunPHPMigration(t *testing.T) {
	oldFetchSource := fetchSource
	t.Cleanup(func() {
		fetchSource = oldFetchSource
	})
	absGoogleapis, err := filepath.Abs("../../internal/testdata/googleapis")
	if err != nil {
		t.Fatal(err)
	}
	// Override fetchSource.
	fetchSource = func(ctx context.Context) (*config.Source, error) {
		return &config.Source{
			Commit: "abcd123",
			SHA256: "sha123",
			Dir:    absGoogleapis,
		}, nil
	}
	dir := t.TempDir()
	// Create a fake library SecretManager.
	libDir := filepath.Join(dir, "SecretManager")
	if err := os.Mkdir(libDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(libDir, "VERSION"), []byte("2.3.0\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(libDir, "composer.json"), []byte("{}"), 0644); err != nil {
		t.Fatal(err)
	}
	// Create a fake non-library directory to ensure it is ignored.
	ignoredDir := filepath.Join(dir, "dev")
	if err := os.Mkdir(ignoredDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(ignoredDir, "composer.json"), []byte("{}"), 0644); err != nil {
		t.Fatal(err)
	}
	t.Chdir(dir)
	err = runPHPMigration(t.Context(), ".")
	if err != nil {
		t.Fatal(err)
	}
	// Verify librarian.yaml is written and contains the expected content.
	got, err := yaml.Read[config.Config](config.LibrarianYAML)
	if err != nil {
		t.Fatalf("reading generated librarian.yaml: %v", err)
	}
	want := &config.Config{
		Language: config.LanguagePhp,
		Sources: &config.Sources{
			Googleapis: &config.Source{
				Commit: "abcd123",
				SHA256: "sha123",
			},
		},
		Libraries: []*config.Library{
			{
				Name:    "SecretManager",
				Version: "2.3.0",
			},
		},
		Tools: &config.Tools{
			Composer: []*config.ComposerTool{
				{
					Name:     "google/gapic-generator-php",
					Version:  "v1.21.2",
					Package:  "https://github.com/googleapis/gapic-generator-php/archive/refs/tags/v1.21.2.tar.gz",
					Checksum: "29635b02c6e505fe31cba2f88ae999f00d2710fe1d65cb7cad521a82e7c5a518",
					Build:    []string{"composer install"},
				},
			},
		},
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}
