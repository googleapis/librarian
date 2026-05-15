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
	"slices"
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
		if javaAPI.GenerateProtoGRPC == nil {
			javaAPI.GenerateProtoGRPC = new(true)
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
	if library.Output == deriveOutput(library.Name) {
		library.Output = ""
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
		if api.Java.GenerateProtoGRPC != nil && *api.Java.GenerateProtoGRPC {
			api.Java.GenerateProtoGRPC = nil
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
		j.GenerateProtoGRPC == nil &&
		j.GenerateResourceNames == nil &&
		len(j.CopyFiles) == 0 &&
		j.Samples == nil
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
	if library.Java != nil && library.Java.DistributionNameOverride != "" {
		parts := strings.Split(library.Java.DistributionNameOverride, ":")
		if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
			return fmt.Errorf("%w: %s: want \"groupId:artifactId\", got %q", ErrInvalidDistributionName, library.Name, library.Java.DistributionNameOverride)
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
