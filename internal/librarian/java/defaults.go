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
	if library.Java == nil {
		library.Java = &config.JavaModule{}
	}
	var javaAPIs []*config.JavaAPI
	for _, api := range library.APIs {
		javaAPI := ResolveJavaAPI(library, api)
		if javaAPI.Samples == nil {
			javaAPI.Samples = new(true)
		}
		javaAPIs = append(javaAPIs, javaAPI)
	}
	library.Java.JavaAPIs = javaAPIs
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

	// ErrOmitCommonResourcesConflict is returned when OmitCommonResources is true
	// but common_resources.proto is also explicitly listed in AdditionalProtos.
	ErrOmitCommonResourcesConflict = fmt.Errorf("conflict: OmitCommonResources is true but google/cloud/common_resources.proto is explicitly listed in AdditionalProtos")
)

// Validate checks that the Java-specific configuration for a library is
// correctly formatted. It ensures that the distribution name override
// contains exactly two parts separated by a colon, and that there are no
// conflicts in common resources configuration.
func Validate(library *config.Library) error {
	if library.Java == nil {
		return nil
	}
	if library.Java.DistributionNameOverride != "" {
		parts := strings.Split(library.Java.DistributionNameOverride, ":")
		if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
			return fmt.Errorf("%w: %s: want \"groupId:artifactId\", got %q", ErrInvalidDistributionName, library.Name, library.Java.DistributionNameOverride)
		}
	}
	for _, javaAPI := range library.Java.JavaAPIs {
		if !javaAPI.OmitCommonResources {
			continue
		}
		for _, p := range javaAPI.AdditionalProtos {
			if p == commonResourcesProto {
				return fmt.Errorf("%s: %w", javaAPI.Path, ErrOmitCommonResourcesConflict)
			}
		}
	}
	return nil
}
