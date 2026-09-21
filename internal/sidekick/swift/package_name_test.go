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

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/sidekick/api"
)

func TestPackageName(t *testing.T) {
	for _, test := range []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "cloud storage v2",
			input: "google.cloud.storage.v2",
			want:  "google-cloud-storage-v2",
		},
		{
			name:  "iam v1",
			input: "google.iam.v1",
			want:  "google-iam-v1",
		},
		{
			name:  "cloud location",
			input: "google.cloud.location",
			want:  "google-cloud-location",
		},
		{
			name:  "api",
			input: "google.api",
			want:  "google-api",
		},
		{
			name:  "grafeas v1",
			input: "grafeas.v1",
			want:  "grafeas-v1",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			model := api.NewTestAPI(nil, nil, nil)
			model.PackageName = test.input
			got := PackageName(model)
			if got != test.want {
				t.Errorf("mismatch got = %q, want %q", got, test.want)
			}
		})
	}
}

func TestPackageRepoName(t *testing.T) {
	for _, test := range []struct {
		name        string
		outdir      string
		library     *config.Library
		packageName string
		want        string
	}{
		{
			name:        "from outdir with swift prefix",
			outdir:      "generated/swift-google-cloud-secretmanager-v1",
			library:     &config.Library{Name: "google-cloud-secretmanager-v1"},
			packageName: "google-cloud-secretmanager-v1",
			want:        "swift-google-cloud-secretmanager-v1",
		},
		{
			name:        "from packages outdir with swift prefix",
			outdir:      "packages/swift-google-gax",
			library:     &config.Library{Name: "swift-google-gax"},
			packageName: "swift-google-gax",
			want:        "swift-google-gax",
		},
		{
			name:        "from library name without prefix",
			outdir:      "/tmp/random-dir",
			library:     &config.Library{Name: "google-cloud-secretmanager-v1"},
			packageName: "google-cloud-secretmanager-v1",
			want:        "swift-google-cloud-secretmanager-v1",
		},
		{
			name:        "from library name already with prefix",
			outdir:      "/tmp/random-dir",
			library:     &config.Library{Name: "swift-google-cloud-secretmanager-v1"},
			packageName: "google-cloud-secretmanager-v1",
			want:        "swift-google-cloud-secretmanager-v1",
		},
		{
			name:        "from package name without prefix",
			outdir:      "/tmp/random-dir",
			library:     nil,
			packageName: "google-type",
			want:        "swift-google-type",
		},
		{
			name:        "from package name already with prefix",
			outdir:      "/tmp/random-dir",
			library:     nil,
			packageName: "swift-google-type",
			want:        "swift-google-type",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := PackageRepoName(test.outdir, test.library, test.packageName)
			if got != test.want {
				t.Errorf("PackageRepoName() = %q, want %q", got, test.want)
			}
		})
	}
}
