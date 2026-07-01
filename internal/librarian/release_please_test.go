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
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
)

func TestHasBulkReleasePleaseConfigs(t *testing.T) {
	for _, test := range []struct {
		name           string
		language       string
		createConfig   bool
		createManifest bool
		want           bool
	}{
		{
			name:           "both missing (Go)",
			language:       config.LanguageGo,
			createConfig:   false,
			createManifest: false,
			want:           false,
		},
		{
			name:           "config missing (Go)",
			language:       config.LanguageGo,
			createConfig:   false,
			createManifest: true,
			want:           false,
		},
		{
			name:           "manifest missing (Go)",
			language:       config.LanguageGo,
			createConfig:   true,
			createManifest: false,
			want:           false,
		},
		{
			name:           "both exist (Go)",
			language:       config.LanguageGo,
			createConfig:   true,
			createManifest: true,
			want:           true,
		},
		{
			name:           "both missing (Nodejs)",
			language:       config.LanguageNodejs,
			createConfig:   false,
			createManifest: false,
			want:           false,
		},
		{
			name:           "config missing (Nodejs)",
			language:       config.LanguageNodejs,
			createConfig:   false,
			createManifest: true,
			want:           false,
		},
		{
			name:           "manifest missing (Nodejs)",
			language:       config.LanguageNodejs,
			createConfig:   true,
			createManifest: false,
			want:           false,
		},
		{
			name:           "both exist (Nodejs)",
			language:       config.LanguageNodejs,
			createConfig:   true,
			createManifest: true,
			want:           true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			tmp := t.TempDir()
			manifestFile, configFile := releasePleaseFiles(
				&config.Config{
					Language: test.language,
				},
			)
			if test.createConfig {
				if err := os.WriteFile(filepath.Join(tmp, configFile), []byte("{}"), 0644); err != nil {
					t.Fatal(err)
				}
			}
			if test.createManifest {
				if err := os.WriteFile(filepath.Join(tmp, manifestFile), []byte("{}"), 0644); err != nil {
					t.Fatal(err)
				}
			}
			got := hasBulkReleasePleaseConfigs(tmp, &config.Config{Language: test.language})
			if got != test.want {
				t.Errorf("hasBulkReleasePleaseConfigs(%s, %s) = %t, want %t", tmp, test.language, got, test.want)
			}
		})
	}
}

