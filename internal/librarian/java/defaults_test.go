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
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
)

func TestFill(t *testing.T) {
	for _, test := range []struct {
		name string
		lib  *config.Library
		want *config.Library
	}{
		{
			name: "fill output from name",
			lib: &config.Library{
				Name: "secretmanager",
			},
			want: &config.Library{
				Name:   "secretmanager",
				Output: "java-secretmanager",
				Java: &config.JavaModule{
					JavaAPIs: nil,
				},
			},
		},
		{
			name: "do not overwrite output",
			lib: &config.Library{
				Name:   "secretmanager",
				Output: "custom-output",
			},
			want: &config.Library{
				Name:   "secretmanager",
				Output: "custom-output",
				Java: &config.JavaModule{
					JavaAPIs: nil,
				},
			},
		},
		{
			name: "fill samples default",
			lib: &config.Library{
				Name: "secretmanager",
				APIs: []*config.API{
					{Path: "google/cloud/secretmanager/v1"},
				},
			},
			want: &config.Library{
				Name:   "secretmanager",
				Output: "java-secretmanager",
				APIs: []*config.API{
					{Path: "google/cloud/secretmanager/v1"},
				},
				Java: &config.JavaModule{
					JavaAPIs: []*config.JavaAPI{
						{
							Path:    "google/cloud/secretmanager/v1",
							Samples: new(true),
						},
					},
				},
			},
		},
		{
			name: "do not overwrite samples override",
			lib: &config.Library{
				Name: "secretmanager",
				APIs: []*config.API{
					{Path: "google/cloud/secretmanager/v1"},
				},
				Java: &config.JavaModule{
					JavaAPIs: []*config.JavaAPI{
						{
							Path:    "google/cloud/secretmanager/v1",
							Samples: new(false),
						},
					},
				},
			},
			want: &config.Library{
				Name:   "secretmanager",
				Output: "java-secretmanager",
				APIs: []*config.API{
					{Path: "google/cloud/secretmanager/v1"},
				},
				Java: &config.JavaModule{
					JavaAPIs: []*config.JavaAPI{
						{
							Path:    "google/cloud/secretmanager/v1",
							Samples: new(false),
						},
					},
				},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := Fill(test.lib)
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestTidy(t *testing.T) {
	for _, test := range []struct {
		name string
		lib  *config.Library
		want *config.Library
	}{
		{
			name: "tidy default output",
			lib: &config.Library{
				Name:   "secretmanager",
				Output: "java-secretmanager",
			},
			want: &config.Library{
				Name:   "secretmanager",
				Output: "",
			},
		},
		{
			name: "do not tidy custom output",
			lib: &config.Library{
				Name:   "secretmanager",
				Output: "custom-output",
			},
			want: &config.Library{
				Name:   "secretmanager",
				Output: "custom-output",
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := Tidy(test.lib)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestValidate(t *testing.T) {
	for _, test := range []struct {
		name string
		lib  *config.Library
	}{
		{
			name: "valid distribution name override",
			lib: &config.Library{
				Java: &config.JavaModule{
					DistributionNameOverride: "part1:part2",
				},
			},
		},
		{
			name: "empty java config",
			lib:  &config.Library{},
		},
		{
			name: "empty distribution name override",
			lib: &config.Library{
				Java: &config.JavaModule{},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			if err := Validate(test.lib); err != nil {
				t.Errorf("Validate(%+v) error = %v, want nil", test.lib, err)
			}
		})
	}
}

func TestValidate_Error(t *testing.T) {
	for _, test := range []struct {
		name    string
		lib     *config.Library
		wantErr error
	}{
		{
			name: "missing colon",
			lib: &config.Library{
				Java: &config.JavaModule{
					DistributionNameOverride: "nocolon",
				},
			},
			wantErr: ErrInvalidDistributionName,
		},
		{
			name: "too many colons",
			lib: &config.Library{
				Java: &config.JavaModule{
					DistributionNameOverride: "too:many:colons",
				},
			},
			wantErr: ErrInvalidDistributionName,
		},
		{
			name: "empty parts",
			lib: &config.Library{
				Java: &config.JavaModule{
					DistributionNameOverride: ":",
				},
			},
			wantErr: ErrInvalidDistributionName,
		},
		{
			name: "missing artifact id",
			lib: &config.Library{
				Java: &config.JavaModule{
					DistributionNameOverride: "groupid:",
				},
			},
			wantErr: ErrInvalidDistributionName,
		},
		{
			name: "missing group id",
			lib: &config.Library{
				Java: &config.JavaModule{
					DistributionNameOverride: ":artifactid",
				},
			},
			wantErr: ErrInvalidDistributionName,
		},
		{
			name: "omit common resources conflict",
			lib: &config.Library{
				Java: &config.JavaModule{
					JavaAPIs: []*config.JavaAPI{
						{
							Path:                "google/cloud/conflict/v1",
							OmitCommonResources: true,
							AdditionalProtos:    []string{"google/cloud/common_resources.proto"},
						},
					},
				},
			},
			wantErr: ErrOmitCommonResourcesConflict,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			err := Validate(test.lib)
			if !errors.Is(err, test.wantErr) {
				t.Errorf("Validate() error = %v, want %v", err, test.wantErr)
			}
		})
	}
}
