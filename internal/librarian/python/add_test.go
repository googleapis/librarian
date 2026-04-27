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
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
)

func TestAdd(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name    string
		lib     *config.Library
		wantLib *config.Library
	}{
		{
			name: "non-versioned API",
			lib: &config.Library{
				Name: "google-cloud-foo",
				APIs: []*config.API{
					{Path: "google/cloud/foo/type"},
				},
			},
			wantLib: &config.Library{
				Name: "google-cloud-foo",
				APIs: []*config.API{
					{Path: "google/cloud/foo/type"},
				},
				Version: defaultVersion,
			},
		},
		{
			name: "versioned API",
			lib: &config.Library{
				Name: "google-cloud-foo",
				APIs: []*config.API{
					{Path: "google/cloud/foo/v1beta"},
				},
			},
			wantLib: &config.Library{
				Name: "google-cloud-foo",
				APIs: []*config.API{
					{Path: "google/cloud/foo/v1beta"},
				},
				Version: defaultVersion,
				Python: &config.PythonPackage{
					DefaultVersion: "v1beta",
				},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			gotLib, err := Add(test.lib)
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(test.wantLib, gotLib); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestAdd_Error(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name    string
		lib     *config.Library
		wantErr error
	}{
		{
			name: "no APIs",
			lib: &config.Library{
				Name: "no-apis",
			},
			wantErr: errNewLibraryMustHaveOneAPI,
		},
		{
			name: "multiple APIs",
			lib: &config.Library{
				Name: "multiple-apis",
				APIs: []*config.API{
					{Path: "google/cloud/api/v1"},
					{Path: "google/cloud/api/v2"},
				},
			},
			wantErr: errNewLibraryMustHaveOneAPI,
		},
		{
			name: "API in unapproved namespace",
			lib: &config.Library{
				Name: "google-unapproved-bad",
				APIs: []*config.API{
					{Path: "google/unapproved/bad/v1"},
				},
			},
			wantErr: errNewLibraryBadNamespace,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, gotErr := Add(test.lib)
			if !errors.Is(gotErr, test.wantErr) {
				t.Errorf("error = %v, wantErr %v", gotErr, test.wantErr)
			}
		})
	}
}

func TestValidateNewAPIs(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name    string
		lib     *config.Library
		wantErr error
	}{
		{
			name: "valid",
			lib: &config.Library{
				Name: "google-cloud-test",
				APIs: []*config.API{{Path: "google/cloud/test/v1"}},
				Python: &config.PythonPackage{
					DefaultVersion: "v1",
				},
			},
		},
		{
			name: "no python configuration at all",
			lib: &config.Library{
				Name: "google-cloud-test",
				APIs: []*config.API{{Path: "google/cloud/test/v1"}},
			},
			wantErr: errExistingLibraryNoDefaultVersion,
		},
		{
			name: "no default version",
			lib: &config.Library{
				Name:   "google-cloud-test",
				APIs:   []*config.API{{Path: "google/cloud/test/v1"}},
				Python: &config.PythonPackage{},
			},
			wantErr: errExistingLibraryNoDefaultVersion,
		},
		{
			name: "custom GAPIC options",
			lib: &config.Library{
				Name: "google-cloud-test",
				APIs: []*config.API{{Path: "google/cloud/test/v1"}},
				Python: &config.PythonPackage{
					DefaultVersion: "v1",
					OptArgsByAPI: map[string][]string{
						"google/cloud/test/v1": []string{"x=y"},
					},
				},
			},
			wantErr: errExistingLibraryCustomGAPICOptions,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			gotErr := ValidateNewAPIs(test.lib)
			if !errors.Is(gotErr, test.wantErr) {
				t.Errorf("error = %v, wantErr %v", gotErr, test.wantErr)
			}
		})
	}
}
