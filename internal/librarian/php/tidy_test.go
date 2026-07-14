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

package php

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
)

func TestTidy(t *testing.T) {
	for _, test := range []struct {
		name string
		in   *config.Library
		want *config.Library
	}{
		{
			name: "nil configurations",
			in:   &config.Library{},
			want: &config.Library{},
		},
		{
			name: "nilifies empty package config",
			in: &config.Library{
				PHP: &config.PHPPackage{},
			},
			want: &config.Library{
				PHP: nil,
			},
		},
		{
			name: "sorts and deduplicates API additional protos",
			in: &config.Library{
				APIs: []*config.API{
					{
						Path: "google/cloud/ces/v1",
						PHP: &config.PHPAPI{
							AdditionalProtos: []string{"d.proto", "b.proto", "d.proto", "a.proto", "b.proto"},
						},
					},
				},
			},
			want: &config.Library{
				APIs: []*config.API{
					{
						Path: "google/cloud/ces/v1",
						PHP: &config.PHPAPI{
							AdditionalProtos: []string{"a.proto", "b.proto", "d.proto"},
						},
					},
				},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := Tidy(test.in)
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
