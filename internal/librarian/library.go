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

package librarian

import (
	"fmt"
	"maps"
	"strings"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/librarian/golang"
	"github.com/googleapis/librarian/internal/librarian/java"
	"github.com/googleapis/librarian/internal/librarian/python"
	"github.com/googleapis/librarian/internal/librarian/rust"
)

// fillDefaults populates empty library fields from the provided defaults.
func fillDefaults(lib *config.Library, d *config.Default) *config.Library {
	if d == nil {
		return lib
	}
	if d.Keep != nil {
		lib.Keep = append(lib.Keep, d.Keep...)
	}
	if lib.Output == "" {
		lib.Output = d.Output
	}
	if d.Rust != nil {
		return fillRust(lib, d)
	}
	if d.Dart != nil {
		return fillDart(lib, d)
	}
	if d.Python != nil {
		return fillPython(lib, d)
	}
	if d.Swift != nil {
		return fillSwift(lib, d)
	}
	return lib
}

// fillRust populates empty Rust-specific fields in lib from the provided default.
func fillRust(lib *config.Library, d *config.Default) *config.Library {
	if lib.Rust == nil {
		lib.Rust = &config.RustCrate{}
	}
	lib.Rust.PackageDependencies = mergePackageDependencies(
		d.Rust.PackageDependencies,
		lib.Rust.PackageDependencies,
	)
	if len(lib.Rust.DisabledRustdocWarnings) == 0 {
		lib.Rust.DisabledRustdocWarnings = d.Rust.DisabledRustdocWarnings
	}
	if lib.Rust.DetailedTracingAttributes == nil {
		lib.Rust.DetailedTracingAttributes = d.Rust.DetailedTracingAttributes
	}
	if lib.Rust.ResourceNameHeuristic == nil {
		lib.Rust.ResourceNameHeuristic = d.Rust.ResourceNameHeuristic
	}
	if lib.Rust.GenerateSetterSamples == "" {
		lib.Rust.GenerateSetterSamples = d.Rust.GenerateSetterSamples
	}
	if lib.Rust.GenerateRpcSamples == "" {
		lib.Rust.GenerateRpcSamples = d.Rust.GenerateRpcSamples
	}
	for _, mod := range lib.Rust.Modules {
		if mod.GenerateSetterSamples == "" {
			mod.GenerateSetterSamples = lib.Rust.GenerateSetterSamples
		}
		if mod.GenerateRpcSamples == "" {
			mod.GenerateRpcSamples = lib.Rust.GenerateRpcSamples
		}
	}
	return lib
}

func fillDart(lib *config.Library, d *config.Default) *config.Library {
	if lib.Version == "" {
		lib.Version = d.Dart.Version
	}
	if lib.Dart == nil {
		lib.Dart = &config.DartPackage{}
	}
	if lib.Dart.APIKeysEnvironmentVariables == "" {
		lib.Dart.APIKeysEnvironmentVariables = d.Dart.APIKeysEnvironmentVariables
	}
	if lib.Dart.IssueTrackerURL == "" {
		lib.Dart.IssueTrackerURL = d.Dart.IssueTrackerURL
	}
	lib.Dart.Packages = mergeMaps(lib.Dart.Packages, d.Dart.Packages)
	lib.Dart.Prefixes = mergeMaps(lib.Dart.Prefixes, d.Dart.Prefixes)
	lib.Dart.Protos = mergeMaps(lib.Dart.Protos, d.Dart.Protos)
	lib.Dart.Dependencies = mergeDartDependencies(lib.Dart.Dependencies, d.Dart.Dependencies)
	return lib
}

// fillPython populates empty Python-specific fields in lib from the provided
// default.
func fillPython(lib *config.Library, d *config.Default) *config.Library {
	if lib.Python == nil {
		lib.Python = &config.PythonPackage{}
	}
	lib.Python.CommonGAPICPaths = append(d.Python.CommonGAPICPaths, lib.Python.CommonGAPICPaths...)
	if lib.Python.LibraryType == "" {
		lib.Python.LibraryType = d.Python.LibraryType
	}
	return lib
}

