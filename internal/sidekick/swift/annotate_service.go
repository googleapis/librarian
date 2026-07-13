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
	"slices"
	"strings"

	"github.com/googleapis/librarian/internal/sidekick/api"
)

type serviceAnnotations struct {
	Name             string
	ClientName       string
	StubPrefix       string
	HostnameShort    string
	DocLines         []string
	RestMethods      []*api.Method
	PackageName      string
	QuickstartMethod *api.Method
	Model            *modelAnnotations
	DependsOn        map[string]*Dependency
	IsGated          bool

	// Any additional services required by this service.
	//
	// Typically this happens on discovery-based APIs where services with LROs
	// depend on request messages provided by the service that can poll the LRO.
	RequiredServices map[string]*api.Service
}

// ServiceImports returns the list of dependencies for this service.
func (ann *serviceAnnotations) ServiceImports() []string {
	result := make([]string, 0, len(ann.DependsOn))
	for _, dep := range ann.DependsOn {
		result = append(result, dep.Name)
	}
	slices.Sort(result)
	return result
}

// SnippetImports returns the sorted list of dependencies for this service's
// snippets.
//
// Service snippets are generated examples that show how to initialize the
// service and call its key methods. They need imports beyond the package for
// mixins and other external packages. But they do not need the implementation
// dependencies, such as `GoogleCloudAuth` or `GoogleCloudGax`.
func (ann *serviceAnnotations) SnippetImports() []string {
	var result []string
	for _, dep := range ann.DependsOn {
		// Only dependencies that map to some source-API package
		// (e.g. google.protobuf) are needed by the snippets.
		if dep.ApiPackage != "" {
			result = append(result, dep.Name)
		}
	}
	slices.Sort(result)
	return result
}

func (c *codec) annotateService(service *api.Service, model *modelAnnotations) (*serviceAnnotations, error) {
	docLines, err := c.formatDocumentation(service.Documentation, service.Scopes())
	if err != nil {
		return nil, err
	}
	requiredServices := make(map[string]*api.Service)
	var restMethods []*api.Method
	for _, method := range service.Methods {
		if isGeneratedMethod(method) {
			if err := c.annotateMethod(method, model); err != nil {
				return nil, err
			}
			restMethods = append(restMethods, method)
			if method.IsLroPoller && method.SourceService != nil {
				requiredServices[method.SourceService.ID] = method.SourceService
			}
		}
	}
	var quickstartMethod *api.Method
	if service.QuickstartMethod != nil && isGeneratedMethod(service.QuickstartMethod) {
		quickstartMethod = service.QuickstartMethod
	}

	name := pascalCase(service.Name)
	annotations := &serviceAnnotations{
		Name:             name,
		ClientName:       pascalCase(service.Name + "Client"),
		StubPrefix:       pascalCaseNoMangling(service.Name),
		HostnameShort:    strings.TrimSuffix(service.DefaultHost, ".googleapis.com"),
		DocLines:         docLines,
		RestMethods:      restMethods,
		PackageName:      c.PackageName,
		QuickstartMethod: quickstartMethod,
		Model:            model,
		DependsOn:        map[string]*Dependency{},
	}
	if c.PerServiceTraits {
		annotations.IsGated = true
		annotations.RequiredServices = requiredServices
	}

	// Iterate through the list of all dependencies declared in librarian.yaml
	// If the dependency is marked as "required_by_services", then we force it
	// as an import for the generated service files.
	for _, p := range c.Dependencies {
		if p.ApiPackage == c.Model.PackageName || p.Name == c.PackageName {
			continue
		}
		if p.RequiredByServices {
			if _, err := c.addDependency(p); err != nil {
				return nil, err
			}
			annotations.DependsOn[p.Name] = p
		}
	}

	// Services always depend on well known types
	wktDep, err := c.addApiPackageDependency(wellKnownProtobufPackage)
	if err != nil {
		return nil, err
	}
	annotations.DependsOn[wktDep.Name] = wktDep

	for _, method := range restMethods {
		if method.InputType != nil {
			if method.InputType.Package != c.Model.PackageName {
				dep, err := c.addApiPackageDependency(method.InputType.Package)
				if err != nil {
					return nil, err
				}
				if dep != nil {
					annotations.DependsOn[dep.Name] = dep
				}
			}
		}
		if method.OutputType != nil {
			if method.OutputType.Package != c.Model.PackageName {
				dep, err := c.addApiPackageDependency(method.OutputType.Package)
				if err != nil {
					return nil, err
				}
				if dep != nil {
					annotations.DependsOn[dep.Name] = dep
				}
			}
		}
		if method.IsLRO && method.OperationInfo != nil {
			// LROs depend on PollableOperation package
			lroDep, err := c.addPackageDependency(lroSwiftPackage)
			if err != nil {
				return nil, err
			}
			if lroDep != nil {
				annotations.DependsOn[lroDep.Name] = lroDep
			}

			// LRO error mapping relies on GoogleRpc.Code, so we depend on GoogleRpc package
			rpcDep, err := c.addApiPackageDependency("google.rpc")
			if err != nil {
				return nil, err
			}
			if rpcDep != nil {
				annotations.DependsOn[rpcDep.Name] = rpcDep
			}

			// Ensure we have the necessary dependencies for the LRO response and metadata types.
			respMsg, err := lookupMessage(c.Model, method.OperationInfo.ResponseTypeID)
			if err != nil {
				return nil, err
			}
			if respMsg.Package != c.Model.PackageName {
				dep, err := c.addApiPackageDependency(respMsg.Package)
				if err != nil {
					return nil, err
				}
				if dep != nil {
					annotations.DependsOn[dep.Name] = dep
				}
			}
			metaMsg, err := lookupMessage(c.Model, method.OperationInfo.MetadataTypeID)
			if err != nil {
				return nil, err
			}
			if metaMsg.Package != c.Model.PackageName {
				dep, err := c.addApiPackageDependency(metaMsg.Package)
				if err != nil {
					return nil, err
				}
				if dep != nil {
					annotations.DependsOn[dep.Name] = dep
				}
			}
		}
	}

	service.Codec = annotations
	if err := c.addFeatureAnnotations(service); err != nil {
		return nil, err
	}
	return annotations, nil
}

