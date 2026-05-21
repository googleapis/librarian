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
func Tidy(library *config.Library) *config.Library {
	if len(library.Keep) > 0 {
		// 1. Find all GAPIC module directories to identify where marker cleaning applies.
		coordinates := DeriveLibraryCoordinates(library)
		seen := map[string]bool{coordinates.GAPIC.ArtifactID: true}
		gapicDirs := []string{coordinates.GAPIC.ArtifactID}
		for _, api := range library.APIs {
			if api.Java != nil && api.Java.GAPICArtifactIDOverride != "" && !seen[api.Java.GAPICArtifactIDOverride] {
				seen[api.Java.GAPICArtifactIDOverride] = true
				gapicDirs = append(gapicDirs, api.Java.GAPICArtifactIDOverride)
			}
		}
		var kept []string
		for _, keepPath := range library.Keep {
			keepPathSlash := filepath.ToSlash(keepPath)
			// 2. Skip files matching standard test/version regexes since they are preserved globally.
			if isDefaultPreserved(keepPathSlash) {
				continue
			}
			omit := false
			// 3. Skip manual files in GAPIC src folders since they are preserved by marker-based clean.
			for _, gapicDir := range gapicDirs {
				prefix := gapicDir + "/src/"
				if strings.HasPrefix(keepPathSlash, prefix) {
					filename := filepath.Base(keepPathSlash)
					if filename != "gapic_metadata.json" && filename != "reflect-config.json" {
						omit = true
						break
					}
				}
			}
			if !omit {
				kept = append(kept, keepPath)
			}
		}
		slices.Sort(kept)
		library.Keep = slices.Compact(kept)
		if len(library.Keep) == 0 {
			library.Keep = nil
		}
	}
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
		if isEmptyJavaModule(library.Java) {
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
		if isEmptyJavaAPI(api.Java) {
			api.Java = nil
		}
	}
	return library
}

func isEmptyJavaAPI(j *config.JavaAPI) bool {
	if j == nil {
		return true
	}
	return !j.Monolithic &&
		len(j.AdditionalProtos) == 0 &&
		!j.OmitCommonResources &&
		len(j.ExcludedProtos) == 0 &&
		len(j.SkipProtoClassGeneration) == 0 &&
		j.GAPICArtifactIDOverride == "" &&
		j.GRPCArtifactIDOverride == "" &&
		j.ProtoArtifactIDOverride == "" &&
		j.GenerateGAPIC == nil &&
		j.GenerateProto == nil &&
		j.GenerateGRPC == nil &&
		j.GenerateResourceNames == nil &&
		len(j.CopyFiles) == 0 &&
		j.Samples == nil
}

func isEmptyJavaModule(j *config.JavaModule) bool {
	if j == nil {
		return true
	}
	return j.APIIDOverride == "" &&
		j.APIReference == "" &&
		j.APIDescriptionOverride == "" &&
		j.APIShortnameOverride == "" &&
		j.ArtifactID == "" &&
		j.ClientDocumentationOverride == "" &&
		j.CodeownerTeam == "" &&
		j.ExcludedDependencies == "" &&
		len(j.ExcludedPOMs) == 0 &&
		j.ExtraVersionedModules == "" &&
		j.GroupID == "" &&
		j.IssueTrackerOverride == "" &&
		j.LibrariesBOMVersion == "" &&
		j.LibraryTypeOverride == "" &&
		j.MinJavaVersion == 0 &&
		j.NamePrettyOverride == "" &&
		j.ProductDocumentationOverride == "" &&
		j.RecommendedPackage == "" &&
		!j.BillingNotRequired &&
		j.RestDocumentation == "" &&
		j.RpcDocumentation == "" &&
		j.TransportOverride == "" &&
		!j.SkipPOMUpdates &&
		!j.SkipAPIID
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
