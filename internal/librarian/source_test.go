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

package librarian

import (
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
	sidekickconfig "github.com/googleapis/librarian/internal/sidekick/config"
)

func TestLoadSources(t *testing.T) {
	for _, test := range []struct {
		name    string
		src     *config.Sources
		want    *sidekickconfig.Sources
		wantErr error
	}{
		{
			name: "success with pre-configured directories",
			src: &config.Sources{
				Googleapis:  &config.Source{Dir: "/path/to/googleapis"},
				Conformance: &config.Source{Dir: "/path/to/conformance"},
				Discovery:   &config.Source{Dir: "/path/to/discovery"},
				Showcase:    &config.Source{Dir: "/path/to/showcase"},
				ProtobufSrc: &config.Source{Dir: "/path/to/protobuf", Subpath: "src"},
			},
			want: &sidekickconfig.Sources{
				Googleapis:  "/path/to/googleapis",
				Conformance: "/path/to/conformance",
				Discovery:   "/path/to/discovery",
				ProtobufSrc: "/path/to/protobuf/src",
				Showcase:    "/path/to/showcase",
			},
		},
		{
			name:    "nil sources",
			src:     nil,
			wantErr: ErrMissingGoogleapisSource,
		},
		{
			name:    "empty sources",
			src:     &config.Sources{},
			wantErr: ErrMissingGoogleapisSource,
		},
		{
			name: "googleapis dir set",
			src: &config.Sources{
				Googleapis: &config.Source{Dir: "/tmp/googleapis"},
			},
			want: &sidekickconfig.Sources{
				Googleapis: "/tmp/googleapis",
			},
		},
		{
			name: "discovery dir set",
			src: &config.Sources{
				Googleapis: &config.Source{Dir: "/tmp/googleapis"},
				Discovery:  &config.Source{Dir: "/tmp/discovery"},
			},
			want: &sidekickconfig.Sources{
				Googleapis: "/tmp/googleapis",
				Discovery:  "/tmp/discovery",
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := LoadSources(t.Context(), test.src)
			if test.wantErr != nil {
				if !errors.Is(err, test.wantErr) {
					t.Errorf("LoadSources() got error = %v, wantErr %v", err, test.wantErr)
				}
				return
			}
			if err != nil {
				t.Errorf("LoadSources() got unexpected error: %v", err)
			}
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("LoadSources() mismatch (-want +got):%s", diff)
			}
		})
	}
}
