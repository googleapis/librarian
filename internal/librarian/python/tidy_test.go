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

// TestTidy is a single general test which covers various aspects.
// TestTidyAPI should be used for more fine-grained testing.
func TestTidy(t *testing.T) {
	lib := &config.Library{
		Name: "test-library",
		APIs: []*config.API{
			{Path: "google/api"},
			{Path: "google/cloud/customized/v1"},
			{Path: "google/cloud/semiderived/v1"},
			{Path: "google/cloud/fullyderived/v1"},
		},
		Python: &config.PythonPackage{
			ProtoOnlyAPIs: []string{"google/api"},
			OptArgsByAPI: map[string][]string{
				"google/cloud/customized/v1": []string{
					"warehouse-package-name=x",
					"python-gapic-namespace=y",
					"python-gapic-name=z",
					"other=123",
				},
				"google/cloud/semiderived/v1": []string{
					"warehouse-package-name=test-library",
					"python-gapic-namespace=google.cloud",
					"python-gapic-name=semiderived",
					"other=456",
				},
				"google/cloud/fullyderived/v1": []string{
					"warehouse-package-name=test-library",
					"python-gapic-namespace=google.cloud",
					"python-gapic-name=fullyderived",
				},
			},
		},
	}
	want := &config.Library{
		Name: "test-library",
		APIs: []*config.API{
			{Path: "google/api"},
			{Path: "google/cloud/customized/v1"},
			{Path: "google/cloud/semiderived/v1"},
			{Path: "google/cloud/fullyderived/v1"},
		},
		Python: &config.PythonPackage{
			ProtoOnlyAPIs: []string{"google/api"},
			OptArgsByAPI: map[string][]string{
				"google/cloud/customized/v1": []string{
					"warehouse-package-name=x",
					"python-gapic-namespace=y",
					"python-gapic-name=z",
					"other=123",
				},
				"google/cloud/semiderived/v1": []string{
					"other=456",
				},
			},
		},
	}
	got := Tidy(lib)
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

// TestTidyAPI performs testing for a library with a single API, focused
// on per-option testing within an API.
func TestTidyAPI(t *testing.T) {
	for _, test := range []struct {
		name string
		lib  *config.Library
		want *config.Library
	}{
		{
			name: "all library options derived",
			lib: &config.Library{
				Name: "test-library",
				APIs: []*config.API{
					{Path: "google/cloud/derived/v1"},
				},
				Python: &config.PythonPackage{
					DefaultVersion: "v1",
					OptArgsByAPI: map[string][]string{
						"google/cloud/derived/v1": []string{
							"warehouse-package-name=test-library",
							"python-gapic-namespace=google.cloud",
							"python-gapic-name=derived",
						},
					},
				},
			},
			want: &config.Library{
				Name: "test-library",
				APIs: []*config.API{
					{Path: "google/cloud/derived/v1"},
				},
				Python: &config.PythonPackage{
					DefaultVersion: "v1",
				},
			},
		},
		{
			name: "no options specified initially",
			lib: &config.Library{
				Name: "test-library",
				APIs: []*config.API{
					{Path: "google/cloud/derived/v1"},
				},
				Python: &config.PythonPackage{
					DefaultVersion: "v1",
				},
			},
			want: &config.Library{
				Name: "test-library",
				APIs: []*config.API{
					{Path: "google/cloud/derived/v1"},
				},
				Python: &config.PythonPackage{
					DefaultVersion: "v1",
				},
			},
		},
		{
			name: "only one option can be derived",
			lib: &config.Library{
				Name: "test-library",
				APIs: []*config.API{
					{Path: "google/cloud/derived/v1"},
				},
				Python: &config.PythonPackage{
					DefaultVersion: "v1",
					OptArgsByAPI: map[string][]string{
						"google/cloud/derived/v1": []string{
							"warehouse-package-name=other-package-name",
							"python-gapic-namespace=google.cloud",
							"python-gapic-name=other-gapic-name",
						},
					},
				},
			},
			want: &config.Library{
				Name: "test-library",
				APIs: []*config.API{
					{Path: "google/cloud/derived/v1"},
				},
				Python: &config.PythonPackage{
					DefaultVersion: "v1",
					OptArgsByAPI: map[string][]string{
						"google/cloud/derived/v1": []string{
							"warehouse-package-name=other-package-name",
							"python-gapic-name=other-gapic-name",
						},
					},
				},
			},
		},
		{
			// This currently shouldn't happen as we should always have a
			// default version, but let's guard against a future where
			// everything is optional.
			name: "no python config",
			lib: &config.Library{
				Name: "test-library",
				APIs: []*config.API{
					{Path: "google/cloud/derived/v1"},
				},
			},
			want: &config.Library{
				Name: "test-library",
				APIs: []*config.API{
					{Path: "google/cloud/derived/v1"},
				},
			},
		},
		{
			// This currently shouldn't happen as we should always have a
			// default version, but let's guard against a future where
			// everything is optional.
			name: "proto-only API",

			lib: &config.Library{
				Name: "test-library",
				APIs: []*config.API{
					{Path: "google/cloud/common"},
				},
				Python: &config.PythonPackage{
					ProtoOnlyAPIs: []string{"google/cloud/common"},
				},
			},
			want: &config.Library{
				Name: "test-library",
				APIs: []*config.API{
					{Path: "google/cloud/common"},
				},
				Python: &config.PythonPackage{
					ProtoOnlyAPIs: []string{"google/cloud/common"},
				},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := tidyAPI(test.lib, test.lib.APIs[0])
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestDeleteMatchingOption(t *testing.T) {
	for _, test := range []struct {
		name    string
		options []string
		want    []string
	}{
		{
			name:    "option not found",
			options: []string{"a=b", "c=d"},
			want:    []string{"a=b", "c=d"},
		},
		{
			name:    "option found with different value",
			options: []string{"a=b", "test=other", "c=d"},
			want:    []string{"a=b", "test=other", "c=d"},
		},
		{
			name:    "option found with derived value",
			options: []string{"a=b", "test=derived", "c=d"},
			want:    []string{"a=b", "c=d"},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := deleteMatchingOption(test.options, "test", "derived")
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
