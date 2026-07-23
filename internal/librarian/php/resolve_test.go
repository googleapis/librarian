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
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/sources"
)

func TestResolveDependencies(t *testing.T) {
	googleapisDir, err := filepath.Abs("../../testdata/googleapis")
	if err != nil {
		t.Fatal(err)
	}
	srcs := &sources.Sources{
		Googleapis: googleapisDir,
	}
	for _, test := range []struct {
		name string
		lib  *config.Library
		want []string
	}{
		{
			name: "no APIs",
			lib: &config.Library{
				APIs: []*config.API{},
			},
		},
		{
			name: "locations mixin (developerconnect)",
			lib: &config.Library{
				APIs: []*config.API{
					{Path: "google/cloud/developerconnect/v1"},
				},
			},
			want: []string{
				"google/cloud/location/locations.proto",
			},
		},
		{
			name: "locations mixin (secretmanager)",
			lib: &config.Library{
				APIs: []*config.API{
					{Path: "google/cloud/secretmanager/v1"},
				},
			},
			want: []string{
				"google/cloud/location/locations.proto",
			},
		},
		{
			name: "no mixins (dataproc)",
			lib: &config.Library{
				APIs: []*config.API{
					{Path: "google/cloud/dataproc/v1"},
				},
			},
		},
		{
			name: "preserve existing and sort",
			lib: &config.Library{
				APIs: []*config.API{
					{
						Path: "google/cloud/developerconnect/v1",
						PHP: &config.PHPAPI{
							AdditionalProtos: []string{
								"google/example/v1/example.proto",
							},
						},
					},
				},
			},
			want: []string{
				"google/cloud/location/locations.proto",
				"google/example/v1/example.proto",
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, err := ResolveMixinDependencies(&config.Config{}, test.lib, srcs)
			if err != nil {
				t.Fatal(err)
			}
			var got []string
			if len(test.lib.APIs) > 0 && test.lib.APIs[0].PHP != nil {
				got = test.lib.APIs[0].PHP.AdditionalProtos
			}
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
