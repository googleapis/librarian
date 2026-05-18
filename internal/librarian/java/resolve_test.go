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

package java

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
	for _, tt := range []struct {
		name    string
		lib     *config.Library
		want    []*config.AdditionalProto
		wantErr bool
	}{
		{
			name: "no APIs",
			lib: &config.Library{
				APIs: []*config.API{},
			},
			want: nil,
		},
		{
			name: "locations mixin (developerconnect)",
			lib: &config.Library{
				APIs: []*config.API{
					{Path: "google/cloud/developerconnect/v1"},
				},
			},
			want: []*config.AdditionalProto{
				{
					Path:                 "google/cloud/location/locations.proto",
					GenerateProtoClasses: false,
					CopyToOutput:         false,
				},
			},
		},
		{
			name: "locations mixin (secretmanager)",
			lib: &config.Library{
				APIs: []*config.API{
					{Path: "google/cloud/secretmanager/v1"},
				},
			},
			want: []*config.AdditionalProto{
				{
					Path:                 "google/cloud/location/locations.proto",
					GenerateProtoClasses: false,
					CopyToOutput:         false,
				},
			},
		},
		{
			name: "no mixins (dataproc)",
			lib: &config.Library{
				APIs: []*config.API{
					{Path: "google/cloud/dataproc/v1"},
				},
			},
			want: nil,
		},
		{
			name: "preserve existing and sort",
			lib: &config.Library{
				APIs: []*config.API{
					{
						Path: "google/cloud/developerconnect/v1",
						Java: &config.JavaAPI{
							AdditionalProtos: []*config.AdditionalProto{
								{
									Path: "google/example/v1/example.proto",
								},
							},
						},
					},
				},
			},
			want: []*config.AdditionalProto{
				{
					Path:                 "google/cloud/location/locations.proto",
					GenerateProtoClasses: false,
					CopyToOutput:         false,
				},
				{
					Path: "google/example/v1/example.proto",
				},
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ResolveMixinDependencies(&config.Config{}, tt.lib, srcs)
			if (err != nil) != tt.wantErr {
				t.Errorf("ResolveDependencies() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil {
				return
			}

			var got []*config.AdditionalProto
			if len(tt.lib.APIs) > 0 && tt.lib.APIs[0].Java != nil {
				got = tt.lib.APIs[0].Java.AdditionalProtos
			}

			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("ResolveDependencies() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
