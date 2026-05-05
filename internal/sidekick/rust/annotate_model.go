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

// Package rust implements a native Rust code generator.
package rust

import (
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"strings"

	"github.com/googleapis/librarian/internal/license"
	"github.com/googleapis/librarian/internal/sidekick/api"
	"github.com/googleapis/librarian/internal/sidekick/language"
)

// errQuickstartServiceNotFound is returned when the requested quickstart service override is not found.
var errQuickstartServiceNotFound = errors.New("quickstart_service_override not found")

type modelAnnotations struct {
	PackageName      string
	PackageVersion   string
	ReleaseLevel     string
	PackageNamespace string
	RequiredPackages []string
	ExternPackages   []string
	HasLROs          bool
	CopyrightYear    string
	BoilerPlate      []string
	DefaultHost      string
	DefaultHostShort string
	// Services without methods create a lot of warnings in Rust. The dead code
	// analysis is extremely good, and can determine that several types and
	// member variables are going unused if the service does not have any
	// generated methods. Filter out the services to the subset that will
	// produce at least one method.
	Services          []*api.Service
	NameToLower       string
	NotForPublication bool
	// A list of `#[allow(rustdoc::*)]` warnings to disable.
	DisabledRustdocWarnings []string
	// A list of `#[allow(clippy::*)]` warnings to disable.
	DisabledClippyWarnings []string
	// Sets the default system parameters.
	DefaultSystemParameters []systemParameter
	// Enables per-service features.
	PerServiceFeatures bool
	// The set of default features, only applicable if `PerServiceFeatures` is
	// true.
	DefaultFeatures []string
	// A list of additional modules loaded by the `lib.rs` file.
	ExtraModules []string
	// If true, at lease one service has a method we cannot wrap (yet).
	Incomplete bool
	// If true, the generator will produce reference documentation samples for message fields setters.
	GenerateSetterSamples bool
	// If true, the generator will produce reference documentation samples for functions that correspond to RPCs.
	GenerateRpcSamples bool
	// If true, the generated code includes detailed tracing attributes on HTTP
	// requests.
	DetailedTracingAttributes bool
	// If true, the generated builders's visibility should be restricted to the crate.
	InternalBuilders bool
	// The service to use for the package-level quickstart sample.
	// Rust generation may decide not to generate some services,
	// e.g. if the methods have no bindings. On occasion the service
	// selected at the model level will be skipped for Rust generation
	// so we need to choose a different one.
	QuickstartService *api.Service
}

// IsWktCrate returns true when bootstrapping the well-known types crate the templates add some
// ad-hoc code.
func (m *modelAnnotations) IsWktCrate() bool {
	return m.PackageName == "google-cloud-wkt"
}

// BuilderVisibility returns the visibility for client and request builders.
func (m *modelAnnotations) BuilderVisibility() string {
	if m.InternalBuilders {
		return "pub(crate)"
	}
	return "pub"
}

// HasServices returns true if there are any services in the model.
func (m *modelAnnotations) HasServices() bool {
	return len(m.Services) > 0
}

// IsGaxiCrate returns true if we handle references to `gaxi` traits from within the `gaxi` crate, by
// injecting some ad-hoc code.
func (m *modelAnnotations) IsGaxiCrate() bool {
	return m.PackageName == "google-cloud-gax-internal"
}

// ReleaseLevelIsGA returns true if the ReleaseLevel attribute indicates this
// is a GA library.
func (m *modelAnnotations) ReleaseLevelIsGA() bool {
	return m.ReleaseLevel == "GA" || m.ReleaseLevel == "stable"
}

