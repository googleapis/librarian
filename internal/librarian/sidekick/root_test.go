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

package sidekick

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/sidekick/source"
)

func TestAddLibraryRoots(t *testing.T) {
	for _, test := range []struct {
		name    string
		library *config.Library
		source  *source.Sources
		want    map[string]string
	}{
		{
			name:    "empty roots",
			library: &config.Library{},
			source: &source.Sources{
				Googleapis: "example/path",
			},
			want: map[string]string{
				"googleapis-root": "example/path",
				"roots":           "googleapis",
			},
		},
		{
			name: "non existed sources",
			library: &config.Library{
				Roots: []string{"non-existed", "googleapis"},
			},
			source: &source.Sources{
				Googleapis:  "example/path",
				ProtobufSrc: "protobuf/path",
			},
			want: map[string]string{
				"googleapis-root": "example/path",
				"roots":           "non-existed,googleapis",
			},
		},
		{
			name: "all sources",
			library: &config.Library{
				Roots: []string{
					"conformance",
					"discovery",
					"googleapis",
					"protobuf-src",
					"showcase",
				},
			},
			source: &source.Sources{
				Conformance: "conformance/path",
				Discovery:   "discovery/path",
				Googleapis:  "googleapis/path",
				ProtobufSrc: "protobuf/path",
				Showcase:    "showcase/path",
			},
			want: map[string]string{
				"conformance-root":  "conformance/path",
				"discovery-root":    "discovery/path",
				"googleapis-root":   "googleapis/path",
				"protobuf-src-root": "protobuf/path",
				"showcase-root":     "showcase/path",
				"roots":             "conformance,discovery,googleapis,protobuf-src,showcase",
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := AddLibraryRoots(test.library, test.source)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