// fillSwift populates empty Swift-specific fields in lib from the provided default.
func fillSwift(lib *config.Library, d *config.Default) *config.Library {
	if lib.Swift == nil {
		lib.Swift = &config.SwiftPackage{}
	}
	lib.Swift.Dependencies = mergeSwiftDependencies(
		d.Swift.Dependencies,
		lib.Swift.Dependencies,
	)
	return lib
}

// mergeSwiftDependencies merges library dependencies with default dependencies,
// with library dependencies taking precedence for duplicates.
func mergeSwiftDependencies(defaults, lib []config.SwiftDependency) []config.SwiftDependency {
	seen := make(map[string]bool)
	var result []config.SwiftDependency
	for _, dep := range lib {
		seen[dep.Name] = true
		result = append(result, dep)
	}
	for _, dep := range defaults {
		if seen[dep.Name] {
			continue
		}
		result = append(result, dep)
	}
	return result
}

// mergeDartDependencies merges library dependencies with default dependencies.
// Duplicate dependencies in defaults will be ignored.
func mergeDartDependencies(libDeps, defaultDeps string) string {
	seen := make(map[string]bool)
	var deps []string
	for _, dep := range strings.Split(libDeps, ",") {
		dep = strings.TrimSpace(dep)
		if dep == "" {
			continue
		}
		seen[dep] = true
		deps = append(deps, dep)
	}
	for _, dep := range strings.Split(defaultDeps, ",") {
		dep = strings.TrimSpace(dep)
		if dep == "" || seen[dep] {
			continue
		}
		deps = append(deps, dep)
	}
	return strings.Join(deps, ",")
}

// mergePackageDependencies merges default and library package dependencies,
// with library dependencies taking precedence for duplicates.
func mergePackageDependencies(defaults, lib []*config.RustPackageDependency) []*config.RustPackageDependency {
	seen := make(map[string]bool)
	var result []*config.RustPackageDependency
	for _, dep := range lib {
		seen[dep.Name] = true
		result = append(result, dep)
	}
	for _, dep := range defaults {
		if seen[dep.Name] {
			continue
		}
		copied := *dep
		result = append(result, &copied)
	}
	return result
}

// isVeneer reports whether the library has handwritten code wrapping generated
// code.
func isVeneer(language string, lib *config.Library) bool {
	switch language {
	case config.LanguageRust:
		return rust.IsVeneer(lib)
	case config.LanguageNodejs:
		return lib.Output != "" && len(lib.APIs) == 0
	default:
		return false
	}
}

// libraryOutput returns the output path for a library. If the library has an
// explicit output path, it returns that. Otherwise, it computes the default
// output path based on the api path and default configuration.
func libraryOutput(language string, lib *config.Library, defaults *config.Default) string {
	if lib.Output != "" {
		return lib.Output
	}
	if isVeneer(language, lib) {
		// Veneers require explicit output, so return empty if not set.
		return ""
	}
	apiPath := deriveAPIPath(language, lib.Name)
	if len(lib.APIs) > 0 && lib.APIs[0].Path != "" {
		apiPath = lib.APIs[0].Path
	}
	defaultOut := ""
	if defaults != nil {
		defaultOut = defaults.Output
	}
	return defaultOutput(language, lib.Name, apiPath, defaultOut)
}

// applyDefaults applies language-specific derivations and fills defaults.
func applyDefaults(language string, lib *config.Library, defaults *config.Default) (*config.Library, error) {
	if !isVeneer(language, lib) {
		if len(lib.APIs) == 0 && canDeriveAPIPath(language) {
			// Do not derive API path for Go because the library name
			// doesn't contain relevant info.
			lib.APIs = append(lib.APIs, &config.API{})
		}
		for _, api := range lib.APIs {
			if api.Path == "" {
				api.Path = deriveAPIPath(language, lib.Name)
			}
		}
	}
	if lib.Output == "" {
		if isVeneer(language, lib) {
			return nil, fmt.Errorf("veneer %q requires an explicit output path", lib.Name)
		}
		var apiPath string
		if len(lib.APIs) > 0 {
			apiPath = lib.APIs[0].Path
		}
		lib.Output = defaultOutput(language, lib.Name, apiPath, defaults.Output)
	}
	return fillLibraryDefaults(language, fillDefaults(lib, defaults))
}

