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
	"log/slog"
	"maps"
	"slices"

	"github.com/googleapis/librarian/internal/license"
)

type modelAnnotations struct {
	CopyrightYear    string
	BoilerPlate      []string
	PackageName      string
	PackageVersion   string
	ReleaseLevel     string
	MonorepoRoot     string
	DependsOn        map[string]*Dependency
	WktPackage       string
	PerServiceTraits bool
	DefaultTraits    []string
	AllTraits        []*traitDefinition
}

// traitDefinition provides information about each package trait.
//
// In Swift, a library may be configured to generate a package trait
// per-service. Some services depend on request types that are only available if
// other services are enabled. In other words, if the trait for service A is
// defined, then the trait for service B may need to be defined because A uses
// types from B.
type traitDefinition struct {
	// The name of this trait.
	Name string
	// The set of additional traits enabled by the service's trait.
	EnabledTraits []string
}

// HasDependencies returns true if the package has dependencies on other packages.
//
// The mustache templates use this to omit code that goes unused when the package has no
// dependencies.
func (ann *modelAnnotations) HasDependencies() bool {
	return len(ann.DependsOn) != 0
}

// EnablesOtherTraits returns true if this service's trait enables other traits too.
func (ann *traitDefinition) EnablesOtherTraits() bool {
	return len(ann.EnabledTraits) != 0
}

// Dependencies returns the list of dependencies for this package.
func (ann *modelAnnotations) Dependencies() []*Dependency {
	deps := slices.Collect(maps.Values(ann.DependsOn))
	slices.SortFunc(deps, func(a, b *Dependency) int { return cmp.Compare(a.Name, b.Name) })
	return deps
}

func (ann *modelAnnotations) HasTraits() bool {
	return len(ann.AllTraits) != 0
}

func (c *codec) annotateModel() error {
	annotations := &modelAnnotations{
		CopyrightYear:  c.GenerationYear,
		BoilerPlate:    license.HeaderBulk(),
		PackageName:    c.PackageName,
		PackageVersion: c.PackageVersion,
		ReleaseLevel:   c.ReleaseLevel,
		MonorepoRoot:   c.MonorepoRoot,
		DependsOn:      map[string]*Dependency{},
		WktPackage:     wellKnownSwiftPackage,
		DefaultTraits:  c.DefaultTraits,
	}
	if dep, ok := c.ApiPackages[wellKnownProtobufPackage]; ok {
		annotations.WktPackage = dep.Name
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
	// The services are annotated last because the annotation assumes messages
	// and enums are already annotated.
	allTraits := make([]*traitDefinition, 0, len(c.Model.Services))
	for _, service := range c.Model.Services {
		ann, err := c.annotateService(service, annotations)
		if err != nil {
			return err
		}
		var enabledTraits []string
		for _, svc := range ann.RequiredServices {
			enabledTraits = append(enabledTraits, c.traitName(svc))
		}
		slices.Sort(enabledTraits)
		trait := &traitDefinition{
			Name:          c.traitName(service),
			EnabledTraits: enabledTraits,
		}
		allTraits = append(allTraits, trait)
	}
	if !c.PerServiceTraits {
		// The maximum (15) was chosen more or less arbitrarily circa 2026-05. At
		// the time, only a handful of packages exceeded this number of services.
		if len(c.Model.Services) > 15 {
			slog.Warn("package has more than 15 services, consider enabling per-service features", "package", c.PackageName, "count", len(c.Model.Services))
		}
		return nil
	}
	// Rarely, some messages and enums are not used by any service. These
	// will lack any feature gates, but may depend on messages that do.
	// Change them to work only if all features are enabled.
	slices.SortFunc(allTraits, func(a, b *traitDefinition) int {
		return cmp.Compare(a.Name, b.Name)
	})
	annotations.AllTraits = allTraits
	allTraitNames := make([]string, 0, len(allTraits))
	for _, t := range allTraits {
		allTraitNames = append(allTraitNames, t.Name)
	}
	for msg := range c.Model.AllMessages() {
		if msg.Codec == nil {
			continue
		}
		annotation := msg.Codec.(*messageAnnotations)
		if len(annotation.GatedBy) > 0 {
			continue
		}
		if msg.ServicePlaceholder {
			// Messages that are placeholders for services just get the same
			// gating traits as the service.
			annotation.GatedOp = " || "
			annotation.GatedBy = []string{pascalCase(msg.Name)}
		} else {
			annotation.GatedOp = " && "
			annotation.GatedBy = allTraitNames
		}
	}
	for enum := range c.Model.AllEnums() {
		if enum.Codec == nil {
			continue
		}
		annotation := enum.Codec.(*enumAnnotations)
		if len(annotation.GatedBy) > 0 {
			continue
		}
		annotation.GatedOp = " && "
		annotation.GatedBy = allTraitNames
	}
	return nil
}
