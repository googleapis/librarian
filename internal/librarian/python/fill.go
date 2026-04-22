// Copyright 2025 Google LLC
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

package python

import (
	"strings"

	"github.com/googleapis/librarian/internal/config"
)

// Fill populates empty Python-specific fields.
// Library configurations takes precedence.
func Fill(library *config.Library) (*config.Library, error) {
	if library.Preview != nil {
		fillPythonPreview(library, library.Preview)
	}

	return library, nil
}

// fillPythonPreview fills the [Library.Python] section of the [Library.Preview] and
// returns the filled Library.Preview. This must be called after the containing
// [Library] has been filled already.
func fillPythonPreview(stable, preview *config.Library) *config.Library {
	// Preview clients are generated into a dedicated subdirectory that lives
	// alongside the stable client packages directory.
	if preview.Output == "" {
		preview.Output = strings.Replace(stable.Output, "packages", "preview-packages", 1)
	}

	if stable.Python == nil {
		return preview
	}

	if preview.Python == nil {
		preview.Python = &config.PythonPackage{}
	}

	p, s := preview.Python, stable.Python

	// Merge stable into preview, favoring preview.
	if len(p.OptArgsByAPI) == 0 && len(s.OptArgsByAPI) > 0 {
		filtered := make(map[string][]string)
		for _, api := range preview.APIs {
			if args, ok := s.OptArgsByAPI[api.Path]; ok {
				filtered[api.Path] = args
			}
		}
		p.OptArgsByAPI = filtered
	}

	return preview
}
