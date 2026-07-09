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
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}