// canDeriveAPIPath reports whether the language's library name contains enough information to
// derive the API path.
func canDeriveAPIPath(language string) bool {
	switch language {
	case config.LanguageGo, config.LanguagePython:
		return false
	default:
		return true
	}
}

// mergeMaps merges key-values of src and dst maps.
// When a key in src is already present in dst, the value in dst will NOT be overwritten
// by the value associated with the key in src.
func mergeMaps(dst, src map[string]string) map[string]string {
	res := make(map[string]string)
	maps.Copy(res, src)
	if dst != nil {
		maps.Copy(res, dst)
	}
	return res
}

// fillLibraryDefaults populates language-specific default values for the library.
func fillLibraryDefaults(language string, lib *config.Library) (*config.Library, error) {
	switch language {
	case config.LanguageGo:
		return golang.Fill(lib)
	case config.LanguageJava:
		return java.Fill(lib)
	case config.LanguagePython:
		return python.Fill(lib)
	default:
		return lib, nil
	}
}

// FindLibrary returns a library with the given name from the config.
func FindLibrary(c *config.Config, name string) (*config.Library, error) {
	if c.Libraries == nil {
		return nil, fmt.Errorf("%w: %q", ErrLibraryNotFound, name)
	}
	for _, library := range c.Libraries {
		if library.Name == name {
			return library, nil
		}
	}
	return nil, fmt.Errorf("%w: %q", ErrLibraryNotFound, name)
}

// ResolvePreview returns a library where fields from lib.Preview override
// those in the base lib, if set. If lib.Preview is not set or lib itself is nil
// this returns nil.
func ResolvePreview(lib *config.Library, language string) *config.Library {
	if lib == nil || lib.Preview == nil {
		return nil
	}
	res := *lib
	p := lib.Preview
	if p.Name != "" {
		res.Name = p.Name
	}
	if p.Version != "" {
		res.Version = p.Version
	}
	if p.APIs != nil {
		res.APIs = p.APIs
	}
	if p.CopyrightYear != "" {
		res.CopyrightYear = p.CopyrightYear
	}
	if p.DescriptionOverride != "" {
		res.DescriptionOverride = p.DescriptionOverride
	}
	if p.Keep != nil {
		res.Keep = p.Keep
	}
	if p.Output != "" {
		res.Output = p.Output
	}
	if p.Roots != nil {
		res.Roots = p.Roots
	}
	if p.SkipGenerate {
		res.SkipGenerate = p.SkipGenerate
	}
	if p.SkipRelease {
		res.SkipRelease = p.SkipRelease
	}
	if p.SpecificationFormat != "" {
		res.SpecificationFormat = p.SpecificationFormat
	}
	switch language {
	case config.LanguageDotnet:
		res.Dotnet = mergeDotnet(res.Dotnet, p.Dotnet)
	case config.LanguageDart:
		res.Dart = mergeDart(res.Dart, p.Dart)
	case config.LanguageGo:
		res.Go = mergeGo(res.Go, p.Go)
	case config.LanguageJava:
		res.Java = mergeJava(res.Java, p.Java)
	case config.LanguageNodejs:
		res.Nodejs = mergeNodejs(res.Nodejs, p.Nodejs)
	case config.LanguagePython:
		res.Python = mergePython(res.Python, p.Python)
	case config.LanguageRust:
		res.Rust = mergeRust(res.Rust, p.Rust)
	}
	res.Preview = nil
	return &res
}

func mergeDotnet(dst, src *config.DotnetPackage) *config.DotnetPackage {
	if src == nil {
		return dst
	}
	if dst == nil {
		return src
	}
	res := *dst
	if src.AdditionalServiceDescriptors != nil {
		res.AdditionalServiceDescriptors = src.AdditionalServiceDescriptors
	}
	res.Csproj = mergeDotnetCsproj(res.Csproj, src.Csproj)
	if src.Dependencies != nil {
		res.Dependencies = src.Dependencies
	}
	if src.Generator != "" {
		res.Generator = src.Generator
	}
	if src.PackageGroup != nil {
		res.PackageGroup = src.PackageGroup
	}
	if src.Postgeneration != nil {
		res.Postgeneration = src.Postgeneration
	}
	if src.Pregeneration != nil {
		res.Pregeneration = src.Pregeneration
	}
	return &res
}

