// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package config

import (
	"bytes"
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
	"gopkg.in/yaml.v3"
)

func TestReadWrite(t *testing.T) {
	cfg, err := Read("testdata/rust/librarian.yaml")
	if err != nil {
		t.Fatal(err)
	}
	var got bytes.Buffer
	enc := yaml.NewEncoder(&got)
	enc.SetIndent(2)
	if err := enc.Encode(cfg); err != nil {
		t.Fatal(err)
	}

	wantCfg, err := Read("testdata/rust/librarian.yaml")
	if err != nil {
		t.Fatal(err)
	}
	var gotCfg Config
	if err := yaml.Unmarshal(got.Bytes(), &gotCfg); err != nil {
		t.Fatal(err)
	}

	if diff := cmp.Diff(wantCfg, &gotCfg); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestLibraryByName(t *testing.T) {
	for _, test := range []struct {
		name        string
		libraryName string
		config      *Config
		want        *Library
		wantErr     error
	}{
		{
			name:        "find_a_library",
			libraryName: "example-library",
			config: &Config{
				Libraries: []*Library{
					{
						Name: "example-library",
					},
					{
						Name: "another-library",
					},
				},
			},
			want: &Library{
				Name: "example-library",
			},
		},
		{
			name:        "no_library_in_config",
			libraryName: "example-library",
			config:      &Config{},
			wantErr:     errLibraryNotFound,
		},
		{
			name:        "does_not_find_a_library",
			libraryName: "non-existent-library",
			config: &Config{
				Libraries: []*Library{
					{
						Name: "example-library",
					},
					{
						Name: "another-library",
					},
				},
			},
			wantErr: errLibraryNotFound,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := test.config.LibraryByName(test.libraryName)
			if test.wantErr != nil {
				if !errors.Is(err, test.wantErr) {
					t.Errorf("got error %v, want %v", err, test.wantErr)
				}
				return
			}

			if err != nil {
				t.Errorf("FindByName(%q): %v", test.libraryName, err)
				return
			}

			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