func isGeneratedMethod(method *api.Method) bool {
	return method.PathInfo != nil && len(method.PathInfo.Bindings) != 0
}

func (c *codec) addFeatureAnnotations(
	service *api.Service) error {
	if !c.PerServiceTraits {
		return nil
	}
	traitName := c.traitName(service)
	deps := api.FindServiceDependencies(c.Model, service.ID)
	for _, id := range deps.Enums {
		enum := c.Model.Enum(id)
		// Some messages are not annotated (e.g. external messages).
		if enum == nil || enum.Codec == nil {
			continue
		}
		annotation, ok := enum.Codec.(*enumAnnotations)
		if !ok {
			return fmt.Errorf("bad annotation type for %s", id)
		}
		annotation.GatedBy = insertGatingTrait(annotation.GatedBy, traitName)
		annotation.GatedOp = " || "
	}
	for _, id := range deps.Messages {
		msg := c.Model.Message(id)
		// Some messages are not annotated (e.g. external messages).
		if msg == nil || msg.Codec == nil {
			continue
		}
		annotation, ok := msg.Codec.(*messageAnnotations)
		if !ok {
			return fmt.Errorf("bad annotation type for %s", id)
		}
		if !msg.ServicePlaceholder {
			// Messages that are placeholders for services just get the same
			// gating traits as the service.
			annotation.GatedBy = insertGatingTrait(annotation.GatedBy, traitName)
			annotation.GatedOp = " || "
		}
	}
	return nil
}

func (c *codec) traitName(service *api.Service) string {
	return pascalCase(service.Name)
}

func insertGatingTrait(gatedBy []string, traitName string) []string {
	if index, found := slices.BinarySearch(gatedBy, traitName); !found {
		gatedBy = slices.Insert(gatedBy, index, traitName)
	}
	return gatedBy
}
