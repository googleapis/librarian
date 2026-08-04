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
)

func TestAddManifest(t *testing.T) {
	for _, test := range []struct {
		name     string
		manifest map[string]string
		pkgName  string
		want     map[string]string
	}{
		{
			name:     "add new package to empty manifest",
			manifest: map[string]string{},
			pkgName:  "google-cloud-secret_manager-v1",
			want: map[string]string{
				"google-cloud-secret_manager-v1":        "0.0.1",
				"google-cloud-secret_manager-v1+FILLER": "0.0.0",
			},
		},
		{
			name: "add new package preserving existing entries",
			manifest: map[string]string{
				"google-cloud-asset": "1.0.0",
			},
			pkgName: "google-cloud-secret_manager",
			want: map[string]string{
				"google-cloud-asset":                 "1.0.0",
				"google-cloud-secret_manager":        "0.0.1",
				"google-cloud-secret_manager+FILLER": "0.0.0",
			},
		},
		{
			name: "existing package unchanged",
			manifest: map[string]string{
				"google-cloud-secret_manager":        "2.1.0",
				"google-cloud-secret_manager+FILLER": "0.0.0",
			},
			pkgName: "google-cloud-secret_manager",
			want: map[string]string{
				"google-cloud-secret_manager":        "2.1.0",
				"google-cloud-secret_manager+FILLER": "0.0.0",
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := AddManifest(test.manifest, test.pkgName)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
