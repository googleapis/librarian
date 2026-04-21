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
	"cmp"
	"fmt"
	"maps"
	"slices"

	"github.com/googleapis/librarian/internal/license"
)

type modelAnnotations struct {
	CopyrightYear  string
	BoilerPlate    []string
	PackageName    string
	MonorepoRoot   string
	DependsOn      map[string]*Dependency
	WktPackage     string
	ServiceImports []string
	MessageImports []string
}

// HasDependencies returns true if the package has dependencies on other packages.
//
// The mustache templates use this to omit code that goes unused when the package has no
// dependencies.
func (ann *modelAnnotations) HasDependencies() bool {
	return len(ann.DependsOn) != 0
}

// Dependencies returns the list of dependencies for this package.
func (ann *modelAnnotations) Dependencies() []*Dependency {
	deps := slices.Collect(maps.Values(ann.DependsOn))
	slices.SortFunc(deps, func(a, b *Dependency) int { return cmp.Compare(a.Name, b.Name) })
	return deps
}

// HasMessageImports returns true if the package needs imports for the methods.
//
// The mustache templates use this to format the generated code.
func (ann *modelAnnotations) HasMessageImports() bool {
	return len(ann.MessageImports) != 0
}

func (c *codec) annotateModel() error {
	annotations := &modelAnnotations{
		CopyrightYear: c.GenerationYear,
		BoilerPlate:   license.HeaderBulk(),
		PackageName:   c.PackageName,
		MonorepoRoot:  c.MonorepoRoot,
		DependsOn:     map[string]*Dependency{},
		WktPackage:    wellKnownSwiftPackage,
	}
	c.Model.Codec = annotations
	for _, message := range c.Model.Messages {
		if err := c.annotateMessage(message, annotations); err != nil {
			return err
		}
	}
	for _, enum := range c.Model.Enums {
		if err := c.annotateEnum(enum, annotations); err != nil {
			return err
		}
	}
	// If there is at least one message, the generated library depends on `GoogleCloudWkt` because
	// the generated messages must conform to the `GoogleCloudWkt._AnyPackable` protocol.
	if len(c.Model.Messages) != 0 {
		if dep, ok := c.ApiPackages[wellKnownPackage]; ok {
			dep.Required = true
			annotations.WktPackage = dep.Name
		} else {
			return fmt.Errorf("missing dependency for `google.protobuf` this is required to generate the `Any` extensions")
		}
	}
	for _, service := range c.Model.Services {
		if err := c.annotateService(service, annotations); err != nil {
			return err
		}
	}
	var serviceImports []string
	var messageImports []string
	for _, p := range c.Dependencies {
		if p.ApiPackage == c.Model.PackageName || p.Name == c.PackageName {
			continue
		}
		if p.RequiredByServices && len(c.Model.Services) != 0 {
			serviceImports = append(serviceImports, p.Name)
			annotations.DependsOn[p.Name] = p
		} else if p.Required {
			messageImports = append(messageImports, p.Name)
			annotations.DependsOn[p.Name] = p
		}
	}
	slices.Sort(serviceImports)
	slices.Sort(messageImports)
	annotations.ServiceImports = serviceImports
	annotations.MessageImports = messageImports
	return nil
}
