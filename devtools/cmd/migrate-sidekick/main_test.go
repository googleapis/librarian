// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package migrate_sidekick provides tools to create a librarian configuration from
// .sidekick.toml in a repository.
package migrate_sidekick

import (
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
)

func TestReadRootSidekick(t *testing.T) {
	for _, test := range []struct {
		name    string
		path    string
		want    *RootDefaults
		wantErr error
	}{
		{
			name: "success",
			path: "testdata/rootSidekick/success",
			want: &RootDefaults{
				DisabledRustdocWarnings: []string{
					"redundant_explicit_links",
					"broken_intra_doc_links",
				},
				PackageDependencies: []*config.RustPackageDependency{
					{
						Feature: "_internal-http-client",
						Name:    "gaxi",
						Package: "google-cloud-gax-internal",
						Source:  "internal",
						UsedIf:  "services",
					},
					{
						Name:      "lazy_static",
						Package:   "lazy_static",
						UsedIf:    "services",
						ForceUsed: true,
					},
				},
				Remote: "upstream",
				Branch: "main",
			},
		},
		{
			name:    "no_sidekick_file",
			path:    "testdata/rootSidekick/no_sidekick_file",
			wantErr: errSidekickNotFound,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := readRootSidekick(test.path)
			if test.wantErr != nil {
				if !errors.Is(err, test.wantErr) {
					t.Errorf("got error %v, want %v", err, test.wantErr)
				}
				return
			}

			if err != nil {
				t.Errorf("got error %v, want nil", err)
				return
			}
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
