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
	"fmt"
	"strings"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/serviceconfig"
	"github.com/googleapis/librarian/internal/sidekick/api"
	sidekickconfig "github.com/googleapis/librarian/internal/sidekick/config"
	"github.com/googleapis/librarian/internal/sidekick/parser"
	"github.com/googleapis/librarian/internal/sidekick/source"
)

func libraryToModelConfig(library *config.Library, ch *config.API, sources *source.Sources) (*parser.ModelConfig, error) {
	specFormat := config.SpecProtobuf
	if library.SpecificationFormat != "" {
		specFormat = library.SpecificationFormat
	}

	src := addLibraryRoots(library, sources)
	root := sources.Googleapis
	if ch.Path == "schema/google/showcase/v1beta1" {
		root = sources.Showcase
	}
	svcConfig, err := serviceconfig.Find(root, ch.Path, serviceconfig.LangRust)
	if err != nil {
		return nil, err
	}

	var specSource string
	switch specFormat {
	case config.SpecDiscovery:
		specSource = svcConfig.Discovery
	case config.SpecOpenAPI:
		specSource = svcConfig.OpenAPI
	default:
		specSource = ch.Path
	}

	modelCfg := &parser.ModelConfig{
		Language:            "rust",
		SpecificationFormat: specFormat,
		SpecificationSource: specSource,
		Source:              src,
		ServiceConfig:       svcConfig.ServiceConfig,
		Codec:               buildCodec(library),
		Override: api.ModelOverride{
			Description: library.DescriptionOverride,
			Title:       svcConfig.Title,
		},
	}

	if library.Rust != nil {
		if len(library.Rust.SkippedIds) > 0 {
			modelCfg.Override.SkippedIDs = library.Rust.SkippedIds
		}
		if len(library.Rust.DocumentationOverrides) > 0 {
			modelCfg.CommentOverrides = make([]api.DocumentationOverride, len(library.Rust.DocumentationOverrides))
			for i, override := range library.Rust.DocumentationOverrides {
				modelCfg.CommentOverrides[i] = api.DocumentationOverride{
					ID:      override.ID,
					Match:   override.Match,
					Replace: override.Replace,
				}
			}
		}
		if len(library.Rust.PaginationOverrides) > 0 {
			modelCfg.PaginationOverrides = make([]api.PaginationOverride, len(library.Rust.PaginationOverrides))
			for i, override := range library.Rust.PaginationOverrides {
				modelCfg.PaginationOverrides[i] = api.PaginationOverride{
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
			modelCfg.Discovery = &sidekickconfig.Discovery{
				OperationID: library.Rust.Discovery.OperationID,
				Pollers:     pollers,
			}
		}
	}
	return modelCfg, nil
}

func buildCodec(library *config.Library) map[string]string {
	codec := newLibraryCodec(library)
	if library.Version != "" {
		codec["version"] = library.Version
	}
	if library.ReleaseLevel != "" {
		codec["release-level"] = library.ReleaseLevel
	}
	if library.SkipRelease {
		codec["not-for-publication"] = "true"
	}
	if extraModules := extraModulesFromKeep(library.Keep); len(extraModules) > 0 {
		codec["extra-modules"] = strings.Join(extraModules, ",")
	}
	if library.Rust == nil {
		return codec
	}
	rust := library.Rust
	if rust.ModulePath != "" {
		codec["module-path"] = rust.ModulePath
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
	if rust.RoutingRequired {
		codec["routing-required"] = "true"
	}
	if rust.GenerateSetterSamples != "" {
		codec["generate-setter-samples"] = rust.GenerateSetterSamples
	}
	if rust.GenerateRpcSamples != "" {
		codec["generate-rpc-samples"] = rust.GenerateRpcSamples
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

func moduleToModelConfig(library *config.Library, module *config.RustModule, sources *source.Sources) (*parser.ModelConfig, error) {
	src := addLibraryRoots(library, sources)
	var title string
	if module.APIPath != "" && src["roots"] == "googleapis" {
		api, err := serviceconfig.Find(sources.Googleapis, module.APIPath, serviceconfig.LangRust)
		if err != nil {
			return nil, fmt.Errorf("failed to find service config for %q: %w", module.APIPath, err)
		}
		if api != nil && api.Title != "" {
			title = api.Title
		}
	}

	language := "rust"
	if module.Language != "" {
		language = module.Language
	} else if module.Template == "prost" {
		language = "rust+prost"
	}

	specificationFormat := config.SpecProtobuf
	if module.SpecificationFormat != "" {
		specificationFormat = module.SpecificationFormat
	}
	if module.IncludeList != "" {
		src["include-list"] = module.IncludeList
	}
	modelCfg := &parser.ModelConfig{
		Language:            language,
		SpecificationFormat: specificationFormat,
		ServiceConfig:       module.ServiceConfig,
		SpecificationSource: module.APIPath,
		Source:              src,
		Codec:               buildModuleCodec(library, module),
		Override: api.ModelOverride{
			Title:       title,
			IncludedIDs: module.IncludedIds,
			SkippedIDs:  module.SkippedIds,
		},
	}
	if len(module.DocumentationOverrides) > 0 {
		modelCfg.CommentOverrides = make([]api.DocumentationOverride, len(module.DocumentationOverrides))
		for i, override := range module.DocumentationOverrides {
			modelCfg.CommentOverrides[i] = api.DocumentationOverride{
				ID:      override.ID,
				Match:   override.Match,
				Replace: override.Replace,
			}
		}
	}
	if len(library.Rust.PaginationOverrides) > 0 {
		modelCfg.PaginationOverrides = make([]api.PaginationOverride, len(library.Rust.PaginationOverrides))
		for i, override := range library.Rust.PaginationOverrides {
			modelCfg.PaginationOverrides[i] = api.PaginationOverride{
				ID:        override.ID,
				ItemField: override.ItemField,
			}
		}
	}
	return modelCfg, nil
}

func buildModuleCodec(library *config.Library, module *config.RustModule) map[string]string {
	codec := newLibraryCodec(library)
	if module.GenerateSetterSamples != "" {
		codec["generate-setter-samples"] = module.GenerateSetterSamples
	}
	if module.GenerateRpcSamples != "" {
		codec["generate-rpc-samples"] = module.GenerateRpcSamples
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
	if module.DisabledRustdocWarnings != nil {
		codec["disabled-rustdoc-warnings"] = strings.Join(module.DisabledRustdocWarnings, ",")
	}
	if module.RootName != "" {
		codec["root-name"] = module.RootName
	}
	if module.InternalBuilders {
		codec["internal-builders"] = "true"
	}
	return codec
}

// TODO(https://github.com/googleapis/librarian/issues/3863): remove this function once we removed sidekick config.
func addLibraryRoots(library *config.Library, sources *source.Sources) map[string]string {
	src := make(map[string]string)
	if library.Rust == nil {
		library.Rust = &config.RustCrate{}
	}

	if len(library.Roots) == 0 && sources.Googleapis != "" {
		// Default to googleapis if no roots are specified.
		src["googleapis-root"] = sources.Googleapis
		src["roots"] = "googleapis"
	} else {
		src["roots"] = strings.Join(library.Roots, ",")
		rootMap := map[string]struct {
			path string
			key  string
		}{
			"googleapis":   {path: sources.Googleapis, key: "googleapis-root"},
			"discovery":    {path: sources.Discovery, key: "discovery-root"},
			"showcase":     {path: sources.Showcase, key: "showcase-root"},
			"protobuf-src": {path: sources.ProtobufSrc, key: "protobuf-src-root"},
			"conformance":  {path: sources.Conformance, key: "conformance-root"},
		}
		for _, root := range library.Roots {
			if r, ok := rootMap[root]; ok && r.path != "" {
				src[r.key] = r.path
			}
		}
	}

	return src
}
