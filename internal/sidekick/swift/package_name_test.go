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

	"github.com/googleapis/librarian/internal/sidekick/api"
)

func TestPackageName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		override string
		want     string
	}{
		{
			name:  "cloud storage v2",
			input: "google.cloud.storage.v2",
			want:  "GoogleCloudStorageV2",
		},
		{
			name:  "iam v1",
			input: "google.iam.v1",
			want:  "GoogleIamV1",
		},
		{
			name:  "cloud location",
			input: "google.cloud.location",
			want:  "GoogleCloudLocation",
		},
		{
			name:  "api",
			input: "google.api",
			want:  "GoogleApi",
		},
		{
			name:  "grafeas v1",
			input: "grafeas.v1",
			want:  "GoogleGrafeasV1",
		},
		{
			name:     "override",
			input:    "google.cloud.storage.v2",
			override: "OverrideName",
			want:     "OverrideName",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := PackageName(&api.API{PackageName: tt.input}, tt.override)
			if got != tt.want {
				t.Errorf("PackageName(%q, %q) = %q, want %q", tt.input, tt.override, got, tt.want)
			}
		})
	}
}
