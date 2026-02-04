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

package internalsidekick

import (
	"strings"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/librarian/source"
)

// AddLibraryRoots configures the source roots for dart and rust generation.
//
// It populates a map with the paths to the necessary repositories based on the provided library configuration and
// available sources.
// If no specific roots are defined in the library configuration, it defaults to using googleapis.
func AddLibraryRoots(library *config.Library, sources *source.Sources) map[string]string {
	src := make(map[string]string)
	if len(library.Roots) == 0 && sources.Googleapis != "" {
		// Default to googleapis if no roots are specified.
		src["googleapis-root"] = sources.Googleapis
		src["roots"] = "googleapis"
	} else {
		src["roots"] = strings.Join(library.Roots, ",")
		rootMap := map[string]struct {
			path string
			key  string
		}{
			"googleapis":   {path: sources.Googleapis, key: "googleapis-root"},
			"discovery":    {path: sources.Discovery, key: "discovery-root"},
			"showcase":     {path: sources.Showcase, key: "showcase-root"},
			"protobuf-src": {path: sources.ProtobufSrc, key: "protobuf-src-root"},
			"conformance":  {path: sources.Conformance, key: "conformance-root"},
		}
		for _, root := range library.Roots {
			if r, ok := rootMap[root]; ok && r.path != "" {
				src[r.key] = r.path
			}
		}
	}

	return src
}
