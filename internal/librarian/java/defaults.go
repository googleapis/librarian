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
	"errors"
	"fmt"
	"path/filepath"
	"slices"
	"strings"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/semver"
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
	if library.Java.ReleasedVersion == "" && !library.SkipGenerate && library.Version != "" {
		derived, err := deriveLastReleasedVersion(library.Version)
		if err != nil {
			return nil, fmt.Errorf("library %q: %w", library.Name, err)
		}
		library.Java.ReleasedVersion = derived
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
	library.Keep = tidyKeep(library.Keep)
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
		tidyReleasedVersion(library)
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
func tidyKeep(keep []string) []string {
	if len(keep) == 0 {
		return nil
	}
	var filteredKeepPaths []string
	for _, keepPath := range keep {
		keepPathSlash := filepath.ToSlash(keepPath)
		if isDefaultPreserved(keepPathSlash) {
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

var (
	// ErrOmitCommonResourcesConflict is returned when OmitCommonResources is true
	// but common_resources.proto is also explicitly listed in AdditionalProtos.
	ErrOmitCommonResourcesConflict = errors.New("conflict: OmitCommonResources is true but google/cloud/common_resources.proto is explicitly listed in AdditionalProtos")
	// ErrCannotDeriveReleasedVersion is returned when released_version cannot be derived.
	ErrCannotDeriveReleasedVersion = errors.New("cannot derive released version")
)

// Validate checks that the Java-specific configuration for a library is
// correctly formatted. It ensures that there are no conflicts in common
// resources configuration.
func Validate(library *config.Library) error {
	if library.Version != "" {
		if _, err := semver.Parse(library.Version); err != nil {
			return fmt.Errorf("library %q: invalid version %q: %w", library.Name, library.Version, err)
		}
	}
	if !library.SkipGenerate && library.Java != nil && library.Java.ReleasedVersion != "" {
		if _, err := semver.Parse(library.Java.ReleasedVersion); err != nil {
			return fmt.Errorf("library %q: invalid released_version %q: %w", library.Name, library.Java.ReleasedVersion, err)
		}
	}
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

// deriveLastReleasedVersion derives the last released version from a snapshot version
// (e.g., x.y.z-SNAPSHOT or x.y.z-beta-SNAPSHOT) by decrementing the patch or minor version.
// If the version has a prerelease tag (like "beta") before "SNAPSHOT", that tag is preserved
// in the derived version (e.g., x.y.z-beta-SNAPSHOT becomes x.y.(z-1)-beta).
//
// It returns an error if both minor and patch versions are zero, as it's
// ambiguous what the last released version was in that case.
func deriveLastReleasedVersion(v string) (string, error) {
	sv, err := semver.Parse(v)
	if err != nil {
		return "", err
	}
	if !strings.HasSuffix(sv.Prerelease, "SNAPSHOT") {
		return sv.String(), nil
	}
	if sv.Patch > 0 {
		sv.Patch--
	} else if sv.Minor > 0 {
		sv.Minor--
		sv.Patch = 0
	} else {
		return "", ErrCannotDeriveReleasedVersion
	}
	sv.Prerelease = strings.TrimSuffix(sv.Prerelease, "SNAPSHOT")
	sv.Prerelease = strings.TrimSuffix(sv.Prerelease, "-")
	return sv.String(), nil
}

// isSnapshot reports whether the version represents a SNAPSHOT release.
func isSnapshot(v string) bool {
	sv, err := semver.Parse(v)
	return err == nil && strings.HasSuffix(sv.Prerelease, "SNAPSHOT")
}

// tidyReleasedVersion clears the Java module's released_version if it can be
// derived from the library version. It only tidies when the current library
// version is a SNAPSHOT.
func tidyReleasedVersion(library *config.Library) {
	if library.Java.ReleasedVersion == "" {
		return
	}
	if !isSnapshot(library.Version) {
		return
	}
	derived, err := deriveLastReleasedVersion(library.Version)
	if err == nil && library.Java.ReleasedVersion == derived {
		library.Java.ReleasedVersion = ""
	}
}
