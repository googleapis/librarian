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

func TestDefaultLibraryName(t *testing.T) {
	for _, test := range []struct {
		apiPath string
		want    string
	}{
		{
			apiPath: "google/cloud/speech/v2",
			want:    "Speech",
		},
		{
			apiPath: "google/cloud/security/privateca/v1",
			want:    "SecurityPrivateca",
		},
		{
			apiPath: "google/cloud/bigquery/datatransfer/v1",
			want:    "BigqueryDatatransfer",
		},
		{
			apiPath: "google/pubsub/v1",
			want:    "Pubsub",
		},
		{
			apiPath: "google/cloud/vision",
			want:    "Vision",
		},
		{
			apiPath: "google/cloud/vision/v1",
			want:    "Vision",
		},
		{
			apiPath: "google/cloud/vision/v1p1beta1",
			want:    "Vision",
		},
	} {
		t.Run(test.apiPath, func(t *testing.T) {
			got := DefaultLibraryName(test.apiPath)
			if got != test.want {
				t.Errorf("DefaultLibraryName(%q) = %q, want %q", test.apiPath, got, test.want)
			}
		})
	}
}

func TestAdd(t *testing.T) {
	for _, test := range []struct {
		name string
		lib  *config.Library
		want *config.Library
	}{
		{
			name: "empty php config",
			lib: &config.Library{
				APIs: []*config.API{
					{Path: "google/cloud/speech/v2"},
				},
			},
			want: &config.Library{
				APIs: []*config.API{
					{
						Path: "google/cloud/speech/v2",
						PHP: &config.PHPAPI{
							StagingSubdir: "v2",
							MigrationMode: "NEW_SURFACE_ONLY",
						},
					},
				},
			},
		},
		{
			name: "existing php config preserved",
			lib: &config.Library{
				APIs: []*config.API{
					{
						Path: "google/cloud/speech/v2",
						PHP: &config.PHPAPI{
							StagingSubdir: "custom_subdir",
							MigrationMode: "MIGRATED",
						},
					},
				},
			},
			want: &config.Library{
				APIs: []*config.API{
					{
						Path: "google/cloud/speech/v2",
						PHP: &config.PHPAPI{
							StagingSubdir: "custom_subdir",
							MigrationMode: "MIGRATED",
						},
					},
				},
			},
		},
		{
			name: "multiple APIs",
			lib: &config.Library{
				APIs: []*config.API{
					{Path: "google/cloud/speech/v2"},
					{Path: "google/cloud/speech/v2beta1"},
				},
			},
			want: &config.Library{
				APIs: []*config.API{
					{
						Path: "google/cloud/speech/v2",
						PHP: &config.PHPAPI{
							StagingSubdir: "v2",
							MigrationMode: "NEW_SURFACE_ONLY",
						},
					},
					{
						Path: "google/cloud/speech/v2beta1",
						PHP: &config.PHPAPI{
							StagingSubdir: "v2beta1",
							MigrationMode: "NEW_SURFACE_ONLY",
						},
					},
				},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := Add(test.lib)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("Add() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