func mergeDotnetCsproj(dst, src *config.DotnetCsproj) *config.DotnetCsproj {
	if src == nil {
		return dst
	}
	if dst == nil {
		return src
	}
	res := *dst
	res.Snippets = mergeDotnetCsprojSnippets(res.Snippets, src.Snippets)
	res.IntegrationTests = mergeDotnetCsprojSnippets(res.IntegrationTests, src.IntegrationTests)
	return &res
}

func mergeDotnetCsprojSnippets(dst, src *config.DotnetCsprojSnippets) *config.DotnetCsprojSnippets {
	if src == nil {
		return dst
	}
	if dst == nil {
		return src
	}
	res := *dst
	if src.EmbeddedResources != nil {
		res.EmbeddedResources = src.EmbeddedResources
	}
	return &res
}

func mergeDart(dst, src *config.DartPackage) *config.DartPackage {
	if src == nil {
		return dst
	}
	if dst == nil {
		return src
	}
	res := *dst
	if src.APIKeysEnvironmentVariables != "" {
		res.APIKeysEnvironmentVariables = src.APIKeysEnvironmentVariables
	}
	if src.Dependencies != "" {
		res.Dependencies = src.Dependencies
	}
	if src.DevDependencies != "" {
		res.DevDependencies = src.DevDependencies
	}
	if src.ExtraImports != "" {
		res.ExtraImports = src.ExtraImports
	}
	if src.IncludeList != nil {
		res.IncludeList = src.IncludeList
	}
	if src.IssueTrackerURL != "" {
		res.IssueTrackerURL = src.IssueTrackerURL
	}
	if src.LibraryPathOverride != "" {
		res.LibraryPathOverride = src.LibraryPathOverride
	}
	if src.NameOverride != "" {
		res.NameOverride = src.NameOverride
	}
	if src.Packages != nil {
		res.Packages = src.Packages
	}
	if src.PartFile != "" {
		res.PartFile = src.PartFile
	}
	if src.Prefixes != nil {
		res.Prefixes = src.Prefixes
	}
	if src.Protos != nil {
		res.Protos = src.Protos
	}
	if src.ReadmeAfterTitleText != "" {
		res.ReadmeAfterTitleText = src.ReadmeAfterTitleText
	}
	if src.ReadmeQuickstartText != "" {
		res.ReadmeQuickstartText = src.ReadmeQuickstartText
	}
	if src.RepositoryURL != "" {
		res.RepositoryURL = src.RepositoryURL
	}
	if src.TitleOverride != "" {
		res.TitleOverride = src.TitleOverride
	}
	if src.Version != "" {
		res.Version = src.Version
	}
	return &res
}

func mergeGo(dst, src *config.GoModule) *config.GoModule {
	if src == nil {
		return dst
	}
	if dst == nil {
		return src
	}
	res := *dst
	if src.DeleteGenerationOutputPaths != nil {
		res.DeleteGenerationOutputPaths = src.DeleteGenerationOutputPaths
	}
	if src.GoAPIs != nil {
		res.GoAPIs = src.GoAPIs
	}
	if src.ModulePathVersion != "" {
		res.ModulePathVersion = src.ModulePathVersion
	}
	if src.NestedModule != "" {
		res.NestedModule = src.NestedModule
	}
	return &res
}

