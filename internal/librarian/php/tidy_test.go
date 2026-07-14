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
	trueVal := true
	falseVal := false

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
		{
			name: "removes default true CommonResources and empty PHP structs",
			in: &config.Library{
				APIs: []*config.API{
					{
						Path: "google/cloud/secretmanager/v1",
						PHP: &config.PHPAPI{
							CommonResources: &trueVal,
						},
					},
				},
				PHP: &config.PHPPackage{},
			},
			want: &config.Library{
				APIs: []*config.API{
					{
						Path: "google/cloud/secretmanager/v1",
						PHP:  nil,
					},
				},
				PHP: nil,
			},
		},
		{
			name: "keeps false CommonResources",
			in: &config.Library{
				APIs: []*config.API{
					{
						Path: "google/cloud/secretmanager/v1",
						PHP: &config.PHPAPI{
							CommonResources: &falseVal,
						},
					},
				},
				PHP: &config.PHPPackage{},
			},
			want: &config.Library{
				APIs: []*config.API{
					{
						Path: "google/cloud/secretmanager/v1",
						PHP: &config.PHPAPI{
							CommonResources: &falseVal,
						},
					},
				},
				PHP: nil,
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
