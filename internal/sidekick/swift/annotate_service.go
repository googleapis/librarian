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

	"github.com/googleapis/librarian/internal/sidekick/api"
)

type serviceAnnotations struct {
	Name             string
	ClientName       string
	StubPrefix       string
	DocLines         []string
	RestMethods      []*api.Method
	PackageName      string
	QuickstartMethod *api.Method
	Model            *modelAnnotations
	DependsOn        map[string]*Dependency
}

// Imports returns the list of dependencies for this package.
func (ann *serviceAnnotations) ServiceImports() []*Dependency {
	deps := slices.Collect(maps.Values(ann.DependsOn))
	slices.SortFunc(deps, func(a, b *Dependency) int { return cmp.Compare(a.Name, b.Name) })
	return deps
}

func (c *codec) annotateService(service *api.Service, model *modelAnnotations) error {
	docLines, err := c.formatDocumentation(service.Documentation, service.Scopes())
	if err != nil {
		return err
	}
	var restMethods []*api.Method
	for _, method := range service.Methods {
		if isGeneratedMethod(method) {
			if err := c.annotateMethod(method, model); err != nil {
				return err
			}
			restMethods = append(restMethods, method)
		}
	}
	var quickstartMethod *api.Method
	if service.QuickstartMethod != nil && isGeneratedMethod(service.QuickstartMethod) {
		quickstartMethod = service.QuickstartMethod
	}

	annotations := &serviceAnnotations{
		Name:             pascalCase(service.Name),
		ClientName:       pascalCase(service.Name + "Client"),
		StubPrefix:       pascalCaseNoMangling(service.Name),
		DocLines:         docLines,
		RestMethods:      restMethods,
		PackageName:      c.PackageName,
		QuickstartMethod: quickstartMethod,
		Model:            model,
		DependsOn:        map[string]*Dependency{},
	}

	for _, p := range c.Dependencies {
		if p.ApiPackage == c.Model.PackageName || p.Name == c.PackageName {
			continue
		}
		if p.RequiredByServices {
			c.addPackageDependency(p.Name)
			annotations.DependsOn[p.Name] = p
		}
	}

	// Services always depend on well known types
	if dep, ok := c.ApiPackages[wellKnownProtobufPackage]; ok {
		c.addPackageDependency(dep.Name)
		annotations.DependsOn[dep.Name] = dep
	}

	for _, method := range restMethods {
		if method.InputType != nil {
			if method.InputType.Package != c.Model.PackageName {
				if dep, ok := c.ApiPackages[method.InputType.Package]; ok {
					annotations.DependsOn[dep.Name] = dep
				}
			}
		}
		if method.OutputType != nil {
			if method.OutputType.Package != c.Model.PackageName {
				if dep, ok := c.ApiPackages[method.OutputType.Package]; ok {
					annotations.DependsOn[dep.Name] = dep
				}
			}
		}
	}

	service.Codec = annotations
	return nil
}

func isGeneratedMethod(method *api.Method) bool {
	return method.PathInfo != nil && len(method.PathInfo.Bindings) != 0
}
