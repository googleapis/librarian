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
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/language/internal/rust"
)

// Releaser provides release functions for a specific language.
type Releaser struct {
	ReleaseAll     func(cfg *config.Config) (*config.Config, error)
	ReleaseLibrary func(cfg *config.Config, name string) (*config.Config, error)
}

// Releasers maps language names to their Releaser implementations.
var Releasers = map[string]Releaser{
	"rust": {
		ReleaseAll:     rust.ReleaseAll,
		ReleaseLibrary: rust.ReleaseOne,
	},
	"testhelper": {
		ReleaseAll:     testReleaseAll,
		ReleaseLibrary: testReleaseLibrary,
	},
}
