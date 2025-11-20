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

package rust

import (
	"strings"

	"github.com/googleapis/librarian/internal/sideflip/config"
	sidekickconfig "github.com/googleapis/librarian/internal/sidekick/config"
)

func toSidekickConfig(library *config.Library, googleapisDir, serviceConfig string) *sidekickconfig.Config {
	sidekickCfg := &sidekickconfig.Config{
		General: sidekickconfig.GeneralConfig{
			Language:            "rust",
			SpecificationFormat: "protobuf",
			ServiceConfig:       serviceConfig,
			SpecificationSource: library.Channel,
		},
		Source: map[string]string{
			"googleapis-root": googleapisDir,
		},
		Codec: buildCodec(library),
	}

	if library.Rust != nil {
		if len(library.Rust.DocumentationOverrides) > 0 {
			sidekickCfg.CommentOverrides = make([]sidekickconfig.DocumentationOverride, len(library.Rust.DocumentationOverrides))
			for i, override := range library.Rust.DocumentationOverrides {
				sidekickCfg.CommentOverrides[i] = sidekickconfig.DocumentationOverride{
					ID:      override.ID,
					Match:   override.Match,
					Replace: override.Replace,
				}
			}
		}

		if len(library.Rust.PaginationOverrides) > 0 {
			sidekickCfg.PaginationOverrides = make([]sidekickconfig.PaginationOverride, len(library.Rust.PaginationOverrides))
			for i, override := range library.Rust.PaginationOverrides {
				sidekickCfg.PaginationOverrides[i] = sidekickconfig.PaginationOverride{
					ID:        override.ID,
					ItemField: override.ItemField,
				}
			}
		}
	}
	return sidekickCfg
}

func buildCodec(library *config.Library) map[string]string {
	codec := make(map[string]string)
	if library.Version != "" {
		codec["version"] = library.Version
	}
	if library.ReleaseLevel != "" {
		codec["release-level"] = library.ReleaseLevel
	}
	if library.Name != "" {
		codec["package-name-override"] = library.Name
	}
	if library.CopyrightYear != "" {
		codec["copyright-year"] = library.CopyrightYear
	}
	if library.Rust == nil {
		return codec
	}

	rust := library.Rust
	if rust.ModulePath != "" {
		codec["module-path"] = rust.ModulePath
	}
	if library.Publish != nil && library.Publish.Disabled {
		codec["not-for-publication"] = "true"
	}
	if len(rust.DisabledRustdocWarnings) > 0 {
		codec["disabled-rustdoc-warnings"] = strings.Join(rust.DisabledRustdocWarnings, ",")
	}
	if len(rust.DisabledClippyWarnings) > 0 {
		codec["disabled-clippy-warnings"] = strings.Join(rust.DisabledClippyWarnings, ",")
	}
	if rust.TemplateOverride != "" {
		codec["template-override"] = rust.TemplateOverride
	}
	if rust.IncludeGrpcOnlyMethods {
		codec["include-grpc-only-methods"] = "true"
	}
	if rust.PerServiceFeatures {
		codec["per-service-features"] = "true"
	}
	if len(rust.DefaultFeatures) > 0 {
		codec["default-features"] = strings.Join(rust.DefaultFeatures, ",")
	}
	if rust.DetailedTracingAttributes {
		codec["detailed-tracing-attributes"] = "true"
	}
	if rust.HasVeneer {
		codec["has-veneer"] = "true"
	}
	if len(rust.ExtraModules) > 0 {
		codec["extra-modules"] = strings.Join(rust.ExtraModules, ",")
	}
	if rust.RoutingRequired {
		codec["routing-required"] = "true"
	}
	if rust.GenerateSetterSamples {
		codec["generate-setter-samples"] = "true"
	}

	for _, dep := range rust.PackageDependencies {
		codec["package:"+dep.Name] = formatPackageDependency(dep)
	}
	return codec
}

func formatPackageDependency(dep config.RustPackageDependency) string {
	var parts []string
	if dep.Package != "" {
		parts = append(parts, "package="+dep.Package)
	}
	if dep.Source != "" {
		parts = append(parts, "source="+dep.Source)
	}
	if dep.ForceUsed {
		parts = append(parts, "force-used=true")
	}
	if dep.UsedIf != "" {
		parts = append(parts, "used-if="+dep.UsedIf)
	}
	if dep.Feature != "" {
		parts = append(parts, "feature="+dep.Feature)
	}
	return strings.Join(parts, ",")
}
