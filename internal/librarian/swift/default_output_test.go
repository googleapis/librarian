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

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestDefaultOutput(t *testing.T) {
	for _, test := range []struct {
		name   string
		api    string
		defOut string
		want   string
	}{
		{
			name:   "simple",
			api:    "secretmanager",
			defOut: "packages",
			want:   "packages/swift-secretmanager",
		},
		{
			name:   "empty default",
			api:    "secretmanager",
			defOut: "",
			want:   "swift-secretmanager",
		},
		{
			name:   "nested default",
			api:    "secretmanager",
			defOut: "a/b/c",
			want:   "a/b/c/swift-secretmanager",
		},
		{
			name:   "api path",
			api:    "google/cloud/secretmanager/v1",
			defOut: "generated",
			want:   "generated/swift-google-cloud-secretmanager-v1",
		},
		{
			name:   "api path with trailing slash",
			api:    "google/cloud/secretmanager/v1/",
			defOut: "generated",
			want:   "generated/swift-google-cloud-secretmanager-v1",
		},
		{
			name:   "api path with leading slash",
			api:    "/google/cloud/secretmanager/v1",
			defOut: "generated",
			want:   "generated/swift-google-cloud-secretmanager-v1",
		},
		{
			name:   "api path with both leading and trailing slash (weird)",
			api:    "/google/cloud/secretmanager/v1/",
			defOut: "generated",
			want:   "generated/swift-google-cloud-secretmanager-v1",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := DefaultOutput(test.api, test.defOut)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
