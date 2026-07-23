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
	"path/filepath"
	"strconv"
	"time"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/sidekick/api"
	"github.com/googleapis/librarian/internal/sidekick/parser"
)

const (
	// The name of the Protobuf package that contains the well-known Protobuf types.
	wellKnownProtobufPackage = "google.protobuf"
	// The name of the corresponding Swift package that contains the Swift implementations of these
	// types.
	wellKnownSwiftPackage = "GoogleCloudWkt"
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

	// The name of the Swift package (e.g. "google-cloud-secretmanager-v1").
	//
	// Note that GAPIC packages contain a single product (the library), which
	// contains a single target and module with the same names as the library.
	PackageName string

	// The package version (e.g. "1.2.3").
	PackageVersion string

	// The release level (e.g. "preview" or "stable").
	ReleaseLevel string

	// The location of the monorepo, relative to the current directory.
	//
	// Recall that sidekick only generates clients within a monorepo, so this
	// always makes sense.
	MonorepoRoot string

	// Most libraries are generated from `googleapis`. Rarely, we use protobuf,
	// gapic-showcase, or a different root.
	RootName string

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

	// The name of the private module containing raw stubs (e.g. "StorageControlProtos").
	// Used by convert-swift to generate internal imports and prefix raw types.
	ModulePath string
}

func newCodec(model *api.API, cfg *parser.ModelConfig, swiftCfg *config.SwiftPackage, outdir string) (*codec, error) {
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
	result := &codec{
		Model:              model,
		GenerationYear:     fmt.Sprintf("%04d", year),
		LibraryName:        LibraryName(model, swiftCfg),
		PackageName:        PackageName(model),
		PackageVersion:     "0.0.0",
		ReleaseLevel:       "preview",
		MonorepoRoot:       rel,
		RootName:           "googleapis",
		ApiPackages:        map[string]*Dependency{},
		DependenciesByName: map[string]*Dependency{},
		UrlSafeForBytes:    cfg.SpecificationFormat == config.SpecDiscovery,
	}
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
	for key, definition := range cfg.Codec {
		switch key {
		case "copyright-year":
			result.GenerationYear = definition
		case "version":
			result.PackageVersion = definition
		case "release-level":
			result.ReleaseLevel = definition
		case "package-name-override":
			result.LibraryName = definition
		case "root-name":
			result.RootName = definition
		case "module":
			value, err := strconv.ParseBool(definition)
			if err != nil {
				return nil, fmt.Errorf("cannot convert `module` value %q to boolean: %w", definition, err)
			}
			result.Module = value
		case "module-path":
			result.ModulePath = definition
		default:
			// Ignore other options.
		}
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
	// Skip including self as a dependency
	if dep.Name == c.LibraryName {
		return nil, nil
	}
	if ann, ok := c.Model.Codec.(*modelAnnotations); ok {
		ann.DependsOn[dep.Name] = dep
	}
	return dep, nil
}