// annotateModel creates a struct used as input for Mustache templates.
// Fields and methods defined in this struct directly correspond to Mustache
// tags. For example, the Mustache tag {{#Services}} uses the
// [Template.Services] field.
func annotateModel(model *api.API, codec *codec) (*modelAnnotations, error) {
	codec.hasServices = len(model.Services) > 0

	resolveUsedPackages(model, codec.extraPackages)
	// Annotate enums and messages that we intend to generate. In the
	// process we discover the external dependencies and trim the list of
	// packages used by this API.
	// This API's enums and messages get full annotations.
	for _, e := range model.Enums {
		if err := codec.annotateEnum(e, model, true); err != nil {
			return nil, err
		}
	}
	for _, m := range model.Messages {
		if err := codec.annotateMessage(m, model, true); err != nil {
			return nil, err
		}
	}
	// External enums and messages get only basic annotations
	// used for sample generation.
	// External enums and messages are the ones that have yet
	// to be annotated.
	for e := range model.AllEnums() {
		if e.Codec == nil {
			if err := codec.annotateEnum(e, model, false); err != nil {
				return nil, err
			}
		}
	}
	for m := range model.AllMessages() {
		if m.Codec == nil {
			if err := codec.annotateMessage(m, model, false); err != nil {
				return nil, err
			}
		}
	}
	hasLROs := false
	for _, s := range model.Services {
		for _, m := range s.Methods {
			if m.OperationInfo != nil || m.DiscoveryLro != nil {
				hasLROs = true
			}
			if !codec.generateMethod(m) {
				continue
			}
			if _, err := codec.annotateMethod(m); err != nil {
				return nil, err
			}
			if m := m.InputType; m != nil {
				if err := codec.annotateMessage(m, model, true); err != nil {
					return nil, err
				}
			}
			if m := m.OutputType; m != nil {
				if err := codec.annotateMessage(m, model, true); err != nil {
					return nil, err
				}
			}
			if si := m.SampleInfo; si != nil {
				codec.annotateSampleInfo(si, m)
			}
		}
		if _, err := codec.annotateService(s); err != nil {
			return nil, err
		}
	}

	servicesSubset := language.FilterSlice(model.Services, func(s *api.Service) bool {
		return slices.ContainsFunc(s.Methods, func(m *api.Method) bool { return codec.generateMethod(m) })
	})
	// The maximum (15) was chosen more or less arbitrarily circa 2025-06. At
	// the time, only a handful of services exceeded this number of services.
	if len(servicesSubset) > 15 && !codec.perServiceFeatures {
		slog.Warn("package has more than 15 services, consider enabling per-service features", "package", codec.packageName(model), "count", len(servicesSubset))
	}

	// Delay this until the Codec had a chance to compute what packages are
	// used.
	findUsedPackages(model, codec)
	defaultHost := func() string {
		if len(model.Services) > 0 {
			return model.Services[0].DefaultHost
		}
		return ""
	}()
	defaultHostShort := func() string {
		idx := strings.Index(defaultHost, ".")
		if idx == -1 {
			return defaultHost
		}
		return defaultHost[:idx]
	}()

	var quickstartService *api.Service
	if codec.quickstartServiceOverride != "" {
		idx := slices.IndexFunc(servicesSubset, func(s *api.Service) bool {
			return strings.EqualFold(codec.ServiceName(s), codec.quickstartServiceOverride) || strings.EqualFold(s.Name, codec.quickstartServiceOverride)
		})
		if idx != -1 {
			quickstartService = servicesSubset[idx]
		} else {
			return nil, fmt.Errorf("%w: %q not found in generated services for package %q", errQuickstartServiceNotFound, codec.quickstartServiceOverride, codec.packageName(model))
		}
	} else if model.QuickstartService != nil {
		if slices.ContainsFunc(servicesSubset, func(s *api.Service) bool { return s == model.QuickstartService }) {
			quickstartService = model.QuickstartService
		}
	}

	ann := &modelAnnotations{
		PackageName:      codec.packageName(model),
		PackageNamespace: codec.rootModuleName(model),
		PackageVersion:   codec.version,
		ReleaseLevel:     codec.releaseLevel,
		RequiredPackages: requiredPackages(codec.extraPackages),
		ExternPackages:   externPackages(codec.extraPackages),
		HasLROs:          hasLROs,
		CopyrightYear:    codec.generationYear,
		BoilerPlate: append(license.HeaderBulk(),
			"",
			" Code generated by sidekick. DO NOT EDIT."),
		DefaultHost:             defaultHost,
		DefaultHostShort:        defaultHostShort,
		Services:                servicesSubset,
		NameToLower:             strings.ToLower(model.Name),
		NotForPublication:       codec.doNotPublish,
		DisabledRustdocWarnings: codec.disabledRustdocWarnings,
		DisabledClippyWarnings:  codec.disabledClippyWarnings,
		PerServiceFeatures:      codec.perServiceFeatures && len(servicesSubset) > 0,
		ExtraModules:            codec.extraModules,
		Incomplete: slices.ContainsFunc(model.Services, func(s *api.Service) bool {
			return slices.ContainsFunc(s.Methods, func(m *api.Method) bool { return !codec.generateMethod(m) })
		}),
		GenerateSetterSamples:     codec.generateSetterSamples,
		GenerateRpcSamples:        codec.generateRpcSamples,
		DetailedTracingAttributes: codec.detailedTracingAttributes,
		InternalBuilders:          codec.internalBuilders,
		QuickstartService:         quickstartService,
	}

	codec.addFeatureAnnotations(model, ann)

	model.Codec = ann
	return ann, nil
}

