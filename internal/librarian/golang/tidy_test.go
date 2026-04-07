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

package golang

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
)

func TestTidy(t *testing.T) {
	for _, test := range []struct {
		name          string
		library       *config.Library
		defaultOutput string
		want          *config.Library
	}{
		{
			name: "library output is removed",
			library: &config.Library{
				Name:   "secretmanager",
				Output: "secretmanager",
			},
			want: &config.Library{
				Name: "secretmanager",
			},
		},
		{
			name: "library output unchanged because it doesn't match the library name",
			library: &config.Library{
				Name:   "secretmanager",
				Output: "app/repo/secret-manager",
			},
			want: &config.Library{
				Name:   "secretmanager",
				Output: "app/repo/secret-manager",
			},
		},
		{
			name: "absolute library output is tidied",
			library: &config.Library{
				Name:   "secretmanager",
				Output: "/workspace/secretmanager",
			},
			defaultOutput: "/workspace",
			want: &config.Library{
				Name: "secretmanager",
			},
		},
		{
			name: "absolute library output at root is tidied",
			library: &config.Library{
				Name:   "secretmanager",
				Output: "/secretmanager",
			},
			defaultOutput: "/",
			want: &config.Library{
				Name: "secretmanager",
			},
		},
		{
			name: "absolute library output with trailing slash is tidied",
			library: &config.Library{
				Name:   "secretmanager",
				Output: "/workspace/secretmanager/",
			},
			defaultOutput: "/workspace",
			want: &config.Library{
				Name: "secretmanager",
			},
		},
		{
			name: "library path is kept if it is totally different",
			library: &config.Library{
				Name:   "secretmanager",
				Output: "/workspace/aRandomName",
			},
			defaultOutput: "/workspace",
			want: &config.Library{
				Name:   "secretmanager",
				Output: "/workspace/aRandomName",
			},
		},
		{
			name: "absolute library output kept when default is relative path matching suffix",
			library: &config.Library{
				Name:   "secretmanager",
				Output: "/workspace/secretmanager",
			},
			defaultOutput: "workspace",
			want: &config.Library{
				Name:   "secretmanager",
				Output: "/workspace/secretmanager",
			},
		},
		{
			name: "root module does not change",
			library: &config.Library{
				Name:   "root-module",
				Output: "/home/repo",
				Keep:   []string{"README.md", "internal/version.go"},
			},
			want: &config.Library{
				Name:   "root-module",
				Output: "/home/repo",
				Keep:   []string{"README.md", "internal/version.go"},
			},
		},
		{
			name: "nested module suffix is removed",
			library: &config.Library{
				Name:   "bigquery/v2",
				Output: "bigquery/v2",
			},
			want: &config.Library{
				Name: "bigquery/v2",
			},
		},
		{
			name: "Go config is nil",
			library: &config.Library{
				Name: "test-lib",
			},
			want: &config.Library{
				Name: "test-lib",
			},
		},
		{
			name: "empty Go config is removed",
			library: &config.Library{
				Name: "test-lib",
				Go:   &config.GoModule{},
			},
			want: &config.Library{
				Name: "test-lib",
			},
		},
		{
			name: "GoAPIs default ImportPath and ClientPackage are cleared",
			library: &config.Library{
				Name: "test-lib",
				Go: &config.GoModule{
					GoAPIs: []*config.GoAPI{
						{
							Path:          "google/cloud/speech/v1",
							ImportPath:    "speech/apiv1",
							ClientPackage: "speech",
							ProtoPackage:  "google.cloud.speech.v1", // Prevents goAPI from being empty
						},
					},
				},
			},
			want: &config.Library{
				Name: "test-lib",
				Go: &config.GoModule{
					GoAPIs: []*config.GoAPI{
						{
							Path:         "google/cloud/speech/v1",
							ProtoPackage: "google.cloud.speech.v1",
						},
					},
				},
			},
		},
		{
			name: "empty go module is cleared",
			library: &config.Library{
				Name: "test-lib",
				Go: &config.GoModule{
					GoAPIs: []*config.GoAPI{
						{
							Path:          "google/cloud/speech/v1",
							ImportPath:    "speech/apiv1",
							ClientPackage: "speech",
						},
					},
				},
			},
			want: &config.Library{
				Name: "test-lib",
			},
		},
		{
			name: "GoAPIs empty config is removed",
			library: &config.Library{
				Name: "test-lib",
				Go: &config.GoModule{
					GoAPIs: []*config.GoAPI{
						{
							Path: "google/cloud/speech/v1",
						},
						{
							Path:         "google/cloud/vision/v1",
							ProtoPackage: "google.cloud.vision.v1",
						},
					},
				},
			},
			want: &config.Library{
				Name: "test-lib",
				Go: &config.GoModule{
					GoAPIs: []*config.GoAPI{
						{
							Path:         "google/cloud/vision/v1",
							ProtoPackage: "google.cloud.vision.v1",
						},
					},
				},
			},
		},
		{
			name: "Non-default ImportPath and ClientPackage are retained",
			library: &config.Library{
				Name: "test-lib",
				Go: &config.GoModule{
					GoAPIs: []*config.GoAPI{
						{
							Path:          "google/cloud/speech/v1",
							ImportPath:    "custom/path/apiv1",
							ClientPackage: "customspeech",
							ProtoPackage:  "google.cloud.speech.v1",
						},
					},
				},
			},
			want: &config.Library{
				Name: "test-lib",
				Go: &config.GoModule{
					GoAPIs: []*config.GoAPI{
						{
							Path:          "google/cloud/speech/v1",
							ImportPath:    "custom/path/apiv1",
							ClientPackage: "customspeech",
							ProtoPackage:  "google.cloud.speech.v1",
						},
					},
				},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := Tidy(test.library, test.defaultOutput)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestIsEmptyAPI(t *testing.T) {
	for _, test := range []struct {
		name  string
		goAPI *config.GoAPI
		want  bool
	}{
		{
			name:  "empty",
			goAPI: &config.GoAPI{},
			want:  true,
		},
		{
			name: "not empty with client package",
			goAPI: &config.GoAPI{
				ClientPackage: "foo",
			},
		},
		{
			name: "not empty with DIRE GAPIC",
			goAPI: &config.GoAPI{
				DIREGAPIC: true,
			},
		},
		{
			name: "not empty with EnabledGeneratorFeatures",
			goAPI: &config.GoAPI{
				EnabledGeneratorFeatures: []string{"feature"},
			},
		},
		{
			name: "not empty with ImportPath",
			goAPI: &config.GoAPI{
				ImportPath: "foo",
			},
		},
		{
			name: "not empty with NestedProtos",
			goAPI: &config.GoAPI{
				NestedProtos: []string{"foo"},
			},
		},
		{
			name: "not empty with NoMetadata",
			goAPI: &config.GoAPI{
				NoMetadata: true,
			},
		},
		{
			name: "not empty with NoSnippets",
			goAPI: &config.GoAPI{
				NoSnippets: true,
			},
		},
		{
			name: "not empty with ProtoOnly",
			goAPI: &config.GoAPI{
				ProtoOnly: true,
			},
		},
		{
			name: "not empty with ProtoPackage",
			goAPI: &config.GoAPI{
				ProtoPackage: "foo",
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := isEmptyAPI(test.goAPI)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestIsEmptyGoModule(t *testing.T) {
	for _, test := range []struct {
		name     string
		goModule *config.GoModule
		want     bool
	}{
		{
			name:     "empty",
			goModule: &config.GoModule{},
			want:     true,
		},
		{
			name: "not empty with DeleteGenerationOutputPaths",
			goModule: &config.GoModule{
				DeleteGenerationOutputPaths: []string{"path"},
			},
		},
		{
			name: "not empty with GoAPIs",
			goModule: &config.GoModule{
				GoAPIs: []*config.GoAPI{{}},
			},
		},
		{
			name: "not empty with ModulePathVersion",
			goModule: &config.GoModule{
				ModulePathVersion: "v2",
			},
		},
		{
			name: "not empty with NestedModule",
			goModule: &config.GoModule{
				NestedModule: "foo",
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := isEmptyGoModule(test.goModule)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
