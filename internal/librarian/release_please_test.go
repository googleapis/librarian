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

func TestHasReleasePleaseConfigs(t *testing.T) {
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
			files := releasePleaseFiles(&config.Config{Language: test.language})
			if test.createConfig {
				if err := os.WriteFile(filepath.Join(tmp, files.bulk.configFile), []byte("{}"), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			if test.createManifest {
				if err := os.WriteFile(filepath.Join(tmp, files.bulk.manifestFile), []byte("{}"), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			got := hasReleasePleaseConfigs(tmp, &config.Config{Language: test.language})
			if got != test.want {
				t.Errorf("hasReleasePleaseConfigs(%s, %s) = %t, want %t", tmp, test.language, got, test.want)
			}
		})
	}
}

func TestHasReleasePleaseConfigs_Python(t *testing.T) {
	for _, test := range []struct {
		name  string
		files []string
		want  bool
	}{
		{
			name:  "both individual exist",
			files: []string{individualConfigFile, individualManifestFile},
			want:  true,
		},
		{
			name:  "individual config missing",
			files: []string{individualManifestFile},
			want:  false,
		},
		{
			name:  "individual manifest missing",
			files: []string{individualConfigFile},
			want:  false,
		},
		{
			name:  "both bulk exist",
			files: []string{bulkConfigFile, bulkManifestFile},
			want:  true,
		},
		{
			name:  "bulk config missing",
			files: []string{bulkManifestFile},
			want:  false,
		},
		{
			name:  "bulk manifest missing",
			files: []string{bulkConfigFile},
			want:  false,
		},
		{
			name:  "all exist",
			files: []string{bulkConfigFile, bulkManifestFile, individualConfigFile, individualManifestFile},
			want:  true,
		},
		{
			name:  "all missing",
			files: nil,
			want:  false,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			tmp := t.TempDir()
			for _, file := range test.files {
				if err := os.WriteFile(filepath.Join(tmp, file), []byte("{}"), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			got := hasReleasePleaseConfigs(tmp, &config.Config{Language: config.LanguagePython})
			if got != test.want {
				t.Errorf("hasReleasePleaseConfigs(%s, %s) = %t, want %t", tmp, config.LanguagePython, got, test.want)
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
			wantConfig: `{
				"packages": {
					"secretmanager": {
						"component": "secretmanager",
						"extra-files": [
							{
								"jsonpath": "$.clientLibrary.version",
								"path": "examples/apiv1/snippet_metadata.google.cloud.secretmanager.v1.json",
								"type": "json"
							}
						]
					}
				}
			}`,
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
						"extra-files": [
							"some/manual/file.txt",
							{
								"jsonpath": "$.clientLibrary.version",
								"path": "examples/apiv1/snippet_metadata.google.cloud.secretmanager.v1.json",
								"type": "json"
							}
						]
					}
				}
			}`,
		},
		{
			name:            "new ruby versioned library",
			language:        config.LanguageRuby,
			initialManifest: `{}`,
			initialConfig:   `{"packages": {}}`,
			library: &config.Library{
				Name:    "google-cloud-secret_manager-v1",
				Version: "0.0.1",
				APIs: []*config.API{
					{Path: "google/cloud/secretmanager/v1"},
				},
			},
			wantManifest: `{
				"google-cloud-secret_manager-v1": "0.0.1",
				"google-cloud-secret_manager-v1+FILLER": "0.0.0"
			}`,
			wantConfig: `{
				"packages": {
					"google-cloud-secret_manager-v1": {
						"component": "google-cloud-secret_manager-v1",
						"version_file": "lib/google/cloud/secret_manager/v1/version.rb"
					}
				}
			}`,
		},
		{
			name:     "new ruby main client",
			language: config.LanguageRuby,
			initialManifest: `{
				"google-cloud-secret_manager-v1": "0.0.1",
				"google-cloud-secret_manager-v1+FILLER": "0.0.0"
			}`,
			initialConfig: `{
				"packages": {
					"google-cloud-secret_manager-v1": {
						"component": "google-cloud-secret_manager-v1",
						"version_file": "lib/google/cloud/secret_manager/v1/version.rb"
					}
				}
			}`,
			library: &config.Library{
				Name:    "google-cloud-secret_manager",
				Version: "0.0.1",
				APIs: []*config.API{
					{Path: "google/cloud/secretmanager/v1"},
				},
			},
			wantManifest: `{
				"google-cloud-secret_manager": "0.0.1",
				"google-cloud-secret_manager+FILLER": "0.0.0",
				"google-cloud-secret_manager-v1": "0.0.1",
				"google-cloud-secret_manager-v1+FILLER": "0.0.0"
			}`,
			wantConfig: `{
				"packages": {
					"google-cloud-secret_manager": {
						"component": "google-cloud-secret_manager",
						"version_file": "lib/google/cloud/secret_manager/version.rb"
					},
					"google-cloud-secret_manager-v1": {
						"component": "google-cloud-secret_manager-v1",
						"version_file": "lib/google/cloud/secret_manager/v1/version.rb"
					}
				}
			}`,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			tmp := t.TempDir()
			files := releasePleaseFiles(
				&config.Config{
					Language: test.language,
				},
			)
			manifestPath := filepath.Join(tmp, files.bulk.manifestFile)
			configPath := filepath.Join(tmp, files.bulk.configFile)
			if err := os.WriteFile(manifestPath, []byte(test.initialManifest), 0o644); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(configPath, []byte(test.initialConfig), 0o644); err != nil {
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
			cfg := &config.Config{
				Language:  config.LanguagePython,
				Libraries: []*config.Library{test.library},
			}
			files := releasePleaseFiles(cfg)
			manifestPath := filepath.Join(tmp, files.individual.manifestFile)
			configPath := filepath.Join(tmp, files.individual.configFile)
			if err := os.WriteFile(manifestPath, []byte("{}"), 0o644); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(configPath, []byte(test.initialConfig), 0o644); err != nil {
				t.Fatal(err)
			}
			err := syncToReleasePlease(tmp, cfg, test.library.Name)
			if err == nil {
				t.Errorf("expected error, got nil")
			}
		})
	}
}

func TestSyncToReleasePlease_Python(t *testing.T) {
	for _, test := range []struct {
		name                 string
		library              *config.Library
		files                map[string]string
		wantModifiedConfig   string
		wantUnmodifiedConfig string
		wantManifest         string
		wantConfig           string
	}{
		{
			name: "updates bulk config when tracked in bulk (merge, deduplicate, sort extra-files)",
			library: &config.Library{
				Name:    "google-cloud-biglake-hive",
				Version: "0.3.2",
				APIs: []*config.API{
					{Path: "google/cloud/biglake/hive/v1"},
					{Path: "google/cloud/biglake/hive/v1beta"},
				},
			},
			files: map[string]string{
				bulkManifestFile: `{"packages/google-cloud-biglake-hive": "0.3.2"}`,
				bulkConfigFile: `{
					"packages": {
						"packages/google-cloud-biglake-hive": {
							"component": "google-cloud-biglake-hive",
							"extra-files": [
								"google/cloud/biglake_hive/gapic_version.py",
								"google/cloud/biglake_hive_v1beta/gapic_version.py"
							]
						}
					}
				}`,
				individualManifestFile: `{}`,
				individualConfigFile:   `{"packages": {}}`,
			},
			wantModifiedConfig:   bulkConfigFile,
			wantUnmodifiedConfig: individualConfigFile,
			wantManifest:         `{"packages/google-cloud-biglake-hive": "0.3.2"}`,
			wantConfig: `{
				"packages": {
					"packages/google-cloud-biglake-hive": {
						"component": "google-cloud-biglake-hive",
						"extra-files": [
							"google/cloud/biglake_hive/gapic_version.py",
							"google/cloud/biglake_hive_v1/gapic_version.py",
							"google/cloud/biglake_hive_v1beta/gapic_version.py",
							{
								"jsonpath": "$.clientLibrary.version",
								"path": "samples/generated_samples/snippet_metadata_google.cloud.biglake.hive.v1.json",
								"type": "json"
							},
							{
								"jsonpath": "$.clientLibrary.version",
								"path": "samples/generated_samples/snippet_metadata_google.cloud.biglake.hive.v1beta.json",
								"type": "json"
							}
						]
					}
				}
			}`,
		},
		{
			name: "updates individual config when tracked in individual (merge, deduplicate, sort extra-files)",
			library: &config.Library{
				Name:    "google-cloud-secretmanager",
				Version: "1.0.0",
				APIs: []*config.API{
					{Path: "google/cloud/secretmanager/v1"},
					{Path: "google/cloud/secretmanager/v1beta1"},
				},
			},
			files: map[string]string{
				bulkManifestFile:       `{}`,
				bulkConfigFile:         `{"packages": {}}`,
				individualManifestFile: `{"packages/google-cloud-secretmanager":"1.0.0"}`,
				individualConfigFile: `{
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
			},
			wantModifiedConfig:   individualConfigFile,
			wantUnmodifiedConfig: bulkConfigFile,
			wantManifest:         `{"packages/google-cloud-secretmanager":"1.0.0"}`,
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
			name: "updates individual config when tracked in individual (replaces string extra-files with structured map)",
			library: &config.Library{
				Name:    "google-cloud-secretmanager",
				Version: "1.0.0",
				APIs: []*config.API{
					{Path: "google/cloud/secretmanager/v1"},
				},
			},
			files: map[string]string{
				bulkManifestFile:       `{}`,
				bulkConfigFile:         `{"packages": {}}`,
				individualManifestFile: `{"packages/google-cloud-secretmanager":"1.0.0"}`,
				individualConfigFile: `{
					"packages": {
						"packages/google-cloud-secretmanager": {
							"component": "google-cloud-secretmanager",
							"extra-files": [
								"samples/generated_samples/snippet_metadata_google.cloud.secretmanager.v1.json"
							]
						}
					}
				}`,
			},
			wantModifiedConfig:   individualConfigFile,
			wantUnmodifiedConfig: bulkConfigFile,
			wantManifest:         `{"packages/google-cloud-secretmanager":"1.0.0"}`,
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
			name: "new python library defaults to individual config",
			library: &config.Library{
				Name:    "google-cloud-secretmanager",
				Version: "0.0.0",
				APIs: []*config.API{
					{Path: "google/cloud/secretmanager/v1"},
				},
			},
			files: map[string]string{
				bulkManifestFile:       `{}`,
				bulkConfigFile:         `{"packages": {}}`,
				individualManifestFile: `{}`,
				individualConfigFile:   `{"packages": {}}`,
			},
			wantModifiedConfig:   individualConfigFile,
			wantUnmodifiedConfig: bulkConfigFile,
			wantManifest:         `{"packages/google-cloud-secretmanager":"0.0.0"}`,
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
			writeTestFiles(t, tmp, test.files)
			cfg := &config.Config{
				Language:  config.LanguagePython,
				Libraries: []*config.Library{test.library},
			}
			if err := syncToReleasePlease(tmp, cfg, test.library.Name); err != nil {
				t.Fatal(err)
			}

			manifestFile := individualManifestFile
			if test.wantModifiedConfig == bulkConfigFile {
				manifestFile = bulkManifestFile
			}
			gotManifestBytes, err := os.ReadFile(filepath.Join(tmp, manifestFile))
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

			gotConfigBytes, err := os.ReadFile(filepath.Join(tmp, test.wantModifiedConfig))
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

			gotUnmodified, err := readJSONFile[map[string]any](filepath.Join(tmp, test.wantUnmodifiedConfig))
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(map[string]any{"packages": map[string]any{}}, gotUnmodified); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestIsTrackedInBulkConfig(t *testing.T) {
	for _, test := range []struct {
		name       string
		configJSON string
		writeFile  bool
		pkgPath    string
		want       bool
	}{
		{
			name:       "package exists in bulk config",
			configJSON: `{"packages": {"packages/google-cloud-foo": {}}}`,
			writeFile:  true,
			pkgPath:    "packages/google-cloud-foo",
			want:       true,
		},
		{
			name:       "package does not exist in bulk config",
			configJSON: `{"packages": {"packages/google-cloud-other": {}}}`,
			writeFile:  true,
			pkgPath:    "packages/google-cloud-foo",
			want:       false,
		},
		{
			name:      "bulk config file does not exist",
			writeFile: false,
			pkgPath:   "packages/google-cloud-foo",
			want:      false,
		},
		{
			name:       "invalid packages field",
			configJSON: `{"packages": "not-a-map"}`,
			writeFile:  true,
			pkgPath:    "packages/google-cloud-foo",
			want:       false,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			tmp := t.TempDir()
			if test.writeFile {
				if err := os.WriteFile(filepath.Join(tmp, bulkConfigFile), []byte(test.configJSON), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			got := isTrackedInBulkConfig(tmp, test.pkgPath)
			if got != test.want {
				t.Errorf("isTrackedInBulkConfig(%s, %s) = %t, want %t", tmp, test.pkgPath, got, test.want)
			}
		})
	}
}

func writeTestFiles(t *testing.T, dir string, files map[string]string) {
	t.Helper()
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}
