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

package nodejs

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
)

func TestFindExistingLibraryForNewAPI(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name      string
		libraries []*config.Library
		apiPath   string
		// The name of the library that should be returned, or empty if nill
		// should be returned.
		wantName string
	}{
		{
			name:      "no libraries",
			libraries: []*config.Library{},
			apiPath:   "google/cloud/test/v2",
		},
		{
			name: "exact match",
			libraries: []*config.Library{
				{
					Name: "google-cloud-other",
					APIs: []*config.API{{Path: "google/cloud/other"}},
				},
				{
					Name: "google-cloud-test",
					APIs: []*config.API{{Path: "google/cloud/test/v1"}},
				},
			},
			apiPath:  "google/cloud/test/v2",
			wantName: "google-cloud-test",
		},
		{
			name: "new API is prefix of existing after stripping versions",
			libraries: []*config.Library{
				{
					Name: "google-cloud-other",
					APIs: []*config.API{{Path: "google/cloud/other"}},
				},
				{
					Name: "google-cloud-test",
					APIs: []*config.API{{Path: "google/cloud/test/admin/v1"}},
				},
			},
			apiPath: "google/cloud/test/v2",
		},
		{
			name: "existing API is prefix of new one after stripping versions",
			libraries: []*config.Library{
				{
					Name: "google-cloud-other",
					APIs: []*config.API{{Path: "google/cloud/other"}},
				},
				{
					Name: "google-cloud-test",
					APIs: []*config.API{{Path: "google/cloud/test/v1"}},
				},
			},
			apiPath: "google/cloud/test/admin/v2",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := FindExistingLibraryForNewAPI(test.libraries, test.apiPath)
			gotName := ""
			if got != nil {
				gotName = got.Name
			}
			if diff := cmp.Diff(gotName, test.wantName); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
