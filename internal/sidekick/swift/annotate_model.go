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
	"maps"
	"slices"

	"github.com/googleapis/librarian/internal/license"
)

type modelAnnotations struct {
	CopyrightYear string
	BoilerPlate   []string
	PackageName   string
	MonorepoRoot  string
	DependsOn     map[string]*Dependency
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

func (codec *codec) annotateModel() error {
	annotations := &modelAnnotations{
		CopyrightYear: codec.GenerationYear,
		BoilerPlate:   license.HeaderBulk(),
		PackageName:   codec.PackageName,
		MonorepoRoot:  codec.MonorepoRoot,
		DependsOn:     map[string]*Dependency{},
	}
	codec.Model.Codec = annotations
	for _, message := range codec.Model.Messages {
		if err := codec.annotateMessage(message, annotations); err != nil {
			return err
		}
	}
	for _, enum := range codec.Model.Enums {
		codec.annotateEnum(enum, annotations)
	}
	for _, service := range codec.Model.Services {
		codec.annotateService(service, annotations)
	}
	if len(codec.Model.Services) != 0 {
		for _, p := range codec.Dependencies {
			if p.RequiredByServices {
				annotations.DependsOn[p.Name] = p
			}
		}
	}
	return nil
}
