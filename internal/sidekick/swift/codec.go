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

package swift

import (
	"fmt"
	"maps"
	"path/filepath"
	"time"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/sidekick/api"
)

const (
	// The name of the Protobuf package that contains the well-known Protobuf types.
	wellKnownProtobufPackage = "google.protobuf"
	// The name of the corresponding Swift package that contains the Swift implementations of these
	// types.
	wellKnownSwiftPackage = "GoogleCloudWKT"
	// The name of the Swift package that contains the pagination helper types.
	paginationSwiftPackage = "GoogleCloudGax"
	// The name of the Swift package that contains the long-running operation helper types.
	lroSwiftPackage = "GoogleCloudGax"
)

// codec represents the configuration for a Swift sidekick Codec.
//
// A sideckick Codec is a package that generates libraries from an `api.API`
// model and some configuration. In the Swift codec, the `Generate()`
// function  creates a `codec` object for each `api.API` that needs to be
// generated. That lends naturally into a single object that carries all the
// information needed to generate the library.
type codec struct {
	// The API this codec is bound to.
	Model *api.API

	// When was the library originally generated.
	//
	// This preserves the copyright year and avoids churn when regenerating the
	// library.
	GenerationYear string

	// LibraryName is the name of the Swift library (e.g. "GoogleCloudSecretManagerV1").
	//
	// Note that GAPIC packages contain a single product (the library), which
	// contains a single target and module with the same names as the library.
	LibraryName string

	// TargetLibraryName is the PascalCase name of the Swift SPM target/library being built
	// (e.g. "GoogleCloudSecretManagerV1", "GoogleCloudStorage", or "GoogleCloudWKT").
	//
	// We need TargetLibraryName to correctly identify self-imports in skipDependency.
	//
	// In librarian.yaml, "modules" refers to individual generator
	// sub-components (such as messages, or convert-swift for wkt/google.type). However,
	// all those generated files are compiled into a single overarching Swift library target
	// named after the PascalCase version of library.Name.
	TargetLibraryName string

	// The name of the Swift package (e.g. "google-cloud-secretmanager-v1").
	PackageName string

	// The package version (e.g. "1.2.3").
	PackageVersion string

	// The location of the monorepo, relative to the current directory.
	//
	// Recall that sidekick only generates clients within a monorepo, so this
	// always makes sense.
	MonorepoRoot string

	// Modules have a different directory structure.
	Module bool

	// The set of dependencies configured for this codec.
	Dependencies []*Dependency

	// Map of proto package to dependency (e.g. "google.protobuf" -> <dependency>)
	ApiPackages map[string]*Dependency

	// Map of dependency name to dependency (e.g. GoogleCloudGax -> <dependency>)
	DependenciesByName map[string]*Dependency

	// If true, the generated code uses a trait (Swift #ifdef-analogs) for each
	// service.
	PerServiceTraits bool

	// If true, these traits are enabled by default.
	DefaultTraits []string

	// If true, bytes need to be serialized and deserialized using the URL-safe
	// base64 alphabet.
	UrlSafeForBytes bool

	// Tracks generated files, considering case-insensitive filesystems.
	//
	// The generated file names are based on message, enum, and service names.
	// When using case-insensitive filesystems (such as APFS on macOS, or NTFS
	// on Windows) this can result in filename clashes when two messages differ
	// only in case, such as `HTTPCheckName` vs. `HttpCheckName`.
	//
	// Furthermore, Swift requires all the files in a package to have different
	// names, even if they are in different subdirectories.
	//
	// The codec uses this map to disambiguate names, the number of clashes for
	// each (lowercased) name are tracked in this hash, and if necessary, the
	// output file is disambiguated by appending `+${Counter}` to the basename.
	GeneratedFiles map[string]int

	// The type of module being generated (e.g. "grpc-client", "convert-swift", "default").
	ModuleType string

	// The name of the private module containing raw stubs (e.g. "StorageControlProtos").
	// Used by convert-swift to generate internal imports and prefix raw types.
	ModulePath string

	// ResponseEncoding sets the `$alt` query parameter value.
	//
	// All RPCS over HTTP sent the `$alt` query parameter. In Google cloud this
	// query parameter controls the format of the response. For most client
	// libraries we use `json;enum-encoding=int`, but for discovery we need to
	// use just `json` as the integer values for enums may not match our values.
	ResponseEncoding string

	// Codec-level overrides for service names.
	// TODO(https://github.com/googleapis/google-cloud-swift/issues/308): Support overriding other symbol types (e.g., messages, enums, oneofs) if needed.
	NameOverrides map[string]string
}

const (
	defaultResponseEncoding   = "json;enum-encoding=int"
	discoveryResponseEncoding = "json"
)

func (c *codec) isGrpc() bool {
	return c.ModuleType == "grpc-client"
}

