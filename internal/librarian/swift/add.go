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

package swift

import "github.com/googleapis/librarian/internal/config"

// defaultVersion provides a default value in case `librarian.yaml` is missing it.
const defaultVersion = "0.0.0-preview"

// Add executes Swift-specific mutations of the given [config.Library]
// entry to be added to the librarian.yaml via `librarian add`.
func Add(lib *config.Library, cfg *config.Config) *config.Library {
	lib.Version = defaultVersion
	if cfg.Default != nil && cfg.Default.Swift != nil && cfg.Default.Swift.DefaultVersion != "" {
		lib.Version = cfg.Default.Swift.DefaultVersion
	}
	return lib
}