func TestSyncToReleasePlease(t *testing.T) {
	for _, test := range []struct {
		name            string
		language        string
		initialManifest string
		initialConfig   string
		library         *config.Library
		wantManifest    string
		wantConfig      string
	}{
		{
			name:            "new go library",
			language:        config.LanguageGo,
			initialManifest: `{}`,
			initialConfig:   `{"packages": {}}`,
			library: &config.Library{
				Name:    "secretmanager",
				Version: "1.0.0",
				APIs: []*config.API{
					{Path: "google/cloud/secretmanager/v1"},
				},
			},
			wantManifest: `{"secretmanager":"1.0.0"}`,
			wantConfig:   `{"packages":{"secretmanager":{"component":"secretmanager"}}}`,
		},
		{
			name:            "new nodejs library",
			language:        config.LanguageNodejs,
			initialManifest: `{}`,
			initialConfig:   `{"packages": {}}`,
			library: &config.Library{
				Name:    "google-cloud-secretmanager",
				Version: "1.0.0",
				APIs: []*config.API{
					{Path: "google/cloud/secretmanager/v1"},
				},
			},
			wantManifest: `{"packages/google-cloud-secretmanager":"1.0.0"}`,
			wantConfig:   `{"packages":{"packages/google-cloud-secretmanager":{}}}`,
		},

		{
			name:            "new python library",
			language:        config.LanguagePython,
			initialManifest: `{}`,
			initialConfig:   `{"packages": {}}`,
			library: &config.Library{
				Name:    "google-cloud-secretmanager",
				Version: "0.0.0",
				APIs: []*config.API{
					{Path: "google/cloud/secretmanager/v1"},
				},
			},
			wantManifest: `{"packages/google-cloud-secretmanager":"0.0.0"}`,
			wantConfig: `{
				"packages": {
					"packages/google-cloud-secretmanager": {
						"component": "google-cloud-secretmanager",
						"extra-files": [
							"google/cloud/secretmanager/gapic_version.py",
							"google/cloud/secretmanager_v1/gapic_version.py",
							{
								"jsonpath": "$.clientLibrary.version",
								"path": "samples/generated_samples/snippet_metadata_google.cloud.secretmanager.v1.json",
								"type": "json"
							}
						]
					}
				}
			}`,
		},
		{
			name:            "update existing python library (merge, deduplicate, sort)",
			language:        config.LanguagePython,
			initialManifest: `{"packages/google-cloud-secretmanager":"1.0.0"}`,
			initialConfig: `{
				"packages": {
					"packages/google-cloud-secretmanager": {
						"component": "google-cloud-secretmanager",
						"release-type": "python",
						"extra-files": [
							"google/cloud/secretmanager/gapic_version.py",
							"google/cloud/secretmanager_v1/gapic_version.py",
							{
								"jsonpath": "$.clientLibrary.version",
								"path": "samples/generated_samples/snippet_metadata_google.cloud.secretmanager.v1.json",
								"type": "json"
							}
						]
					}
				}
			}`,
			library: &config.Library{
				Name:    "google-cloud-secretmanager",
				Version: "1.0.0",
				APIs: []*config.API{
					{Path: "google/cloud/secretmanager/v1"},
					{Path: "google/cloud/secretmanager/v1beta1"},
				},
			},
			wantManifest: `{"packages/google-cloud-secretmanager":"1.0.0"}`,
			wantConfig: `{
				"packages": {
					"packages/google-cloud-secretmanager": {
						"component": "google-cloud-secretmanager",
						"release-type": "python",
						"extra-files": [
							"google/cloud/secretmanager/gapic_version.py",
							"google/cloud/secretmanager_v1/gapic_version.py",
							"google/cloud/secretmanager_v1beta1/gapic_version.py",
							{
								"jsonpath": "$.clientLibrary.version",
								"path": "samples/generated_samples/snippet_metadata_google.cloud.secretmanager.v1.json",
								"type": "json"
							},
							{
								"jsonpath": "$.clientLibrary.version",
								"path": "samples/generated_samples/snippet_metadata_google.cloud.secretmanager.v1beta1.json",
								"type": "json"
							}
						]
					}
				}
			}`,
		},
		{
			name:            "preserve existing extra-files for go library",
			language:        config.LanguageGo,
			initialManifest: `{}`,
			initialConfig: `{
				"packages": {
					"secretmanager": {
						"component": "secretmanager",
						"extra-files": ["some/manual/file.txt"]
					}
				}
			}`,
			library: &config.Library{
				Name:    "secretmanager",
				Version: "1.0.0",
				APIs: []*config.API{
					{Path: "google/cloud/secretmanager/v1"},
				},
			},
			wantManifest: `{"secretmanager":"1.0.0"}`,
			wantConfig: `{
				"packages": {
					"secretmanager": {
						"component": "secretmanager",
						"extra-files": ["some/manual/file.txt"]
					}
				}
			}`,
		},

		{
			name:            "replace existing string extra-files with map if same path",
			language:        config.LanguagePython,
			initialManifest: `{"packages/google-cloud-secretmanager":"1.0.0"}`,
			initialConfig: `{
				"packages": {
					"packages/google-cloud-secretmanager": {
						"component": "google-cloud-secretmanager",
						"extra-files": [
							"samples/generated_samples/snippet_metadata_google.cloud.secretmanager.v1.json"
						]
					}
				}
			}`,
			library: &config.Library{
				Name:    "google-cloud-secretmanager",
				Version: "1.0.0",
				APIs: []*config.API{
					{Path: "google/cloud/secretmanager/v1"},
				},
			},
			wantManifest: `{"packages/google-cloud-secretmanager":"1.0.0"}`,
			wantConfig: `{
				"packages": {
					"packages/google-cloud-secretmanager": {
						"component": "google-cloud-secretmanager",
						"extra-files": [
							"google/cloud/secretmanager/gapic_version.py",
							"google/cloud/secretmanager_v1/gapic_version.py",
							{
								"jsonpath": "$.clientLibrary.version",
								"path": "samples/generated_samples/snippet_metadata_google.cloud.secretmanager.v1.json",
								"type": "json"
							}
						]
					}
				}
			}`,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			tmp := t.TempDir()
			manifestFile, configFile := releasePleaseFiles(
				&config.Config{
					Language: test.language,
				},
			)
			manifestPath := filepath.Join(tmp, manifestFile)
			configPath := filepath.Join(tmp, configFile)
			if err := os.WriteFile(manifestPath, []byte(test.initialManifest), 0644); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(configPath, []byte(test.initialConfig), 0644); err != nil {
				t.Fatal(err)
			}
			cfg := &config.Config{
				Language:  test.language,
				Libraries: []*config.Library{test.library},
			}
			if err := syncToReleasePlease(tmp, cfg, test.library.Name); err != nil {
				t.Fatal(err)
			}

			gotManifestBytes, err := os.ReadFile(manifestPath)
			if err != nil {
				t.Fatal(err)
			}
			var gotManifest, wantManifest map[string]string
			if err := json.Unmarshal(gotManifestBytes, &gotManifest); err != nil {
				t.Fatal(err)
			}
			if err := json.Unmarshal([]byte(test.wantManifest), &wantManifest); err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(wantManifest, gotManifest); diff != "" {
				t.Errorf("manifest mismatch (-want +got):\n%s", diff)
			}

			gotConfigBytes, err := os.ReadFile(configPath)
			if err != nil {
				t.Fatal(err)
			}
			var gotConfig, wantConfig map[string]any
			if err := json.Unmarshal(gotConfigBytes, &gotConfig); err != nil {
				t.Fatal(err)
			}
			if err := json.Unmarshal([]byte(test.wantConfig), &wantConfig); err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(wantConfig, gotConfig); diff != "" {
				t.Errorf("config mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestSyncToReleasePlease_Errors(t *testing.T) {
	for _, test := range []struct {
		name          string
		initialConfig string
		library       *config.Library
	}{
		{
			name: "invalid extra-files element type (int)",
			initialConfig: `{
				"packages": {
					"packages/google-cloud-secretmanager": {
						"component": "google-cloud-secretmanager",
						"extra-files": [123]
					}
				}
			}`,
			library: &config.Library{
				Name:    "google-cloud-secretmanager",
				Version: "1.0.0",
				APIs: []*config.API{
					{Path: "google/cloud/secretmanager/v1"},
				},
			},
		},
		{
			name: "invalid extra-files object (missing path)",
			initialConfig: `{
				"packages": {
					"packages/google-cloud-secretmanager": {
						"component": "google-cloud-secretmanager",
						"extra-files": [{"type": "json"}]
					}
				}
			}`,
			library: &config.Library{
				Name:    "google-cloud-secretmanager",
				Version: "1.0.0",
				APIs: []*config.API{
					{Path: "google/cloud/secretmanager/v1"},
				},
			},
		},
		{
			name: "existing package config is not an object",
			initialConfig: `{
				"packages": {
					"packages/google-cloud-secretmanager": "invalid-string-instead-of-object"
				}
			}`,
			library: &config.Library{
				Name:    "google-cloud-secretmanager",
				Version: "1.0.0",
				APIs: []*config.API{
					{Path: "google/cloud/secretmanager/v1"},
				},
			},
		},
		{
			name: "conflicting map extra-files",
			initialConfig: `{
				"packages": {
					"packages/google-cloud-secretmanager": {
						"component": "google-cloud-secretmanager",
						"extra-files": [
							{
								"jsonpath": "$.clientLibrary.version_different",
								"path": "samples/generated_samples/snippet_metadata_google.cloud.secretmanager.v1.json",
								"type": "json"
							}
						]
					}
				}
			}`,
			library: &config.Library{
				Name:    "google-cloud-secretmanager",
				Version: "1.0.0",
				APIs: []*config.API{
					{Path: "google/cloud/secretmanager/v1"},
				},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			tmp := t.TempDir()
			manifestPath := filepath.Join(tmp, ".release-please-bulk-manifest.json")
			configPath := filepath.Join(tmp, "release-please-bulk-config.json")
			if err := os.WriteFile(manifestPath, []byte("{}"), 0644); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(configPath, []byte(test.initialConfig), 0644); err != nil {
				t.Fatal(err)
			}
			cfg := &config.Config{
				Language:  config.LanguagePython,
				Libraries: []*config.Library{test.library},
			}
			err := syncToReleasePlease(tmp, cfg, test.library.Name)
			if err == nil {
				t.Errorf("expected error, got nil")
			}
		})
	}
}
