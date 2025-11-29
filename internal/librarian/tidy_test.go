// Copyright 2025 Google LLC
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
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/yaml"
)

func TestValidateLibraries(t *testing.T) {
	for _, test := range []struct {
		name      string
		libraries []*config.Library
		wantErr   error
	}{
		{
			name: "valid libraries",
			libraries: []*config.Library{
				{Name: "google-cloud-secretmanager-v1"},
				{Name: "google-cloud-storage-v1"},
			},
		},
		{
			name: "duplicate library names",
			libraries: []*config.Library{
				{Name: "google-cloud-secretmanager-v1"},
				{Name: "google-cloud-secretmanager-v1"},
			},
			wantErr: errDuplicateLibraryName,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			cfg := &config.Config{Libraries: test.libraries}
			err := validateLibraries(cfg)
			if test.wantErr == nil {
				if err != nil {
					t.Fatal(err)
				}
				return
			}
			if err == nil {
				t.Fatalf("expected %v, got nil", test.wantErr)
			}
			if !errors.Is(err, test.wantErr) {
				t.Errorf("expected %v, got %v", test.wantErr, err)
			}
		})
	}
}

func TestFormatConfig(t *testing.T) {
	cfg := formatConfig(&config.Config{
		Libraries: []*config.Library{
			{Name: "google-cloud-storage-v1", Version: "1.0.0"},
			{Name: "google-cloud-bigquery-v1", Version: "2.0.0"},
			{Name: "google-cloud-secretmanager-v1", Version: "3.0.0"},
		},
	})
	want := []string{
		"google-cloud-bigquery-v1",
		"google-cloud-secretmanager-v1",
		"google-cloud-storage-v1",
	}
	var got []string
	for _, lib := range cfg.Libraries {
		got = append(got, lib.Name)
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestTidyCommand(t *testing.T) {
	tempDir := t.TempDir()
	t.Chdir(tempDir)
	configPath := filepath.Join(tempDir, librarianConfigPath)
	configContent := `language: rust
sources:
  googleapis:
    commit: abc123
libraries:
  - name: google-cloud-storage-v1
    version: "1.0.0"
  - name: google-cloud-bigquery-v1
    version: "2.0.0"
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatal(err)
	}
	if err := Run(t.Context(), "librarian", "tidy"); err != nil {
		t.Fatal(err)
	}

	cfg, err := yaml.Read[config.Config](configPath)
	if err != nil {
		t.Fatal(err)
	}

	var got []string
	for _, lib := range cfg.Libraries {
		got = append(got, lib.Name)
	}
	want := []string{
		"google-cloud-bigquery-v1",
		"google-cloud-storage-v1",
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestTidyCommandDuplicateError(t *testing.T) {
	tempDir := t.TempDir()
	t.Chdir(tempDir)
	configPath := filepath.Join(tempDir, librarianConfigPath)
	configContent := `language: rust
sources:
  googleapis:
    commit: abc123
libraries:
  - name: google-cloud-storage-v1
  - name: google-cloud-storage-v1
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatal(err)
	}
	err := Run(t.Context(), "librarian", "tidy")
	if err == nil {
		t.Fatal("expected error for duplicate library")
	}
	if !errors.Is(err, errDuplicateLibraryName) {
		t.Errorf("expected %v, got %v", errDuplicateLibraryName, err)
	}
}
