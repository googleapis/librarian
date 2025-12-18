// Copyright 2025 Google LLC
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

package rust

import (
	"path"
	"strings"

	"cloud.google.com/go/librarian/internal/config"
	sidekickconfig "cloud.google.com/go/librarian/internal/sidekick/config"
)

func toSidekickConfig(library *config.Library, channel *config.Channel, googleapisDir, discoveryDir, protobufRootDir, protobufSubDir, conformanceDir, showcaseDir string) *sidekickconfig.Config {
	source := map[string]string{}
	specFormat := "protobuf"
	if library.SpecificationFormat != "" {
		specFormat = library.SpecificationFormat
	}
	if specFormat == "discovery" {
		specFormat = "disco"
	}

	if len(library.Roots) == 0 && googleapisDir != "" {
		// Default to googleapis if no roots are specified.
		source["googleapis-root"] = googleapisDir
		source["roots"] = "googleapis"
	} else {
		source["roots"] = strings.Join(library.Roots, ",")
		rootMap := map[string]struct {
			path string
			key  string
		}{
			"googleapis":   {path: googleapisDir, key: "googleapis-root"},
			"discovery":    {path: discoveryDir, key: "discovery-root"},
			"showcase":     {path: showcaseDir, key: "showcase-root"},
			"protobuf-src": {path: path.Join(protobufRootDir, protobufSubDir), key: "protobuf-src-root"},
			"conformance":  {path: conformanceDir, key: "conformance-root"},
		}
		for _, root := range library.Roots {
			if r, ok := rootMap[root]; ok && r.path != "" {
				source[r.key] = r.path
			}
		}
	}

	if library.DescriptionOverride != "" {
		source["description-override"] = library.DescriptionOverride
	}
	if library.Rust != nil {
		if library.Rust.TitleOverride != "" {
			source["title-override"] = library.Rust.TitleOverride
		}
		if len(library.Rust.SkippedIds) > 0 {
			source["skipped-ids"] = strings.Join(library.Rust.SkippedIds, ",")
		}
	}
	sidekickCfg := &sidekickconfig.Config{
		General: sidekickconfig.GeneralConfig{
			Language:            "rust",
			SpecificationFormat: specFormat,
			ServiceConfig:       channel.ServiceConfig,
			SpecificationSource: channel.Path,
		},
		Source: source,
		Codec:  buildCodec(library),
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
		if library.Rust.Discovery != nil {
			pollers := make([]*sidekickconfig.Poller, len(library.Rust.Discovery.Pollers))
			for i, poller := range library.Rust.Discovery.Pollers {
				pollers[i] = &sidekickconfig.Poller{
					Prefix:   poller.Prefix,
					MethodID: poller.MethodID,
				}
			}
			sidekickCfg.Discovery = &sidekickconfig.Discovery{
				OperationID: library.Rust.Discovery.OperationID,
				Pollers:     pollers,
			}
		}
	}
	return sidekickCfg
}

