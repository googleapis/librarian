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

package language

import (
	"fmt"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/language/internal/rust"
)

// ReleaseAll bumps versions for all libraries and updates librarian.yaml and
// other release artifacts for the language.
func ReleaseAll(cfg *config.Config) (*config.Config, error) {
	switch cfg.Language {
	case "testhelper":
		return cfg, nil
	case "rust":
		return rust.ReleaseAll(cfg)
	default:
		return nil, fmt.Errorf("language not supported for release --all: %q", cfg.Language)
	}
}

// ReleaseLibrary bumps versions for one library and updates librarian.yaml and
// other release artifacts for the language.
func ReleaseLibrary(cfg *config.Config, name string) (*config.Config, error) {
	switch cfg.Language {
	case "testhelper":
		return cfg, nil
	case "rust":
		return rust.ReleaseLibrary(cfg, name)
	default:
		return nil, fmt.Errorf("language not supported for release --all: %q", cfg.Language)
	}
}
