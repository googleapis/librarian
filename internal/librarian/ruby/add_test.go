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

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
)

func TestSearchVersionedAPI(t *testing.T) {
	cfg := &config.Config{
		Libraries: []*config.Library{
			{
				APIs: []*config.API{
					{Path: "google/cloud/secretmanager/v1"},
				},
			},
			{
				APIs: []*config.API{
					{Path: "google/cloud/dialogflow/cx/v3beta1"},
				},
			},
			{
				APIs: []*config.API{
					{Path: "google/cloud/unrelated/v1"},
					{Path: "google/cloud/asset/v1"},
				},
			},
		},
	}
	for _, test := range []struct {
		name    string
		apiPath string
		want    string
	}{
		{
			name:    "v1 api found",
			apiPath: "google/cloud/secretmanager",
			want:    "google/cloud/secretmanager/v1",
		},
		{
			name:    "beta api found",
			apiPath: "google/cloud/dialogflow/cx",
			want:    "google/cloud/dialogflow/cx/v3beta1",
		},
		{
			name:    "second api in library matched",
			apiPath: "google/cloud/asset",
			want:    "google/cloud/asset/v1",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := searchVersionedAPI(cfg, test.apiPath)
			if err != nil {
				t.Fatalf("searchVersionedAPI() error = %v", err)
			}
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