func (c *codec) addFeatureAnnotations(model *api.API, ann *modelAnnotations) {
	if !c.perServiceFeatures {
		return
	}
	var allFeatures []string
	for _, service := range ann.Services {
		svcAnn := service.Codec.(*serviceAnnotations)
		allFeatures = append(allFeatures, svcAnn.FeatureName())
		deps := api.FindServiceDependencies(model, service.ID)
		for _, id := range deps.Enums {
			enum := model.Enum(id)
			// Some messages are not annotated (e.g. external messages).
			if enum == nil || enum.Codec == nil {
				continue
			}
			annotation := enum.Codec.(*enumAnnotation)
			annotation.FeatureGates = append(annotation.FeatureGates, svcAnn.FeatureName())
			slices.Sort(annotation.FeatureGates)
			annotation.FeatureGatesOp = "any"
		}
		for _, id := range deps.Messages {
			msg := model.Message(id)
			// Some messages are not annotated (e.g. external messages).
			if msg == nil || msg.Codec == nil {
				continue
			}
			annotation := msg.Codec.(*messageAnnotation)
			annotation.FeatureGates = append(annotation.FeatureGates, svcAnn.FeatureName())
			slices.Sort(annotation.FeatureGates)
			annotation.FeatureGatesOp = "any"
			for _, one := range msg.OneOfs {
				if one.Codec == nil {
					continue
				}
				annotation := one.Codec.(*oneOfAnnotation)
				annotation.FeatureGates = append(annotation.FeatureGates, svcAnn.FeatureName())
				slices.Sort(annotation.FeatureGates)
				annotation.FeatureGatesOp = "any"
			}
		}
	}
	// Rarely, some messages and enums are not used by any service. These
	// will lack any feature gates, but may depend on messages that do.
	// Change them to work only if all features are enabled.
	slices.Sort(allFeatures)
	for msg := range model.AllMessages() {
		if msg.Codec == nil {
			continue
		}
		annotation := msg.Codec.(*messageAnnotation)
		if len(annotation.FeatureGates) > 0 {
			continue
		}
		annotation.FeatureGatesOp = "all"
		annotation.FeatureGates = allFeatures
	}
	for enum := range model.AllEnums() {
		if enum.Codec == nil {
			continue
		}
		annotation := enum.Codec.(*enumAnnotation)
		if len(annotation.FeatureGates) > 0 {
			continue
		}
		annotation.FeatureGatesOp = "all"
		annotation.FeatureGates = allFeatures
	}
	ann.DefaultFeatures = c.defaultFeatures
	if ann.DefaultFeatures == nil {
		ann.DefaultFeatures = allFeatures
	}
}

// packageToModuleName maps "google.foo.v1" to "google::foo::v1".
func packageToModuleName(p string) string {
	components := strings.Split(p, ".")
	for i, c := range components {
		components[i] = toSnake(c)
	}
	return strings.Join(components, "::")
}
