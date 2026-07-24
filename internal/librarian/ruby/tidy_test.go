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

func TestTidy(t *testing.T) {
	for _, test := range []struct {
		name string
		lib  *config.Library
		want *config.Library
	}{
		{
			name: "sorts and compacts additional_protos",
			lib: &config.Library{
				Name: "google-cloud-asset-v1",
				Ruby: &config.RubyPackage{
					AdditionalProtos: []string{"b.proto", "a.proto", "a.proto"},
				},
				APIs: []*config.API{
					{
						Path: "google/cloud/asset/v1",
						Ruby: &config.RubyAPI{
							AdditionalProtos: []string{"d.proto", "c.proto", "c.proto"},
						},
					},
				},
			},
			want: &config.Library{
				Name: "google-cloud-asset-v1",
				Ruby: &config.RubyPackage{
					AdditionalProtos: []string{"a.proto", "b.proto"},
				},
				APIs: []*config.API{
					{
						Path: "google/cloud/asset/v1",
						Ruby: &config.RubyAPI{
							AdditionalProtos: []string{"c.proto", "d.proto"},
						},
					},
				},
			},
		},
		{
			name: "removes empty ruby structs",
			lib: &config.Library{
				Name: "test-lib",
				Ruby: &config.RubyPackage{},
				APIs: []*config.API{
					{
						Path: "google/cloud/test/v1",
						Ruby: &config.RubyAPI{},
					},
				},
			},
			want: &config.Library{
				Name: "test-lib",
				APIs: []*config.API{
					{
						Path: "google/cloud/test/v1",
					},
				},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := Tidy(test.lib)
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("Tidy() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