func buildCodec(library *config.Library) map[string]string {
	codec := newLibraryCodec(library)
	if library.Version != "" {
		codec["version"] = library.Version
	}
	if library.ReleaseLevel != "" {
		codec["release-level"] = library.ReleaseLevel
	}
	if library.Rust == nil {
		return codec
	}

	rust := library.Rust
	if rust.ModulePath != "" {
		codec["module-path"] = rust.ModulePath
	}
	if library.SkipPublish {
		codec["not-for-publication"] = "true"
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
	if extraModules := extraModulesFromKeep(library.Keep); len(extraModules) > 0 {
		codec["extra-modules"] = strings.Join(extraModules, ",")
	}
	if rust.RoutingRequired {
		codec["routing-required"] = "true"
	}
	if rust.GenerateSetterSamples != "" {
		codec["generate-setter-samples"] = rust.GenerateSetterSamples
	}
	if rust.GenerateRpcSamples {
		codec["generate-rpc-samples"] = "true"
	}
	if rust.NameOverrides != "" {
		codec["name-overrides"] = rust.NameOverrides
	}
	return codec
}

func newLibraryCodec(library *config.Library) map[string]string {
	codec := make(map[string]string)
	if library.CopyrightYear != "" {
		codec["copyright-year"] = library.CopyrightYear
	}
	if library.Name != "" {
		codec["package-name-override"] = library.Name
	}
	if library.Rust != nil {
		for _, dep := range library.Rust.PackageDependencies {
			codec["package:"+dep.Name] = formatPackageDependency(dep)
		}
		if len(library.Rust.DisabledRustdocWarnings) > 0 {
			codec["disabled-rustdoc-warnings"] = strings.Join(library.Rust.DisabledRustdocWarnings, ",")
		}
		if len(library.Rust.DisabledClippyWarnings) > 0 {
			codec["disabled-clippy-warnings"] = strings.Join(library.Rust.DisabledClippyWarnings, ",")
		}
	}
	return codec
}

// extraModulesFromKeep extracts module names from keep entries that match
// "src/*.rs". For example, "src/errors.rs" becomes "errors".
func extraModulesFromKeep(keep []string) []string {
	var modules []string
	for _, k := range keep {
		if strings.HasPrefix(k, "src/") && strings.HasSuffix(k, ".rs") {
			// Extract module name: "src/errors.rs" -> "errors"
			module := strings.TrimPrefix(k, "src/")
			module = strings.TrimSuffix(module, ".rs")
			modules = append(modules, module)
		}
	}
	return modules
}

func formatPackageDependency(dep *config.RustPackageDependency) string {
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
	if dep.Ignore {
		parts = append(parts, "ignore=true")
	}
	return strings.Join(parts, ",")
}

func moduleToSidekickConfig(library *config.Library, module *config.RustModule, googleapisDir string) *sidekickconfig.Config {
	source := map[string]string{
		"googleapis-root": googleapisDir,
	}
	if len(module.IncludedIds) > 0 {
		source["included-ids"] = strings.Join(module.IncludedIds, ",")
	}
	if len(module.SkippedIds) > 0 {
		source["skipped-ids"] = strings.Join(module.SkippedIds, ",")
	}
	if module.IncludeList != "" {
		source["include-list"] = module.IncludeList
	}
	if module.TitleOverride != "" {
		source["title-override"] = module.TitleOverride
	}

	language := "rust"
	if module.Template == "prost" {
		language = "rust+prost"
	}
	sidekickCfg := &sidekickconfig.Config{
		General: sidekickconfig.GeneralConfig{
			Language:            language,
			SpecificationFormat: "protobuf",
			ServiceConfig:       module.ServiceConfig,
			SpecificationSource: module.Source,
		},
		Source: source,
		Codec:  buildModuleCodec(library, module),
	}
	return sidekickCfg
}

func buildModuleCodec(library *config.Library, module *config.RustModule) map[string]string {
	codec := newLibraryCodec(library)
	if module.GenerateSetterSamples {
		codec["generate-setter-samples"] = "true"
	}
	if module.HasVeneer {
		codec["has-veneer"] = "true"
	}
	if module.IncludeGrpcOnlyMethods {
		codec["include-grpc-only-methods"] = "true"
	}
	if module.ModulePath != "" {
		codec["module-path"] = module.ModulePath
	}
	if module.NameOverrides != "" {
		codec["name-overrides"] = module.NameOverrides
	}
	if module.PostProcessProtos != "" {
		codec["post-process-protos"] = module.PostProcessProtos
	}
	if module.RoutingRequired {
		codec["routing-required"] = "true"
	}
	if module.ExtendGrpcTransport {
		codec["extend-grpc-transport"] = "true"
	}
	if module.Template != "" {
		codec["template-override"] = "templates/" + module.Template
	}
	return codec
}
