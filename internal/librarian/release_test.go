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
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/yaml"
)

func TestReleaseCommand(t *testing.T) {
	const testlib = "test-lib"
	const testlib2 = "test-lib2"

	for _, test := range []struct {
		name         string
		args         []string
		wantErr      error
		wantVersions map[string]string
	}{
		{
			name:    "no args",
			args:    []string{"librarian", "release"},
			wantErr: errMissingLibraryOrAllFlag,
		},
		{
			name:    "library name and all flag",
			args:    []string{"librarian", "release", testlib, "--all"},
			wantErr: errBothLibraryAndAllFlag,
		},
		{
			name: "library name",
			args: []string{"librarian", "release", testlib},
			wantVersions: map[string]string{
				testlib:  testReleaseVersion,
				testlib2: "0.1.0",
			},
		},
		{
			name: "all flag",
			args: []string{"librarian", "release", "--all"},
			wantVersions: map[string]string{
				testlib:  testReleaseVersion,
				testlib2: testReleaseVersion,
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			tempDir := t.TempDir()
			t.Chdir(tempDir)

			configPath := filepath.Join(tempDir, librarianConfigPath)
			configContent := fmt.Sprintf(`language: testhelper
libraries:
  - name: %s
    version: 0.1.0
  - name: %s
    version: 0.1.0
`, testlib, testlib2)
			if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
				t.Fatal(err)
			}

			err := Run(t.Context(), test.args...)
			if !errors.Is(err, test.wantErr) {
				t.Fatalf("Run() error = %v, wantErr %v", err, test.wantErr)
			}
			if test.wantErr != nil {
				return
			}

			if test.wantVersions != nil {
				cfg, err := yaml.Read[config.Config](configPath)
				if err != nil {
					t.Fatal(err)
				}
				gotVersions := make(map[string]string)
				for _, lib := range cfg.Libraries {
					gotVersions[lib.Name] = lib.Version
				}
				if diff := cmp.Diff(test.wantVersions, gotVersions); diff != "" {
					t.Errorf("mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}
}

func TestLibraryByName(t *testing.T) {
	for _, test := range []struct {
		name        string
		libraryName string
		config      *config.Config
		want        *config.Library
		wantErr     error
	}{
		{
			name:        "find_a_library",
			libraryName: "example-library",
			config: &config.Config{
				Libraries: []*config.Library{
					{Name: "example-library"},
					{Name: "another-library"},
				},
			},
			want: &config.Library{Name: "example-library"},
		},
		{
			name:        "no_library_in_config",
			libraryName: "example-library",
			config:      &config.Config{},
			wantErr:     errLibraryNotFound,
		},
		{
			name:        "does_not_find_a_library",
			libraryName: "non-existent-library",
			config: &config.Config{
				Libraries: []*config.Library{
					{Name: "example-library"},
					{Name: "another-library"},
				},
			},
			wantErr: errLibraryNotFound,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := libraryByName(test.config, test.libraryName)
			if test.wantErr != nil {
				if !errors.Is(err, test.wantErr) {
					t.Errorf("got error %v, want %v", err, test.wantErr)
				}
				return
			}
			if err != nil {
				t.Errorf("libraryByName(%q): %v", test.libraryName, err)
				return
			}
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestReleaseRust(t *testing.T) {
	origRustReleaseLibrary := rustReleaseLibrary
	origLibrarianGenerateLibrary := librarianGenerateLibrary
	defer func() {
		rustReleaseLibrary = origRustReleaseLibrary
		librarianGenerateLibrary = origLibrarianGenerateLibrary
	}()

	tests := []struct {
		name               string
		releaseError       error
		generateError      error
		wantReleaseCalled  bool
		wantGenerateCalled bool
		wantErr            bool
	}{
		{
			name:               "rust success",
			wantReleaseCalled:  true,
			wantGenerateCalled: true,
		},
		{
			name:               "generate error",
			wantReleaseCalled:  true,
			wantGenerateCalled: true,
			generateError:      errors.New("generate error"),
			wantErr:            true,
		},
		{
			name:               "rust release error",
			wantReleaseCalled:  true,
			wantGenerateCalled: false,
			releaseError:       errors.New("release error"),
			wantErr:            true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var (
				generateCalled bool
				releaseCalled  bool
			)
			rustReleaseLibrary = func(cfg *config.Config, library *config.Library) error {
				releaseCalled = true
				return test.releaseError
			}
			librarianGenerateLibrary = func(ctx context.Context, all bool, libraryName string) error {
				generateCalled = true
				return test.generateError
			}
			cfg := &config.Config{
				Language: "rust",
			}
			libConfg := &config.Library{}
			err := releaseLibrary(t.Context(), cfg, libConfg)

			if (err != nil) != test.wantErr {
				t.Fatalf("releaseLibrary() error = %v, wantErr %v", err, test.wantErr)
			}
			if releaseCalled != test.wantReleaseCalled {
				t.Errorf("releaseCalled = %v, want %v", releaseCalled, test.wantReleaseCalled)
			}
			if generateCalled != test.wantGenerateCalled {
				t.Errorf("generateCalled = %v, want %v", generateCalled, test.wantGenerateCalled)
			}
			if test.releaseError != nil && test.releaseError != err {
				t.Errorf("releaseError= %v, want %v", err, test.releaseError)

			}
			if test.generateError != nil && test.generateError != err {
				t.Errorf("generateError= %v, want %v", err, test.generateError)

			}
		})

	}
}
