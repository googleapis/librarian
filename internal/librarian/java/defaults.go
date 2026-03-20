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

package java

import (
	"github.com/googleapis/librarian/internal/config"
)

var javaPrefix = "java-"

// Fill populates Java-specific default values for the library.
func Fill(library *config.Library) (*config.Library, error) {
	if library.Output == "" {
		library.Output = javaPrefix + library.Name
	}
	return library, nil
}

// Tidy tidies the Java-specific configuration for a library by removing default
// values.
func Tidy(library *config.Library) *config.Library {
	if library.Output == javaPrefix+library.Name {
		library.Output = ""
	}
	return library
}
