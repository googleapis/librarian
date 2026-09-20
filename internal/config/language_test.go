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

package config

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/yaml"
)

func TestPythonDefault_Unmarshal(t *testing.T) {
	for _, test := range []struct {
		name string
		yaml string
		want *PythonDefault
	}{
		{
			name: "empty YAML",
			yaml: "{}\n",
			want: &PythonDefault{},
		},
		{
			name: "without generator",
			yaml: "library_type: GAPIC_AUTO\n",
			want: &PythonDefault{
				LibraryType: "GAPIC_AUTO",
			},
		},
		{
			name: "with sidekick generator",
			yaml: "generator: sidekick\n",
			want: &PythonDefault{
				Generator: "sidekick",
			},
		},
		{
			name: "with legacy generator",
			yaml: "generator: legacy\n",
			want: &PythonDefault{
				Generator: "legacy",
			},
		},
		{
			name: "with generator and other fields",
			yaml: `generator: sidekick
library_type: GAPIC_AUTO
`,
			want: &PythonDefault{
				Generator:   "sidekick",
				LibraryType: "GAPIC_AUTO",
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := yaml.Unmarshal[PythonDefault]([]byte(test.yaml))
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestPythonDefault_Marshal(t *testing.T) {
	for _, test := range []struct {
		name     string
		input    *PythonDefault
		wantYAML string
	}{
		{
			name:     "empty generator omitted",
			input:    &PythonDefault{},
			wantYAML: "{}\n",
		},
		{
			name: "empty generator with other field omitted",
			input: &PythonDefault{
				LibraryType: "GAPIC_AUTO",
			},
			wantYAML: "library_type: GAPIC_AUTO\n",
		},
		{
			name: "populated generator sidekick",
			input: &PythonDefault{
				Generator: "sidekick",
			},
			wantYAML: "generator: sidekick\n",
		},
		{
			name: "populated generator legacy",
			input: &PythonDefault{
				Generator: "legacy",
			},
			wantYAML: "generator: legacy\n",
		},
		{
			name: "populated generator with other fields",
			input: &PythonDefault{
				Generator:   "sidekick",
				LibraryType: "GAPIC_AUTO",
			},
			wantYAML: `generator: sidekick
library_type: GAPIC_AUTO
`,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			gotBytes, err := yaml.Marshal(test.input)
			if err != nil {
				t.Fatal(err)
			}
			got := string(gotBytes)
			if diff := cmp.Diff(test.wantYAML, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestPythonDefault_RoundTrip(t *testing.T) {
	for _, test := range []struct {
		name  string
		input *PythonDefault
	}{
		{
			name:  "empty",
			input: &PythonDefault{},
		},
		{
			name: "with sidekick generator",
			input: &PythonDefault{
				Generator: "sidekick",
			},
		},
		{
			name: "with legacy generator and options",
			input: &PythonDefault{
				Generator:         "legacy",
				AllowedNamespaces: []string{"google.cloud"},
				LibraryType:       "GAPIC_AUTO",
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			data, err := yaml.Marshal(test.input)
			if err != nil {
				t.Fatal(err)
			}
			got, err := yaml.Unmarshal[PythonDefault](data)
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(test.input, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestPythonPackage_Unmarshal(t *testing.T) {
	for _, test := range []struct {
		name string
		yaml string
		want *PythonPackage
	}{
		{
			name: "empty YAML",
			yaml: "{}\n",
			want: &PythonPackage{},
		},
		{
			name: "without generator",
			yaml: "default_version: v1\n",
			want: &PythonPackage{
				DefaultVersion: "v1",
			},
		},
		{
			name: "with sidekick generator",
			yaml: "generator: sidekick\n",
			want: &PythonPackage{
				PythonDefault: PythonDefault{
					Generator: "sidekick",
				},
			},
		},
		{
			name: "with legacy generator",
			yaml: "generator: legacy\n",
			want: &PythonPackage{
				PythonDefault: PythonDefault{
					Generator: "legacy",
				},
			},
		},
		{
			name: "with generator and package fields",
			yaml: `client_documentation_override: https://cloud.google.com/python
generator: sidekick
`,
			want: &PythonPackage{
				PythonDefault: PythonDefault{
					Generator: "sidekick",
				},
				ClientDocumentationOverride: "https://cloud.google.com/python",
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := yaml.Unmarshal[PythonPackage]([]byte(test.yaml))
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestPythonPackage_Marshal(t *testing.T) {
	for _, test := range []struct {
		name     string
		input    *PythonPackage
		wantYAML string
	}{
		{
			name:     "empty generator omitted",
			input:    &PythonPackage{},
			wantYAML: "{}\n",
		},
		{
			name: "empty generator with other field omitted",
			input: &PythonPackage{
				DefaultVersion: "v1",
			},
			wantYAML: "default_version: v1\n",
		},
		{
			name: "populated generator sidekick",
			input: &PythonPackage{
				PythonDefault: PythonDefault{
					Generator: "sidekick",
				},
			},
			wantYAML: "generator: sidekick\n",
		},
		{
			name: "populated generator legacy",
			input: &PythonPackage{
				PythonDefault: PythonDefault{
					Generator: "legacy",
				},
			},
			wantYAML: "generator: legacy\n",
		},
		{
			name: "populated generator with package fields",
			input: &PythonPackage{
				PythonDefault: PythonDefault{
					Generator: "sidekick",
				},
				ClientDocumentationOverride: "https://cloud.google.com/python",
			},
			wantYAML: `generator: sidekick
client_documentation_override: https://cloud.google.com/python
`,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			gotBytes, err := yaml.Marshal(test.input)
			if err != nil {
				t.Fatal(err)
			}
			got := string(gotBytes)
			if diff := cmp.Diff(test.wantYAML, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestPythonPackage_RoundTrip(t *testing.T) {
	for _, test := range []struct {
		name  string
		input *PythonPackage
	}{
		{
			name:  "empty",
			input: &PythonPackage{},
		},
		{
			name: "with sidekick generator",
			input: &PythonPackage{
				PythonDefault: PythonDefault{
					Generator: "sidekick",
				},
			},
		},
		{
			name: "with legacy generator and package fields",
			input: &PythonPackage{
				PythonDefault: PythonDefault{
					Generator: "legacy",
				},
				DefaultVersion: "v1",
				OptArgsByAPI: map[string][]string{
					"google/cloud/speech/v1": {"python-gapic-name=speech"},
				},
				ProtoOnlyAPIs: []string{"google/cloud/speech/v1/proto"},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			data, err := yaml.Marshal(test.input)
			if err != nil {
				t.Fatal(err)
			}
			got, err := yaml.Unmarshal[PythonPackage](data)
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(test.input, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
