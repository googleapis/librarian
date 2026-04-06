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
	"fmt"
	"strings"

	"github.com/googleapis/librarian/internal/config"
)

const javaPrefix = "java-"

// deriveOutput computes the default output directory name for a given library name.
func deriveOutput(name string) string {
	return javaPrefix + name
}

// Fill populates Java-specific default values for the library.
func Fill(library *config.Library) (*config.Library, error) {
	if library.Output == "" {
		library.Output = deriveOutput(library.Name)
	}
	return library, nil
}

// Tidy tidies the Java-specific configuration for a library by removing default
// values.
func Tidy(library *config.Library) *config.Library {
	if library.Output == deriveOutput(library.Name) {
		library.Output = ""
	}
	return library
}

var (
	// ErrInvalidDistributionName is returned when a distribution name override
	// is incorrectly formatted.
	ErrInvalidDistributionName = fmt.Errorf("invalid distribution name override")
)

// Validate checks that the Java-specific configuration for a library is
// correctly formatted. It ensures that the distribution name override
// contains exactly two parts (GroupID and ArtifactID) separated by a colon.
func Validate(library *config.Library) error {
	if library.Java == nil || library.Java.DistributionNameOverride == "" {
		return nil
	}
	if len(strings.Split(library.Java.DistributionNameOverride, ":")) != 2 {
		return fmt.Errorf("%w: %s: want 2 parts separated by colon, got %q", ErrInvalidDistributionName, library.Name, library.Java.DistributionNameOverride)
	}
	return nil
}
