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
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/serviceconfig"
	"github.com/googleapis/librarian/internal/yaml"
)

func TestSimplifyRestNumericEnums(t *testing.T) {
	for _, test := range []struct {
		name             string
		restNumericEnums map[string]bool
		want             map[string]bool
	}{
		{
			name: "all true",
			restNumericEnums: map[string]bool{
				"csharp": true,
				"go":     true,
				"java":   true,
				"nodejs": true,
				"php":    true,
				"python": true,
				"ruby":   true,
			},
			want: map[string]bool{"all": true},
		},
		{
			name: "some present, different values",
			restNumericEnums: map[string]bool{
				"java":   true,
				"python": true,
			},
			want: map[string]bool{
				"java":   true,
				"python": true,
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := simplifyRESTNumericEnums(test.restNumericEnums)
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
					Path: "google/cloud/workstations/v1",
					NoRESTNumericEnums: map[string]bool{
						"all": true,
					},
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
	tmpDir := t.TempDir()
	googleapisDir := filepath.Join(tmpDir, "googleapis")

	for _, test := range []struct {
		name      string
		apiGo     string
		apiGoPath string
	}{
		{
			name:      "invalid go file",
			apiGo:     "invalid go code",
			apiGoPath: filepath.Join(tmpDir, "invalid.go"),
		},
		{
			name:      "missing APIs variable",
			apiGo:     "package foo\nvar Other = 1",
			apiGoPath: filepath.Join(tmpDir, "missing_apis.go"),
		},
		{
			name:      "non-existent apiGoPath",
			apiGoPath: filepath.Join(tmpDir, "non_existent.go"),
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			if test.apiGo != "" {
				if err := os.WriteFile(test.apiGoPath, []byte(test.apiGo), 0644); err != nil {
					t.Fatal(err)
				}
			}
			if err := runUpdateRestNumericEnums(test.apiGoPath, googleapisDir); err == nil {
				t.Error("runUpdateRestNumericEnums() error = nil, want error")
			}
		})
	}
}

func TestReadRestNumericEnums_Error(t *testing.T) {
	tmpDir := t.TempDir()
	got := readRestNumericEnums(tmpDir, "missing")
	if got != nil {
		t.Errorf("readRestNumericEnums() = %v, want nil", got)
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
