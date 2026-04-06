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
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/serviceconfig"
	"github.com/googleapis/librarian/internal/yaml"
)

func TestCollapseLanguages(t *testing.T) {
	for _, test := range []struct {
		name             string
		restNumericEnums map[string]bool
		want             []string
	}{
		{
			name: "all languages present",
			restNumericEnums: map[string]bool{
				"csharp": true,
				"go":     true,
				"java":   true,
				"nodejs": true,
				"php":    true,
				"python": true,
				"ruby":   true,
			},
			want: []string{"all"},
		},
		{
			name: "some present",
			restNumericEnums: map[string]bool{
				"java":   true,
				"python": true,
			},
			want: []string{"java", "python"},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := collapseLanguages(test.restNumericEnums)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestRunUpdateRestNumericEnums(t *testing.T) {
	for _, test := range []struct {
		name          string
		original      []*serviceconfig.API
		googleapisDir string
		want          []*serviceconfig.API
	}{
		{
			name:          "add cloud api",
			googleapisDir: "testdata/test-update-rne/add-cloud-api",
			original: []*serviceconfig.API{
				{
					Path: "google/cloud/servicemanager/v1",
					Transports: map[string]serviceconfig.Transport{
						config.LanguageAll: serviceconfig.GRPC,
					},
				},
			},
			want: []*serviceconfig.API{
				{
					Path: "google/cloud/servicemanager/v1",
					Transports: map[string]serviceconfig.Transport{
						config.LanguageAll: serviceconfig.GRPC,
					},
				},
				{
					Languages: []string{
						config.LanguageDart,
						config.LanguageGo,
						config.LanguageJava,
						config.LanguagePython,
						config.LanguageRust,
					},
					Path:                 "google/cloud/workstations/v1",
					SkipRESTNumericEnums: []string{"all"},
				},
			},
		},
		{
			name:          "add non-cloud api",
			googleapisDir: "testdata/test-update-rne/add-non-cloud-api",
			original: []*serviceconfig.API{
				{
					Languages: []string{
						config.LanguageDart,
						config.LanguageGo,
						config.LanguageJava,
					},
					Path: "google/non-cloud/v1",
					Transports: map[string]serviceconfig.Transport{
						config.LanguageAll: serviceconfig.GRPC,
					},
				},
			},
			want: []*serviceconfig.API{
				{
					Languages: []string{
						config.LanguageDart,
						config.LanguageGo,
						config.LanguageJava,
					},
					SkipRESTNumericEnums: []string{config.LanguageCsharp, config.LanguageGo},
					Path:                 "google/non-cloud/v1",
					Transports: map[string]serviceconfig.Transport{
						config.LanguageAll: serviceconfig.GRPC,
					},
				},
			},
		},
		{
			name:          "no change cloud api",
			googleapisDir: "testdata/test-update-rne/no-change-cloud-api",
			original: []*serviceconfig.API{
				{
					Path: "google/cloud/workstations/v1",
					Transports: map[string]serviceconfig.Transport{
						config.LanguageAll: serviceconfig.GRPC,
					},
				},
			},
			// No change because all languages have rest_numeric_enums,
			// which is the default value.
			want: []*serviceconfig.API{
				{
					Path: "google/cloud/workstations/v1",
					Transports: map[string]serviceconfig.Transport{
						config.LanguageAll: serviceconfig.GRPC,
					},
				},
			},
		},
		{
			name:          "no change non-cloud api",
			googleapisDir: "testdata/test-update-rne/no-change-non-cloud-api",
			original: []*serviceconfig.API{
				{
					Languages: []string{
						config.LanguageDart,
						config.LanguageGo,
						config.LanguageJava,
					},
					Path: "google/another-non-cloud/v1",
				},
			},
			// No change because the non-cloud api is not listed in
			// the original sdk.yaml.
			want: []*serviceconfig.API{
				{
					Languages: []string{
						config.LanguageDart,
						config.LanguageGo,
						config.LanguageJava,
					},
					Path: "google/another-non-cloud/v1",
				},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			sdkYaml := filepath.Join(tmpDir, "sdk.yaml")
			if err := yaml.Write(sdkYaml, test.original); err != nil {
				t.Fatal(err)
			}

			if err := runUpdateRestNumericEnums(sdkYaml, test.googleapisDir); err != nil {
				t.Fatal(err)
			}
			got, err := yaml.Read[[]*serviceconfig.API](sdkYaml)
			if err != nil {
				t.Fatal(err)
			}

			if diff := cmp.Diff(test.want, *got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestRunUpdateRestNumericEnums_Error(t *testing.T) {
	for _, test := range []struct {
		name          string
		sdkYaml       string
		googleapisDir string
		setup         func(*testing.T, string)
		wantErr       error
	}{
		{
			name:    "invalid yaml file",
			sdkYaml: "non-existent.yaml",
			wantErr: os.ErrNotExist,
		},
		{
			name:    "invalid googleapis dir",
			sdkYaml: filepath.Join(t.TempDir(), "empty.yaml"),
			setup: func(t *testing.T, path string) {
				if err := os.WriteFile(path, []byte(""), 0755); err != nil {
					t.Fatal(err)
				}
			},
			wantErr: os.ErrNotExist,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			if test.setup != nil {
				test.setup(t, test.sdkYaml)
			}
			err := runUpdateRestNumericEnums(test.sdkYaml, test.googleapisDir)
			if !errors.Is(err, test.wantErr) {
				t.Errorf("got error %v, want %v", err, test.wantErr)
			}
		})
	}
}

func TestReadRestNumericEnums_Error(t *testing.T) {
	tmpDir := t.TempDir()
	got := readSkipRESTNumericEnums(tmpDir, "missing")
	if got != nil {
		t.Errorf("readSkipRESTNumericEnums() = %v, want nil", got)
	}
}

func TestUpdateRestNumericEnumsCommand(t *testing.T) {
	cmd := updateRestNumericEnumsCommand()
	ctx := t.Context()
	// Just test that the command can be initialized and run with --help.
	if err := cmd.Run(ctx, []string{"update-rest-numeric-enums", "--help"}); err != nil {
		t.Fatalf("cmd.Run() error = %v", err)
	}
}
