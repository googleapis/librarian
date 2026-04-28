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

import "github.com/googleapis/librarian/internal/config"

// defaultVersion is the first version used for a new library. This is set on
// the initial `librarian add` for a new API.
const defaultVersion = "1.0.0"

// Add executes Rust-specific mutations of the given [config.Library]
// entry to be added to the librarian.yaml via `librarian add`.
//
// Currently, it only sets the [config.Library.Version] property to the
// [defaultVersion] for Rust.
func Add(lib *config.Library) *config.Library {
	lib.Version = defaultVersion
	return lib
}
