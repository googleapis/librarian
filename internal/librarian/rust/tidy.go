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

package rust

import (
	"slices"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/yaml"
)

// Tidy tidies the Rust-specific configuration for a library.
func Tidy(lib *config.Library) (*config.Library, error) {
	if lib.Rust != nil && lib.Rust.Modules != nil {
		lib.Rust.Modules = slices.DeleteFunc(lib.Rust.Modules, isEmptyModule)
	}

	empty, err := yaml.Empty(lib.Rust)
	if err != nil {
		return nil, err
	}
	if empty {
		lib.Rust = nil
	}

	return lib, nil
}

// isEmptyModule returns true if the module is a placeholder that can be removed.
func isEmptyModule(module *config.RustModule) bool {
	if module.Template == "storage" {
		// The Rust storage module has hardcoded API paths and templates, so it is never empty.
		return false
	}
	return module.APIPath == "" && module.Template == ""
}
