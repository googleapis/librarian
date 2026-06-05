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
	"path/filepath"
	"slices"
	"strings"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/yaml"
)

const (
	javaPrefix              = "java-"
	defaultArtifactIDPrefix = "google-cloud-"
	defaultGroupID          = "com.google.cloud"
)

func deriveArtifactID(name string) string {
	return defaultArtifactIDPrefix + name
}

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
	if library.Java.ArtifactID == "" {
		library.Java.ArtifactID = deriveArtifactID(library.Name)
	}
	if library.Java.GroupID == "" {
		library.Java.GroupID = defaultGroupID
	}
	for _, api := range library.APIs {
		if api.Java == nil {
			api.Java = &config.JavaAPI{}
		}
		javaAPI := api.Java
		if javaAPI.Samples == nil {
			javaAPI.Samples = new(true)
		}
		if javaAPI.GenerateGAPIC == nil {
			javaAPI.GenerateGAPIC = new(true)
		}
		if javaAPI.GenerateProto == nil {
			javaAPI.GenerateProto = new(true)
		}
		if javaAPI.GenerateGRPC == nil {
			javaAPI.GenerateGRPC = new(true)
		}
		if javaAPI.GenerateResourceNames == nil {
			javaAPI.GenerateResourceNames = new(true)
		}
	}
	return library, nil
}

// Tidy tidies the Java-specific configuration for a library by removing default
// values.
func Tidy(library *config.Library) (*config.Library, error) {
	library.Keep = tidyKeep(library)
	if library.Output == deriveOutput(library.Name) {
		library.Output = ""
	}
	if library.Java != nil {
		if library.Java.ArtifactID == deriveArtifactID(library.Name) {
			library.Java.ArtifactID = ""
		}
		if library.Java.GroupID == defaultGroupID {
			library.Java.GroupID = ""
		}
		empty, err := yaml.Empty(library.Java)
		if err != nil {
			return nil, err
		}
		if empty {
			library.Java = nil
		}
	}
	for _, api := range library.APIs {
		if api.Java == nil {
			continue
		}
		if api.Java.Samples != nil && *api.Java.Samples {
			api.Java.Samples = nil
		}
		if api.Java.GenerateGAPIC != nil && *api.Java.GenerateGAPIC {
			api.Java.GenerateGAPIC = nil
		}
		if api.Java.GenerateProto != nil && *api.Java.GenerateProto {
			api.Java.GenerateProto = nil
		}
		if api.Java.GenerateGRPC != nil && *api.Java.GenerateGRPC {
			api.Java.GenerateGRPC = nil
		}
		if api.Java.GenerateResourceNames != nil && *api.Java.GenerateResourceNames {
			api.Java.GenerateResourceNames = nil
		}
		api.Java.AdditionalProtos = slices.DeleteFunc(api.Java.AdditionalProtos, func(p *config.AdditionalProto) bool {
			return p == nil || p.Path == ""
		})
		empty, err := yaml.Empty(api.Java)
		if err != nil {
			return nil, err
		}
		if empty {
			api.Java = nil
		}
	}
	return library, nil
}

// tidyKeep removes default files from the library's keep configuration.
func tidyKeep(library *config.Library) []string {
	if len(library.Keep) == 0 {
		return nil
	}
	var filteredKeepPaths []string
	for _, keepPath := range library.Keep {
		keepPathSlash := filepath.ToSlash(keepPath)
		if isDefaultPreserved(keepPathSlash) {
			continue
		}
		if isInGAPICModule(library, keepPath) {
			continue
		}
		filteredKeepPaths = append(filteredKeepPaths, keepPath)
	}
	slices.Sort(filteredKeepPaths)
	filteredKeepPaths = slices.Compact(filteredKeepPaths)
	if len(filteredKeepPaths) == 0 {
		return nil
	}
	return filteredKeepPaths
}

// isInGAPICModule checks if the given keepPath belongs to a generated GAPIC module of the library.
func isInGAPICModule(library *config.Library, keepPath string) bool {
	keepPathSlash := filepath.ToSlash(keepPath)
	for pattern, isGAPIC := range cleanPatterns(library) {
		if isGAPIC {
			moduleName, _, _ := strings.Cut(filepath.ToSlash(pattern), "/")
			if strings.HasPrefix(keepPathSlash, moduleName+"/") {
				return true
			}
		}
	}
	return false
}

var (
	// ErrOmitCommonResourcesConflict is returned when OmitCommonResources is true
	// but common_resources.proto is also explicitly listed in AdditionalProtos.
	ErrOmitCommonResourcesConflict = fmt.Errorf("conflict: OmitCommonResources is true but google/cloud/common_resources.proto is explicitly listed in AdditionalProtos")
)

// Validate checks that the Java-specific configuration for a library is
// correctly formatted. It ensures that there are no conflicts in common
// resources configuration.
func Validate(library *config.Library) error {
	for _, api := range library.APIs {
		if api.Java == nil || !api.Java.OmitCommonResources {
			continue
		}
		for _, proto := range api.Java.AdditionalProtos {
			if proto != nil && proto.Path == commonResourcesProto {
				return fmt.Errorf("%s: %w", api.Path, ErrOmitCommonResourcesConflict)
			}
		}

	}
	return nil
}
