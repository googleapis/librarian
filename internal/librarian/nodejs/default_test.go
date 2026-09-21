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

func TestFill(t *testing.T) {
	for _, test := range []struct {
		name string
		in   *config.Library
		want *config.Library
	}{
		{
			name: "cloud library derives package name",
			in: &config.Library{
				Name: "google-cloud-secretmanager",
			},
			want: &config.Library{
				Name: "google-cloud-secretmanager",
				Nodejs: &config.NodejsPackage{
					PackageName: "@google-cloud/secretmanager",
				},
			},
		},
		{
			name: "cloud library with existing nodejs config preserves other fields",
			in: &config.Library{
				Name: "google-cloud-monitoring",
				Nodejs: &config.NodejsPackage{
					DefaultVersion: "v1",
				},
			},
			want: &config.Library{
				Name: "google-cloud-monitoring",
				Nodejs: &config.NodejsPackage{
					DefaultVersion: "v1",
					PackageName:    "@google-cloud/monitoring",
				},
			},
		},
		{
			name: "preserves explicit package name",
			in: &config.Library{
				Name: "google-cloud-secretmanager",
				Nodejs: &config.NodejsPackage{
					PackageName: "@custom/override",
				},
			},
			want: &config.Library{
				Name: "google-cloud-secretmanager",
				Nodejs: &config.NodejsPackage{
					PackageName: "@custom/override",
				},
			},
		},
		{
			name: "non-cloud library leaves package name empty",
			in: &config.Library{
				Name: "google-apps-meet",
			},
			want: &config.Library{
				Name: "google-apps-meet",
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := Fill(test.in)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
