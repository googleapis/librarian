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
)

// codec represents the configuration for a Swift sidekick Codec.
//
// A sideckick Codec is a package that generates libraries from an `api.API`
// model and some configuration. In the Swift codec, the `Generate()`
// function  creates a `codec` object for each `api.API` that needs to be
// generated. That lends naturally into a single object that carries all the
// information needed to generate the library.
type codec struct {
	GenerationYear string
	// The name of the swift package (e.g. "GoogleCloudSecretManagerV1")
	PackageName  string
	MonorepoRoot string
	// Most libraries are generated from `googleapis`. Rarely, we use protobuf,
	// gapic-showcase, or a different root.
	RootName string
	// Modules have a different directory structure.
	Module       bool
	Model        *api.API
	Dependencies []*Dependency
	// Map of proto package to dependency (e.g. "google.protobuf" -> <dependency>)
	ApiPackages map[string]*Dependency
	// Map of dependency name to dependency (e.g. GoogleCloudGax -> <dependency>)
	DependenciesByName map[string]*Dependency
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
		GenerationYear:     fmt.Sprintf("%04d", year),
		PackageName:        PackageName(model),
		MonorepoRoot:       rel,
		RootName:           "googleapis",
		Model:              model,
		ApiPackages:        map[string]*Dependency{},
		DependenciesByName: map[string]*Dependency{},
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
	}
	for key, definition := range cfg.Codec {
		switch key {
		case "copyright-year":
			result.GenerationYear = definition
		case "package-name-override":
			result.PackageName = definition
		case "root-name":
			result.RootName = definition
		case "module":
			value, err := strconv.ParseBool(definition)
			if err != nil {
				return nil, fmt.Errorf("cannot convert `module` value %q to boolean: %w", definition, err)
			}
			result.Module = value
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
	if dep.Name == c.PackageName {
		return nil, nil
	}
	if ann, ok := c.Model.Codec.(*modelAnnotations); ok {
		ann.DependsOn[dep.Name] = dep
	}
	return dep, nil
}
