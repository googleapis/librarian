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

package nodejs

import (
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/yaml"
)

// Tidy tidies configuration for a library.
func Tidy(lib *config.Library) (*config.Library, error) {
	if lib.Nodejs == nil {
		return lib, nil
	}
	empty, err := yaml.Empty(lib.Nodejs)
	if err != nil {
		return nil, err
	}
	if empty {
		lib.Nodejs = nil
	}
	return lib, nil
}
