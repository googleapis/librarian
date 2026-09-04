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

package python

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
)

func TestGAPICNamespace(t *testing.T) {
	for _, test := range []struct {
		name    string
		apiPath string
		lib     *config.Library
		want    string
	}{
		{
			name:    "no python config",
			apiPath: "google/cloud/secretmanager/v1",
			lib:     &config.Library{},
			want:    "google.cloud",
		},
		{
			name:    "no options for API",
			apiPath: "google/cloud/secretmanager/v1",
			lib:     &config.Library{Python: &config.PythonPackage{}},
			want:    "google.cloud",
		},
		{
			name:    "other options present",
			apiPath: "google/cloud/secretmanager/v1",
			lib: &config.Library{
				Python: &config.PythonPackage{
					OptArgsByAPI: map[string][]string{
						"google/cloud/secretmanager/v1": {"python-gapic-name=secretmanager"},
					},
				},
			},
			want: "google.cloud",
		},
		{
			name:    "explicit namespace option",
			apiPath: "google/cloud/secretmanager/v1",
			lib: &config.Library{
				Python: &config.PythonPackage{
					OptArgsByAPI: map[string][]string{
						"google/cloud/secretmanager/v1": {"python-gapic-namespace=custom.namespace"},
					},
				},
			},
			want: "custom.namespace",
		},
		{
			name:    "fallback single path element",
			apiPath: "grafeas/v1",
			lib:     &config.Library{},
			want:    "grafeas",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := gapicNamespace(test.apiPath, test.lib)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestDeriveGAPICNamespace(t *testing.T) {
	for _, test := range []struct {
		name string
		path string
		want string
	}{
		{
			name: "single path element",
			path: "grafeas",
			want: "grafeas",
		},
		{
			name: "single path element with version",
			path: "grafeas/v1",
			want: "grafeas",
		},
		{
			name: "multiple path elements",
			path: "google/cloud/datacatalog/lineage/v1",
			want: "google.cloud",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := deriveGAPICNamespace(test.path)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestGAPICName(t *testing.T) {
	for _, test := range []struct {
		name    string
		apiPath string
		lib     *config.Library
		want    string
	}{
		{
			name:    "no python config",
			apiPath: "google/cloud/datacatalog/lineage/v1",
			lib:     &config.Library{},
			want:    "datacatalog_lineage",
		},
		{
			name:    "no options for API",
			apiPath: "google/cloud/datacatalog/lineage/v1",
			lib:     &config.Library{Python: &config.PythonPackage{}},
			want:    "datacatalog_lineage",
		},
		{
			name:    "other options present",
			apiPath: "google/cloud/datacatalog/lineage/v1",
			lib: &config.Library{
				Python: &config.PythonPackage{
					OptArgsByAPI: map[string][]string{
						"google/cloud/datacatalog/lineage/v1": {"python-gapic-namespace=custom.namespace"},
					},
				},
			},
			want: "datacatalog_lineage",
		},
		{
			name:    "explicit name option",
			apiPath: "google/cloud/datacatalog/lineage/v1",
			lib: &config.Library{
				Python: &config.PythonPackage{
					OptArgsByAPI: map[string][]string{
						"google/cloud/datacatalog/lineage/v1": {"python-gapic-name=custom_name"},
					},
				},
			},
			want: "custom_name",
		},
		{
			name:    "fallback single path element in name",
			apiPath: "google/cloud/datacatalog/v1",
			lib:     &config.Library{},
			want:    "datacatalog",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := gapicName(test.apiPath, test.lib)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestDeriveGAPICName(t *testing.T) {
	for _, test := range []struct {
		name string
		path string
		want string
	}{
		{
			name: "single path element in name",
			path: "google/cloud/datacatalog/v1",
			want: "datacatalog",
		},
		{
			name: "multiple path elements in name",
			path: "google/cloud/datacatalog/lineage/v1",
			want: "datacatalog_lineage",
		},
		{
			name: "no version",
			path: "google/apps/script/type",
			want: "script_type",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := deriveGAPICName(test.path)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestFindOption(t *testing.T) {
	for _, test := range []struct {
		name      string
		options   []string
		wantValue string
		wantOk    bool
	}{
		{
			name:    "empty options",
			options: []string{},
		},
		{
			name:    "requested option not present",
			options: []string{"a=b"},
		},
		{
			name:    "requested option not present, but similar names are",
			options: []string{"othertest=a", "testother=b"},
		},
		{
			name:      "option present with value",
			options:   []string{"a=b", "test=test-value", "c=d"},
			wantValue: "test-value",
			wantOk:    true,
		},
		{
			name:    "option present without value",
			options: []string{"a=b", "test=", "c=d"},
			wantOk:  true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			gotValue, gotOk := findOption(test.options, "test")
			if diff := cmp.Diff(test.wantValue, gotValue); diff != "" {
				t.Errorf("mismatch in value (-want +got):\n%s", diff)
			}
			if test.wantOk != gotOk {
				t.Errorf("mismatch in found: want %v, got %v", test.wantOk, gotOk)
			}
		})
	}
}
