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
	"context"
	"fmt"
	"path/filepath"
	"sort"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/serviceconfig"
	"github.com/googleapis/librarian/internal/sources"
)

// ResolveDependencies automatically resolves Protobuf dependencies for a Java library.
func ResolveDependencies(ctx context.Context, cfg *config.Config, lib *config.Library, srcs *sources.Sources) (*config.Config, error) {
	if len(lib.APIs) == 0 {
		return cfg, nil
	}
	for _, apiCfg := range lib.APIs {
		if err := resolveAPIDependencies(lib, apiCfg, srcs); err != nil {
			return nil, err
		}
	}
	return cfg, nil
}

func resolveAPIDependencies(lib *config.Library, apiCfg *config.API, srcs *sources.Sources) error {
	if apiCfg.Java == nil {
		apiCfg.Java = &config.JavaAPI{}
	}

	primaryRoot := srcs.Googleapis
	if apiCfg.Path == "schema/google/showcase/v1beta1" {
		primaryRoot = srcs.Showcase
	}

	svcConfig, err := serviceconfig.Find(primaryRoot, apiCfg.Path, config.LanguageJava)
	if err != nil {
		return fmt.Errorf("failed to find service config for %s: %w", apiCfg.Path, err)
	}

	if svcConfig.ServiceConfig == "" {
		return nil
	}

	serviceConfig, err := serviceconfig.Read(filepath.Join(primaryRoot, svcConfig.ServiceConfig))
	if err != nil {
		return fmt.Errorf("failed to read service config for %s: %w", apiCfg.Path, err)
	}

	for _, api := range serviceConfig.GetApis() {
		var mixinProto string
		switch api.GetName() {
		case "google.cloud.location.Locations":
			mixinProto = "google/cloud/location/locations.proto"
		case "google.iam.v1.IAMPolicy":
			mixinProto = "google/iam/v1/iam_policy.proto"
		default:
			continue
		}

		if !hasAdditionalProto(apiCfg.Java.AdditionalProtos, mixinProto) {
			apiCfg.Java.AdditionalProtos = append(apiCfg.Java.AdditionalProtos, &config.AdditionalProto{
				Path:                 mixinProto,
				GenerateProtoClasses: false,
				CopyToOutput:         false,
			})
		}
	}

	sortAdditionalProtos(apiCfg.Java.AdditionalProtos)
	return nil
}

func hasAdditionalProto(protos []*config.AdditionalProto, path string) bool {
	for _, p := range protos {
		if p.Path == path {
			return true
		}
	}
	return false
}

func sortAdditionalProtos(protos []*config.AdditionalProto) {
	sort.Slice(protos, func(i, j int) bool {
		return protos[i].Path < protos[j].Path
	})
}