func mergeJava(dst, src *config.JavaModule) *config.JavaModule {
	if src == nil {
		return dst
	}
	if dst == nil {
		return src
	}
	res := *dst
	if src.APIIDOverride != "" {
		res.APIIDOverride = src.APIIDOverride
	}
	if src.APIReference != "" {
		res.APIReference = src.APIReference
	}
	if src.APIDescriptionOverride != "" {
		res.APIDescriptionOverride = src.APIDescriptionOverride
	}
	if src.ClientDocumentationOverride != "" {
		res.ClientDocumentationOverride = src.ClientDocumentationOverride
	}
	if src.CloudAPI != nil {
		res.CloudAPI = src.CloudAPI
	}
	if src.CodeownerTeam != "" {
		res.CodeownerTeam = src.CodeownerTeam
	}
	if src.DistributionNameOverride != "" {
		res.DistributionNameOverride = src.DistributionNameOverride
	}
	if src.ExcludedDependencies != "" {
		res.ExcludedDependencies = src.ExcludedDependencies
	}
	if src.ExcludedPOMs != "" {
		res.ExcludedPOMs = src.ExcludedPOMs
	}
	if src.ExtraVersionedModules != "" {
		res.ExtraVersionedModules = src.ExtraVersionedModules
	}
	if src.GroupID != "" {
		res.GroupID = src.GroupID
	}
	if src.IssueTrackerOverride != "" {
		res.IssueTrackerOverride = src.IssueTrackerOverride
	}
	if src.LibrariesBOMVersion != "" {
		res.LibrariesBOMVersion = src.LibrariesBOMVersion
	}
	if src.LibraryTypeOverride != "" {
		res.LibraryTypeOverride = src.LibraryTypeOverride
	}
	if src.MinJavaVersion != 0 {
		res.MinJavaVersion = src.MinJavaVersion
	}
	if src.NamePrettyOverride != "" {
		res.NamePrettyOverride = src.NamePrettyOverride
	}
	if src.JavaAPIs != nil {
		res.JavaAPIs = src.JavaAPIs
	}
	if src.ProductDocumentationOverride != "" {
		res.ProductDocumentationOverride = src.ProductDocumentationOverride
	}
	if src.RecommendedPackage != "" {
		res.RecommendedPackage = src.RecommendedPackage
	}
	if src.BillingRequired != nil {
		res.BillingRequired = src.BillingRequired
	}
	if src.RestDocumentation != "" {
		res.RestDocumentation = src.RestDocumentation
	}
	if src.RpcDocumentation != "" {
		res.RpcDocumentation = src.RpcDocumentation
	}
	return &res
}

func mergeNodejs(dst, src *config.NodejsPackage) *config.NodejsPackage {
	if src == nil {
		return dst
	}
	if dst == nil {
		return src
	}
	res := *dst
	if src.BundleConfig != "" {
		res.BundleConfig = src.BundleConfig
	}
	if src.Dependencies != nil {
		res.Dependencies = src.Dependencies
	}
	if src.ExtraProtocParameters != nil {
		res.ExtraProtocParameters = src.ExtraProtocParameters
	}
	if src.HandwrittenLayer {
		res.HandwrittenLayer = src.HandwrittenLayer
	}
	if src.MainService != "" {
		res.MainService = src.MainService
	}
	if src.Mixins != "" {
		res.Mixins = src.Mixins
	}
	if src.PackageName != "" {
		res.PackageName = src.PackageName
	}
	return &res
}

func mergePython(dst, src *config.PythonPackage) *config.PythonPackage {
	if src == nil {
		return dst
	}
	if dst == nil {
		return src
	}
	res := *dst
	if src.CommonGAPICPaths != nil {
		res.CommonGAPICPaths = src.CommonGAPICPaths
	}
	if src.LibraryType != "" {
		res.LibraryType = src.LibraryType
	}
	if src.OptArgsByAPI != nil {
		res.OptArgsByAPI = src.OptArgsByAPI
	}
	if src.ProtoOnlyAPIs != nil {
		res.ProtoOnlyAPIs = src.ProtoOnlyAPIs
	}
	if src.NamePrettyOverride != "" {
		res.NamePrettyOverride = src.NamePrettyOverride
	}
	if src.ProductDocumentationOverride != "" {
		res.ProductDocumentationOverride = src.ProductDocumentationOverride
	}
	if src.APIShortnameOverride != "" {
		res.APIShortnameOverride = src.APIShortnameOverride
	}
	if src.APIIDOverride != "" {
		res.APIIDOverride = src.APIIDOverride
	}
	if src.ClientDocumentationOverride != "" {
		res.ClientDocumentationOverride = src.ClientDocumentationOverride
	}
	if src.IssueTrackerOverride != "" {
		res.IssueTrackerOverride = src.IssueTrackerOverride
	}
	if src.MetadataNameOverride != "" {
		res.MetadataNameOverride = src.MetadataNameOverride
	}
	if src.DefaultVersion != "" {
		res.DefaultVersion = src.DefaultVersion
	}
	if src.SkipReadmeCopy {
		res.SkipReadmeCopy = src.SkipReadmeCopy
	}
	return &res
}

