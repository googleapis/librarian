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

package swift

import "testing"

func TestDefaultLibraryName(t *testing.T) {
	for _, test := range []struct {
		api  string
		want string
	}{
		{"google/cloud/secretmanager/v1", "GoogleCloudSecretmanagerV1"},
		{"google/maps/addressvalidation/v1", "GoogleMapsAddressvalidationV1"},
		{"google/api/v1", "GoogleApiV1"},
		{"grafeas/v1", "GoogleGrafeasV1"},
	} {
		t.Run(test.api, func(t *testing.T) {
			got := DefaultLibraryName(test.api)
			if got != test.want {
				t.Errorf("DefaultLibraryName(%q) = %q, want %q", test.api, got, test.want)
			}
		})
	}
}
