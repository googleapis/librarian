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

package source

import (
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
)

func TestFetchRustSources(t *testing.T) {
	for _, test := range []struct {
		name       string
		cfgSources *config.Sources
		want       *Sources
		wantErr    error
	}{
		{
			name: "success with pre-configured directories",
			cfgSources: &config.Sources{
				Conformance: &config.Source{Dir: "/path/to/conformance"},
				Discovery:   &config.Source{Dir: "/path/to/discovery"},
				Showcase:    &config.Source{Dir: "/path/to/showcase"},
				ProtobufSrc: &config.Source{Dir: "/path/to/protobuf", Subpath: "src"},
			},
			want: &Sources{
				Conformance: "/path/to/conformance",
				Discovery:   "/path/to/discovery",
				Googleapis:  "",
				ProtobufSrc: "/path/to/protobuf/src",
				Showcase:    "/path/to/showcase",
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := FetchRustDartSources(t.Context(), test.cfgSources)
			if test.wantErr != nil {
				if err == nil || !errors.Is(err, test.wantErr) {
					t.Errorf("FetchRustSources() got error = %v, wantErr %v", err, test.wantErr)
				}
				return
			}
			if err != nil {
				t.Errorf("FetchRustSources() got unexpected error: %v", got)
			}
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("FetchRustSources() mismatch (-want +got):%s", diff)
			}
		})
	}
}