// ServiceName returns the service name, taking name_overrides into account.
func (c *codec) ServiceName(service *api.Service) string {
	if override, ok := c.NameOverrides[service.ID]; ok {
		return override
	}
	return service.Name
}

func newCodec(model *api.API, library *config.Library, module *config.SwiftModule, outdir string) (*codec, error) {
	year, _, _ := time.Now().Date()
	absOutdir, err := filepath.Abs(outdir)
	if err != nil {
		return nil, err
	}
	// The generator must run at the root of the monorepo, because that is where we keep the `librarian.yaml` file and
	// because all the `outdir` directories are computed relative to that location. So effectively this gets the root
	// of the monorepo.
	absRoot, err := filepath.Abs(".")
	if err != nil {
		return nil, err
	}
	rel, err := filepath.Rel(absOutdir, absRoot)
	if err != nil {
		return nil, err
	}

	generationYear := library.CopyrightYear
	if generationYear == "" {
		generationYear = fmt.Sprintf("%04d", year)
	}

	packageVersion := library.Version
	if packageVersion == "" {
		packageVersion = "0.0.0"
	}

	packageName := ""
	if library.Swift != nil {
		packageName = library.Swift.PackageNameOverride
	}
	if packageName == "" {
		packageName = PackageName(model)
	}
	responseEncoding := defaultResponseEncoding
	if library.SpecificationFormat == config.SpecDiscovery {
		responseEncoding = discoveryResponseEncoding
	}
	result := &codec{
		Model:              model,
		GenerationYear:     generationYear,
		PackageName:        packageName,
		PackageVersion:     packageVersion,
		MonorepoRoot:       rel,
		ApiPackages:        map[string]*Dependency{},
		DependenciesByName: map[string]*Dependency{},
		UrlSafeForBytes:    library.SpecificationFormat == config.SpecDiscovery,
		ResponseEncoding:   responseEncoding,
	}

	swiftCfg := library.Swift
	if swiftCfg != nil {
		for _, d := range swiftCfg.Dependencies {
			dependency := Dependency{SwiftDependency: d}
			result.Dependencies = append(result.Dependencies, &dependency)
			if d.ApiPackage != "" {
				result.ApiPackages[d.ApiPackage] = &dependency
			}
			result.DependenciesByName[d.Name] = &dependency
		}
		result.PerServiceTraits = swiftCfg.PerServiceTraits
		result.DefaultTraits = swiftCfg.DefaultTraits
	}

	if module != nil {
		result.Module = true
		result.ModuleType = module.ModuleType
		result.ModulePath = module.ModulePath
	}

	nameOverrides := make(map[string]string)
	if swiftCfg != nil {
		maps.Copy(nameOverrides, swiftCfg.NameOverrides)
	}
	if module != nil {
		maps.Copy(nameOverrides, module.NameOverrides)
	}
	if len(nameOverrides) > 0 {
		result.NameOverrides = nameOverrides
	}

	libraryName, err := LibraryName(model, swiftCfg)
	if err != nil {
		return nil, err
	}
	result.TargetLibraryName = libraryName

	if !result.Module {
		// Modules cannot have library names, so they should not try to set the value.
		result.LibraryName = libraryName
	}
	return result, nil
}

func (c *codec) addApiPackageDependency(apiName string) (*Dependency, error) {
	dep, ok := c.ApiPackages[apiName]
	if !ok {
		// If there is no explicitly configured dependency for this API, we assume it is
		// provided by the same package that this API is contained within.
		return nil, nil
	}
	return c.addDependency(dep)
}

func (c *codec) addPackageDependency(packageName string) (*Dependency, error) {
	dep, ok := c.DependenciesByName[packageName]
	if !ok {
		return nil, fmt.Errorf("dependency not found for package %q", packageName)
	}
	return c.addDependency(dep)
}

func (c *codec) addDependency(dep *Dependency) (*Dependency, error) {
	if dep == nil {
		return nil, fmt.Errorf("attempting to add nil dependency")
	}
	if c.skipDependency(dep) {
		return nil, nil
	}
	if ann, ok := c.Model.Codec.(*modelAnnotations); ok {
		ann.DependsOn[dep.Name] = dep
	}
	return dep, nil
}

// skipDependency returns true if the dependency should be omitted from imports and dependency lists.
func (c *codec) skipDependency(dep *Dependency) bool {
	if dep == nil {
		return true
	}

	// Do not import the Swift SPM library target that we are currently compiling into.
	if c.TargetLibraryName != "" && dep.Name == c.TargetLibraryName {
		return true
	}

	// During conversion generation, the raw Protobuf stubs module (c.ModulePath, e.g., "StorageControlProtos")
	// is already statically imported via internal import statements in the conversion file template.
	// We skip it dynamically to avoid emitting duplicate "import StorageControlProtos" statements.
	if c.Module && dep.Name == c.ModulePath {
		return true
	}

	return false
}
