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

package ruby

import (
	"testing"

	"github.com/googleapis/librarian/internal/config"
)

func TestFindConfiguredLibrary(t *testing.T) {
	libraries := []*config.Library{
		{
			Name: "google-cloud-secret_manager",
			APIs: []*config.API{{Path: "google/cloud/secretmanager/v1"}},
			Ruby: &config.RubyPackage{WrapperOf: []string{"v1:0.0"}},
		},
		{
			Name: "google-cloud-secret_manager-v1",
			APIs: []*config.API{{Path: "google/cloud/secretmanager/v1"}},
		},
		{
			Name: "google-cloud-secret_manager-v1beta2",
			APIs: []*config.API{{Path: "google/cloud/secretmanager/v1beta2"}},
		},
		{
			Name: "google-cloud-config_delivery",
			APIs: []*config.API{{Path: "google/cloud/configdelivery/v1alpha"}},
			Ruby: &config.RubyPackage{WrapperOf: []string{"v1:0.0"}},
		},
		{
			Name: "google-cloud-config_delivery-v1",
			APIs: []*config.API{{Path: "google/cloud/configdelivery/v1"}},
		},
		{
			Name: "google-cloud-workflows",
			APIs: []*config.API{
				{Path: "google/cloud/workflows/v1"},
				{Path: "google/cloud/workflows/executions/v1"},
			},
			Ruby: &config.RubyPackage{WrapperOf: []string{"v1:0.0"}},
		},
		{
			Name: "google-cloud-workflows-v1",
			APIs: []*config.API{{Path: "google/cloud/workflows/v1"}},
		},
	}
	for _, test := range []struct {
		name    string
		apiPath string
		want    string
	}{
		{
			name:    "versioned path returns versioned library, not wrapper",
			apiPath: "google/cloud/secretmanager/v1",
			want:    "google-cloud-secret_manager-v1",
		},
		{
			name:    "versioned path without wrapper",
			apiPath: "google/cloud/secretmanager/v1beta2",
			want:    "google-cloud-secret_manager-v1beta2",
		},
		{
			name:    "unversioned path returns wrapper",
			apiPath: "google/cloud/secretmanager",
			want:    "google-cloud-secret_manager",
		},
		{
			name:    "unversioned path returns wrapper listing another version",
			apiPath: "google/cloud/configdelivery",
			want:    "google-cloud-config_delivery",
		},
		{
			name:    "unversioned path returns multi-API wrapper",
			apiPath: "google/cloud/workflows/executions",
			want:    "google-cloud-workflows",
		},
		{
			name:    "versioned path listed only by wrapper is not found",
			apiPath: "google/cloud/configdelivery/v1alpha",
		},
		{
			name:    "unconfigured version is not found",
			apiPath: "google/cloud/secretmanager/v2",
		},
		{
			name:    "unconfigured service is not found",
			apiPath: "google/cloud/tasks/v2",
		},
		{
			name:    "unconfigured unversioned path is not found",
			apiPath: "google/cloud/tasks",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			var got string
			if library := FindConfiguredLibrary(libraries, test.apiPath); library != nil {
				got = library.Name
			}
			if got != test.want {
				t.Errorf("FindConfiguredLibrary(%q) = %q, want %q", test.apiPath, got, test.want)
			}
		})
	}
}