func mergeRust(dst, src *config.RustCrate) *config.RustCrate {
	if src == nil {
		return dst
	}
	if dst == nil {
		return src
	}
	res := *dst
	if src.PackageDependencies != nil {
		res.PackageDependencies = src.PackageDependencies
	}
	if src.DisabledRustdocWarnings != nil {
		res.DisabledRustdocWarnings = src.DisabledRustdocWarnings
	}
	if src.GenerateSetterSamples != "" {
		res.GenerateSetterSamples = src.GenerateSetterSamples
	}
	if src.GenerateRpcSamples != "" {
		res.GenerateRpcSamples = src.GenerateRpcSamples
	}
	if src.DetailedTracingAttributes != nil {
		res.DetailedTracingAttributes = src.DetailedTracingAttributes
	}
	if src.ResourceNameHeuristic != nil {
		res.ResourceNameHeuristic = src.ResourceNameHeuristic
	}
	if src.Modules != nil {
		res.Modules = src.Modules
	}
	if src.PerServiceFeatures {
		res.PerServiceFeatures = src.PerServiceFeatures
	}
	if src.ModulePath != "" {
		res.ModulePath = src.ModulePath
	}
	if src.TemplateOverride != "" {
		res.TemplateOverride = src.TemplateOverride
	}
	if src.PackageNameOverride != "" {
		res.PackageNameOverride = src.PackageNameOverride
	}
	if src.RootName != "" {
		res.RootName = src.RootName
	}
	if src.DefaultFeatures != nil {
		res.DefaultFeatures = src.DefaultFeatures
	}
	if src.IncludeList != nil {
		res.IncludeList = src.IncludeList
	}
	if src.IncludedIds != nil {
		res.IncludedIds = src.IncludedIds
	}
	if src.SkippedIds != nil {
		res.SkippedIds = src.SkippedIds
	}
	if src.DisabledClippyWarnings != nil {
		res.DisabledClippyWarnings = src.DisabledClippyWarnings
	}
	if src.HasVeneer {
		res.HasVeneer = src.HasVeneer
	}
	if src.RoutingRequired {
		res.RoutingRequired = src.RoutingRequired
	}
	if src.IncludeGrpcOnlyMethods {
		res.IncludeGrpcOnlyMethods = src.IncludeGrpcOnlyMethods
	}
	if src.IncludeStreamingMethods {
		res.IncludeStreamingMethods = src.IncludeStreamingMethods
	}
	if src.PostProcessProtos != "" {
		res.PostProcessProtos = src.PostProcessProtos
	}
	if src.DocumentationOverrides != nil {
		res.DocumentationOverrides = src.DocumentationOverrides
	}
	if src.PaginationOverrides != nil {
		res.PaginationOverrides = src.PaginationOverrides
	}
	if src.NameOverrides != "" {
		res.NameOverrides = src.NameOverrides
	}
	res.Discovery = mergeRustDiscovery(res.Discovery, src.Discovery)
	if src.QuickstartServiceOverride != "" {
		res.QuickstartServiceOverride = src.QuickstartServiceOverride
	}
	return &res
}

func mergeRustDiscovery(dst, src *config.RustDiscovery) *config.RustDiscovery {
	if src == nil {
		return dst
	}
	if dst == nil {
		return src
	}
	res := *dst
	if src.OperationID != "" {
		res.OperationID = src.OperationID
	}
	if src.Pollers != nil {
		res.Pollers = src.Pollers
	}
	return &res
}
