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

package rust

import (
	"cmp"
	"slices"

	"github.com/googleapis/librarian/internal/sidekick/api"
	"github.com/googleapis/librarian/internal/sidekick/language"
	"github.com/iancoleman/strcase"
)

type serviceAnnotations struct {
	// The name of the service. The Rust naming conventions requires this to be
	// in `PascalCase`. Notably, names like `IAM` *must* become `Iam`, but
	// `IAMService` can stay unchanged.
	Name string
	// The source specification package name mapped to Rust modules. That is,
	// `google.service.v1` becomes `google::service::v1`.
	PackageModuleName string
	// For each service we generate a module containing all its builders.
	// The Rust naming conventions required this to be `snake_case` format.
	ModuleName string
	DocLines   []string
	// Only a subset of the methods is generated.
	Methods     []*api.Method
	DefaultHost string
	// A set of all types involved in an LRO, whether used as metadata or
	// response.
	LROTypes []*api.Message
	APITitle string
	// If set, gate this service under a feature named `ModuleName`.
	PerServiceFeatures bool
	// If true, there is a handwritten client surface.
	HasVeneer bool
	// If true, the transport stub is extensible from outside of
	// `transport.rs`. This is done to add ad-hoc streaming support.
	ExtendGrpcTransport bool
	// If true, the service has a method we cannot wrap (yet).
	Incomplete bool
	// If true, the generated code includes detailed tracing attributes on HTTP
	// requests.
	DetailedTracingAttributes bool
	// If true, the generated builders's visibility should be restricted to the crate.
	InternalBuilders bool
}

// BuilderVisibility returns the visibility for client and request builders.
func (s *serviceAnnotations) BuilderVisibility() string {
	if s.InternalBuilders {
		return "pub(crate)"
	}
	return "pub"
}

// HasLROs returns true if this service includes methods that return long-running operations.
func (s *serviceAnnotations) HasLROs() bool {
	if len(s.LROTypes) > 0 {
		return true
	}
	return slices.IndexFunc(s.Methods, func(m *api.Method) bool { return m.DiscoveryLro != nil }) != -1
}

// MaximumAPIVersion returns the highest (in alphanumeric order) APIVersion of
// all the methods in the service.
func (s *serviceAnnotations) MaximumAPIVersion() string {
	if len(s.Methods) == 0 {
		return ""
	}
	max := slices.MaxFunc(s.Methods, func(a, b *api.Method) int { return cmp.Compare(a.APIVersion, b.APIVersion) })
	return max.APIVersion
}

// FeatureName returns the feature name for the service.
func (a *serviceAnnotations) FeatureName() string {
	return strcase.ToKebab(a.ModuleName)
}

func (c *codec) annotateService(s *api.Service) (*serviceAnnotations, error) {
	// Some codecs skip some methods.
	methods := language.FilterSlice(s.Methods, func(m *api.Method) bool {
		return c.generateMethod(m)
	})
	seenLROTypes := make(map[string]bool)
	var lroTypes []*api.Message
	for _, m := range methods {
		if m.OperationInfo != nil {
			if _, ok := seenLROTypes[m.OperationInfo.MetadataTypeID]; !ok {
				seenLROTypes[m.OperationInfo.MetadataTypeID] = true
				lroTypes = append(lroTypes, s.Model.Message(m.OperationInfo.MetadataTypeID))
			}
			if _, ok := seenLROTypes[m.OperationInfo.ResponseTypeID]; !ok {
				seenLROTypes[m.OperationInfo.ResponseTypeID] = true
				lroTypes = append(lroTypes, s.Model.Message(m.OperationInfo.ResponseTypeID))
			}
		}
	}
	serviceName := c.ServiceName(s)
	moduleName := toSnake(serviceName)
	docLines, err := c.formatDocComments(
		s.Documentation, s.ID, s.Model, []string{s.ID, s.Package})
	if err != nil {
		return nil, err
	}
	ann := &serviceAnnotations{
		Name:                      toPascal(serviceName),
		PackageModuleName:         packageToModuleName(s.Package),
		ModuleName:                moduleName,
		DocLines:                  docLines,
		Methods:                   methods,
		DefaultHost:               s.DefaultHost,
		LROTypes:                  lroTypes,
		APITitle:                  s.Model.Title,
		PerServiceFeatures:        c.perServiceFeatures,
		HasVeneer:                 c.hasVeneer,
		ExtendGrpcTransport:       c.extendGrpcTransport,
		Incomplete:                slices.ContainsFunc(s.Methods, func(m *api.Method) bool { return !c.generateMethod(m) }),
		DetailedTracingAttributes: c.detailedTracingAttributes,
		InternalBuilders:          c.internalBuilders,
	}
	s.Codec = ann
	return ann, nil
}
