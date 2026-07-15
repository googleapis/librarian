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
			name: "nil_configurations",
			in: &config.Library{
				APIs: []*config.API{
					{
						Path: "google/cloud/secretmanager/v1",
					},
				},
			},
			want: &config.Library{
				APIs: []*config.API{
					{
						Path: "google/cloud/secretmanager/v1",
					},
				},
			},
		},
		{
			name: "nilifies_empty_package_config",
			in: &config.Library{
				APIs: []*config.API{
					{
						PHP: &config.PHPAPI{},
					},
				},
			},
			want: &config.Library{
				APIs: []*config.API{
					{},
				},
			},
		},
		{
			name: "sorts_and_deduplicates_API_additional_protos",
			in: &config.Library{
				APIs: []*config.API{
					{
						PHP: &config.PHPAPI{
							AdditionalProtos: []string{"b.proto", "a.proto", "a.proto"},
						},
					},
				},
			},
			want: &config.Library{
				APIs: []*config.API{
					{
						PHP: &config.PHPAPI{
							AdditionalProtos: []string{"a.proto", "b.proto"},
						},
					},
				},
			},
		},
		{
			name: "keeps_true_CommonResources",
			in: &config.Library{
				APIs: []*config.API{
					{
						PHP: &config.PHPAPI{
							CommonResources: new(true),
						},
					},
				},
			},
			want: &config.Library{
				APIs: []*config.API{
					{
						PHP: &config.PHPAPI{
							CommonResources: new(true),
						},
					},
				},
			},
		},
		{
			name: "keeps_false_CommonResources",
			in: &config.Library{
				APIs: []*config.API{
					{
						PHP: &config.PHPAPI{
							CommonResources: new(false),
						},
					},
				},
			},
			want: &config.Library{
				APIs: []*config.API{
					{
						PHP: &config.PHPAPI{
							CommonResources: new(false),
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

func TestValidate(t *testing.T) {
	for _, test := range []struct {
		name    string
		cfg     *config.Config
		wantErr bool
	}{
		{
			name: "valid when configured per-API",
			cfg: &config.Config{
				Language: config.LanguagePhp,
				Libraries: []*config.Library{
					{
						Name: "secretmanager",
						APIs: []*config.API{
							{
								Path: "google/cloud/secretmanager/v1",
								PHP: &config.PHPAPI{
									CommonResources: new(true),
								},
							},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "valid when configured globally",
			cfg: &config.Config{
				Language: config.LanguagePhp,
				Default: &config.Default{
					PHP: &config.PHPDefault{
						CommonResources: new(true),
					},
				},
				Libraries: []*config.Library{
					{
						Name: "secretmanager",
						APIs: []*config.API{
							{
								Path: "google/cloud/secretmanager/v1",
							},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "invalid when not configured anywhere",
			cfg: &config.Config{
				Language: config.LanguagePhp,
				Libraries: []*config.Library{
					{
						Name: "secretmanager",
						APIs: []*config.API{
							{
								Path: "google/cloud/secretmanager/v1",
							},
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "skips other languages",
			cfg: &config.Config{
				Language: config.LanguageGo,
				Libraries: []*config.Library{
					{
						Name: "secretmanager",
						APIs: []*config.API{
							{
								Path: "google/cloud/secretmanager/v1",
							},
						},
					},
				},
			},
			wantErr: false,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			err := Validate(test.cfg)
			if (err != nil) != test.wantErr {
				t.Errorf("Validate() error = %v, wantErr = %v", err, test.wantErr)
			}
		})
	}
}
