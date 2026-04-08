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
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/serviceconfig"
	"github.com/googleapis/librarian/internal/yaml"
)

func TestRunUpdateReleaseLevel(t *testing.T) {
	for _, test := range []struct {
		name          string
		original      []*serviceconfig.API
		googleapisDir string
		want          []*serviceconfig.API
	}{
		{
			name:          "add cloud api",
			googleapisDir: "testdata/test-update-rl/add-cloud-api",
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
					ReleaseLevels: map[string]string{
						config.LanguageGo:   "beta",
						config.LanguageJava: "ga",
					},
				},
			},
		},
		{
			name:          "add non-cloud api",
			googleapisDir: "testdata/test-update-rl/add-non-cloud-api",
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
					ReleaseLevels: map[string]string{
						config.LanguageGo: "beta",
					},
					Path: "google/non-cloud/v1",
					Transports: map[string]serviceconfig.Transport{
						config.LanguageAll: serviceconfig.GRPC,
					},
				},
			},
		},
		{
			name:          "no change cloud api",
			googleapisDir: "testdata/test-update-rl/no-change-cloud-api",
			original: []*serviceconfig.API{
				{
					Path: "google/cloud/workstations/v1",
					Transports: map[string]serviceconfig.Transport{
						config.LanguageAll: serviceconfig.GRPC,
					},
				},
			},
			// No change because Go has a default value.
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
			googleapisDir: "testdata/test-update-rl/no-change-non-cloud-api",
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

			if err := runUpdateReleaseLevel(sdkYaml, test.googleapisDir); err != nil {
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

func TestRunUpdateReleaseLevel_Error(t *testing.T) {
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
			wantErr: fs.ErrNotExist,
		},
		{
			name:    "invalid googleapis dir",
			sdkYaml: filepath.Join(t.TempDir(), "empty.yaml"),
			setup: func(t *testing.T, path string) {
				if err := os.WriteFile(path, []byte(""), 0755); err != nil {
					t.Fatal(err)
				}
			},
			wantErr: fs.ErrNotExist,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			if test.setup != nil {
				test.setup(t, test.sdkYaml)
			}
			err := runUpdateReleaseLevel(test.sdkYaml, test.googleapisDir)
			if !errors.Is(err, test.wantErr) {
				t.Errorf("got error %v, want %v", err, test.wantErr)
			}
		})
	}
}

func TestUpdateReleaseLevelCommand(t *testing.T) {
	cmd := updateReleaseLevelCommand()
	ctx := t.Context()
	// Just test that the command can be initialized and run with --help.
	if err := cmd.Run(ctx, []string{"update-release-level", "--help"}); err != nil {
		t.Fatalf("cmd.Run() error = %v", err)
	}
}
