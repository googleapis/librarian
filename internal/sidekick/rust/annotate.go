// Copyright 2024 Google LLC
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
	"fmt"
	"log/slog"
	"slices"
	"strings"

	"github.com/googleapis/librarian/internal/sidekick/api"
	"github.com/googleapis/librarian/internal/sidekick/language"
	"github.com/googleapis/librarian/internal/sidekick/license"
	"github.com/iancoleman/strcase"
)

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
	// If true, the generated code includes detailed tracing attributes on HTTP
	// requests.
	DetailedTracingAttributes bool
}

// IsWktCrate returns true when bootstrapping the well-known types crate the templates add some
// ad-hoc code.
func (m *modelAnnotations) IsWktCrate() bool {
	return m.PackageName == "google-cloud-wkt"
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
	// If true, the service has a method we cannot wrap (yet).
	Incomplete bool
	// If true, the generated code includes detailed tracing attributes on HTTP
	// requests.
	DetailedTracingAttributes bool
}

// HasBindingSubstitutions returns true if the method has binding substitutions.
func (m *methodAnnotation) HasBindingSubstitutions() bool {
	for _, b := range m.PathInfo.Bindings {
		for _, s := range b.PathTemplate.Segments {
			if s.Variable != nil {
				return true
			}
		}
	}
	return false
}

// HasLROs returns true if this service includes methods that return long-running operations.
func (s *serviceAnnotations) HasLROs() bool {
	if len(s.LROTypes) > 0 {
		return true
	}
	return slices.IndexFunc(s.Methods, func(m *api.Method) bool { return m.DiscoveryLro != nil }) != -1
}

// FeatureName returns the feature name for the service.
func (a *serviceAnnotations) FeatureName() string {
	return strcase.ToKebab(a.ModuleName)
}

// MultiFeatureGates returns true if there are multiple feature gates.
func (a *messageAnnotation) MultiFeatureGates() bool {
	return len(a.FeatureGates) > 1
}

// MultiFeatureGates returns true if there are multiple feature gates.
func (a *enumAnnotation) MultiFeatureGates() bool {
	return len(a.FeatureGates) > 1
}

// MultiFeatureGates returns true if there are multiple feature gates.
func (a *oneOfAnnotation) MultiFeatureGates() bool {
	return len(a.FeatureGates) > 1
}

// SingleFeatureGate returns true if there is a single feature gate.
func (a *messageAnnotation) SingleFeatureGate() bool {
	return len(a.FeatureGates) == 1
}

// SingleFeatureGate returns true if there is a single feature gate.
func (a *enumAnnotation) SingleFeatureGate() bool {
	return len(a.FeatureGates) == 1
}

// SingleFeatureGate returns true if there is a single feature gate.
func (a *oneOfAnnotation) SingleFeatureGate() bool {
	return len(a.FeatureGates) == 1
}

type messageAnnotation struct {
	Name       string
	ModuleName string
	// The fully qualified name, including the `codec.modulePath` prefix. For
	// messages in external packages this includes the package name.
	QualifiedName string
	// The fully qualified name, relative to `codec.modulePath`. Typically this
	// is the `QualifiedName` with the `crate::model::` prefix removed.
	RelativeName string
	// The fully qualified name for examples. For messages in external packages
	// this is basically `QualifiedName`. For messages in the current package
	// this includes `modelAnnotations.PackageName`.
	NameInExamples string
	// The package name mapped to Rust modules. That is, `google.service.v1`
	// becomes `google::service::v1`.
	PackageModuleName string
	// The FQN is the source specification
	SourceFQN      string
	DocLines       []string
	HasNestedTypes bool
	// All the fields except OneOfs.
	BasicFields []*api.Field
	// If set, this message is only enabled when some features are enabled.
	FeatureGates   []string
	FeatureGatesOp string
	// If true, this message's visibility should only be `pub(crate)`
	Internal bool
}

type methodAnnotation struct {
	Name                      string
	NameNoMangling            string
	BuilderName               string
	DocLines                  []string
	PathInfo                  *api.PathInfo
	Body                      string
	ServiceNameToPascal       string
	ServiceNameToCamel        string
	ServiceNameToSnake        string
	OperationInfo             *operationInfo
	SystemParameters          []systemParameter
	ReturnType                string
	HasVeneer                 bool
	Attributes                []string
	RoutingRequired           bool
	DetailedTracingAttributes bool
}

type pathInfoAnnotation struct {
	// Whether the request has a body or not
	HasBody bool

	// A list of possible request parameters
	//
	// This is only used for gRPC-based clients, where we must consider all
	// possible request parameters.
	//
	// https://google.aip.dev/client-libraries/4222
	//
	// Templates are ignored. We only care about the FieldName and FieldAccessor.
	UniqueParameters []*bindingSubstitution

	// Whether this is idempotent by default
	//
	// This is only used for gRPC-based clients.
	IsIdempotent string
}

type operationInfo struct {
	MetadataType     string
	ResponseType     string
	PackageNamespace string
}

// OnlyMetadataIsEmpty returns true if only the metadata is empty.
func (info *operationInfo) OnlyMetadataIsEmpty() bool {
	return info.MetadataType == "wkt::Empty" && info.ResponseType != "wkt::Empty"
}

// OnlyResponseIsEmpty returns true if only the response is empty.
func (info *operationInfo) OnlyResponseIsEmpty() bool {
	return info.MetadataType != "wkt::Empty" && info.ResponseType == "wkt::Empty"
}

// BothAreEmpty returns true if both the metadata and response are empty.
func (info *operationInfo) BothAreEmpty() bool {
	return info.MetadataType == "wkt::Empty" && info.ResponseType == "wkt::Empty"
}

// NoneAreEmpty returns true if neither the metadata nor the response are empty.
func (info *operationInfo) NoneAreEmpty() bool {
	return info.MetadataType != "wkt::Empty" && info.ResponseType != "wkt::Empty"
}

type discoveryLroAnnotations struct {
	MethodName            string
	ReturnType            string
	PollingPathParameters []discoveryLroPathParameter
}

type discoveryLroPathParameter struct {
	Name       string
	SetterName string
}

type routingVariantAnnotations struct {
	FieldAccessors   []string
	PrefixSegments   []string
	MatchingSegments []string
	SuffixSegments   []string
}

type bindingSubstitution struct {
	// Rust code to access the leaf field, given a `req`
	//
	// This field can be deeply nested. We need to capture code for the entire
	// chain. This accessor always returns an `Option<&T>`, even for fields
	// which are always present. This simplifies the mustache templates.
	//
	// The accessor should not
	// - copy any fields
	// - move any fields
	// - panic
	// - assume context i.e. use the try operator: `?`
	FieldAccessor string

	// The field name
	//
	// Nested fields are '.'-separated.
	//
	// e.g. "message_field.nested_field"
	FieldName string

	// The path template to match this substitution against
	//
	// e.g. ["projects", "*"]
	Template []string
}

// TemplateAsArray returns Rust code that yields an array of path segments.
//
// This array is supplied as an argument to `gaxi::path_parameter::try_match()`,
// and `gaxi::path_parameter::PathMismatchBuilder`.
//
// e.g.: `&[Segment::Literal("projects/"), Segment::SingleWildcard]`.
func (s *bindingSubstitution) TemplateAsArray() string {
	return "&[" + strings.Join(annotateSegments(s.Template), ", ") + "]"
}

// TemplateAsString returns the expected template, which can be used as a static string.
//
// e.g.: "projects/*".
func (s *bindingSubstitution) TemplateAsString() string {
	return strings.Join(s.Template, "/")
}

type pathBindingAnnotation struct {
	// The path format string for this binding
	//
	// e.g. "/v1/projects/{}/locations/{}"
	PathFmt string

	// The fields to be sent as query parameters for this binding
	QueryParams []*api.Field

	// The variables to be substituted into the path
	Substitutions []*bindingSubstitution

	// The codec is configured to generated detailed tracing attributes.
	DetailedTracingAttributes bool
}

// QueryParamsCanFail returns true if we serialize certain query parameters, which can fail. The code we generate
// uses the try operator '?'. We need to run this code in a closure which
// returns a `Result<>`.
func (b *pathBindingAnnotation) QueryParamsCanFail() bool {
	for _, f := range b.QueryParams {
		if f.Typez == api.MESSAGE_TYPE {
			return true
		}
	}
	return false
}

// HasVariablePath returns true if the path has a variable.
func (b *pathBindingAnnotation) HasVariablePath() bool {
	return len(b.Substitutions) != 0
}

// PathTemplate produces a path template suitable for instrumentation and logging.
// Variable parts are replaced with {field_name}.
func (b *pathBindingAnnotation) PathTemplate() string {
	if len(b.Substitutions) == 0 {
		return b.PathFmt
	}

	template := b.PathFmt
	for _, s := range b.Substitutions {
		// Construct the placeholder e.g., "{field_name}"
		placeholder := "{" + s.FieldName + "}"
		// Replace the first instance of "{}" with the field name placeholder
		template = strings.Replace(template, "{}", placeholder, 1)
	}
	return template
}

type oneOfAnnotation struct {
	// In Rust, `oneof` fields are fields inside a struct. These must be
	// `snake_case`. Possibly mangled with `r#` if the name is a Rust reserved
	// word.
	FieldName string
	// In Rust, each field gets a `set_{{FieldName}}` setter. These must be
	// `snake_case`, but are never mangled with a `r#` prefix.
	SetterName string
	// The `oneof` is represented by a Rust `enum`, these need to be `PascalCase`.
	EnumName string
	// The Rust `enum` may be in a deeply nested scope. This is a shortcut.
	QualifiedName string
	// The fully qualified name, relative to `codec.modulePath`. Typically this
	// is the `QualifiedName` with the `crate::model::` prefix removed.
	RelativeName string
	// The Rust `struct` that contains this oneof, fully qualified
	StructQualifiedName string
	// The fully qualified name for examples. For messages in external packages
	// this is basically `QualifiedName`. For messages in the current package
	// this includes `modelAnnotations.PackageName`.
	NameInExamples string
	FieldType      string
	DocLines       []string
	// The best field to show in a oneof related samples.
	// Non deprecated fields are preferred, then scalar, repeated, map fields
	// in that order.
	ExampleField *api.Field
	// If set, this enum is only enabled when some features are enabled.
	FeatureGates   []string
	FeatureGatesOp string
}

type fieldAnnotations struct {
	// In Rust, message fields are fields inside a struct. These must be
	// `snake_case`. Possibly mangled with `r#` if the name is a Rust reserved
	// word.
	FieldName string
	// In Rust, each fields gets a `set_{{FieldName}}` setter. These must be
	// `snake_case`, but are never mangled with a `r#` prefix.
	SetterName string
	// In Rust, fields that appear in a OneOf also appear as a enum branch.
	// These must be in `PascalCase`.
	BranchName string
	// The fully qualified name of the containing message.
	FQMessageName      string
	DocLines           []string
	FieldType          string
	PrimitiveFieldType string
	AddQueryParameter  string
	// For fields that are maps, these are the type of the key and value,
	// respectively.
	KeyType    string
	KeyField   *api.Field
	ValueType  string
	ValueField *api.Field
	// The templates need to generate different code for boxed fields.
	IsBoxed bool
	// If true, it requires a serde_with::serde_as() transformation.
	SerdeAs string
	// If true, use `wkt::internal::is_default()` to skip the field
	SkipIfIsDefault bool
	// If true, this is a `wkt::Value` field, and requires super-extra custom
	// deserialization.
	IsWktValue bool
	// If true, this is a `wkt::NullValue` field, and also requires super-extra
	// custom deserialization.
	IsWktNullValue bool
	// Some fields may be the type of the message they are defined in.
	// We need to know this in sample generation to avoid importing
	// the parent type twice.
	// This applies to single value, repeated and map fields.
	FieldTypeIsParentType bool
	// In some cases, for instance, for OpenApi and Discovery synthetic requests,
	// types in different namespaces have the same name. We need to import those
	// with aliases.
	AliasInExamples string
	// If this field is part of a oneof group, this will contain the other fields
	// in the group.
	OtherFieldsInGroup []*api.Field
}

// SkipIfIsEmpty returns true if the field should be skipped if it is empty.
func (a *fieldAnnotations) SkipIfIsEmpty() bool {
	return !a.SkipIfIsDefault
}

// RequiresSerdeAs returns true if the field requires a serde_as annotation.
func (a *fieldAnnotations) RequiresSerdeAs() bool {
	return a.SerdeAs != ""
}

type enumAnnotation struct {
	Name        string
	ModuleName  string
	DocLines    []string
	UniqueNames []*api.EnumValue
	// The fully qualified name, including the `codec.modulePath`
	// (typically `crate::model::`) prefix. For external enums this is prefixed
	// by the external crate name.
	QualifiedName string
	// The fully qualified name, relative to `codec.modulePath`. Typically this
	// is the `QualifiedName` with the `crate::model::` prefix removed.
	RelativeName string
	// The fully qualified name for examples. For messages in external packages
	// this is basically `QualifiedName`. For messages in the current package
	// this includes `modelAnnotations.PackageName`.
	NameInExamples string
	// There's a missmatch between the sidekick model representation of wkt::NullValue
	// and the representation in Rust. We us this for sample generation.
	IsWktNullValue bool
	// These are some of the enum values that can be used in examples,
	// accompanied by an index to facilitate generating code that can
	// distinctly reference each value. These attempt to avoid deprecated
	// and the default values. Contains at most 3 elements. Will be empty iff
	// the enum has no values.
	ValuesForExamples []*enumValueForExamples
	// If set, this enum is only enabled when some features are enabled
	FeatureGates   []string
	FeatureGatesOp string
}

type enumValueAnnotation struct {
	Name              string
	VariantName       string
	EnumType          string
	DocLines          []string
	SerializeAsString bool
}

type enumValueForExamples struct {
	EnumValue *api.EnumValue
	Index     int
}

// annotateModel creates a struct used as input for Mustache templates.
// Fields and methods defined in this struct directly correspond to Mustache
// tags. For example, the Mustache tag {{#Services}} uses the
// [Template.Services] field.
func annotateModel(model *api.API, codec *codec) *modelAnnotations {
	codec.hasServices = len(model.State.ServiceByID) > 0

	resolveUsedPackages(model, codec.extraPackages)
	// Annotate enums and messages that we intend to generate. In the
	// process we discover the external dependencies and trim the list of
	// packages used by this API.
	// This API's enums and messages get full annotations.
	for _, e := range model.Enums {
		codec.annotateEnum(e, model, true)
	}
	for _, m := range model.Messages {
		codec.annotateMessage(m, model, true)
	}
	// External enums and messages get only basic annotations
	// used for sample generation.
	// External enums and messages are the ones that have yet
	// to be annotated.
	for _, e := range model.State.EnumByID {
		if e.Codec == nil {
			codec.annotateEnum(e, model, false)
		}
	}
	for _, m := range model.State.MessageByID {
		if m.Codec == nil {
			codec.annotateMessage(m, model, false)
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
			codec.annotateMethod(m)
			if m := m.InputType; m != nil {
				codec.annotateMessage(m, model, true)
			}
			if m := m.OutputType; m != nil {
				codec.annotateMessage(m, model, true)
			}
		}
		codec.annotateService(s)
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
	ann := &modelAnnotations{
		PackageName:      codec.packageName(model),
		PackageNamespace: codec.packageNamespace(model),
		PackageVersion:   codec.version,
		ReleaseLevel:     codec.releaseLevel,
		RequiredPackages: requiredPackages(codec.extraPackages),
		ExternPackages:   externPackages(codec.extraPackages),
		HasLROs:          hasLROs,
		CopyrightYear:    codec.generationYear,
		BoilerPlate: append(license.LicenseHeaderBulk(),
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
		DetailedTracingAttributes: codec.detailedTracingAttributes,
	}

	codec.addFeatureAnnotations(model, ann)

	model.Codec = ann
	return ann
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
			enum, ok := model.State.EnumByID[id]
			// Some messages are not annotated (e.g. external messages).
			if !ok || enum.Codec == nil {
				continue
			}
			annotation := enum.Codec.(*enumAnnotation)
			annotation.FeatureGates = append(annotation.FeatureGates, svcAnn.FeatureName())
			slices.Sort(annotation.FeatureGates)
			annotation.FeatureGatesOp = "any"
		}
		for _, id := range deps.Messages {
			msg, ok := model.State.MessageByID[id]
			// Some messages are not annotated (e.g. external messages).
			if !ok || msg.Codec == nil {
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
	for _, msg := range model.State.MessageByID {
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
	for _, enum := range model.State.EnumByID {
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

func (c *codec) annotateService(s *api.Service) {
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
				lroTypes = append(lroTypes, s.Model.State.MessageByID[m.OperationInfo.MetadataTypeID])
			}
			if _, ok := seenLROTypes[m.OperationInfo.ResponseTypeID]; !ok {
				seenLROTypes[m.OperationInfo.ResponseTypeID] = true
				lroTypes = append(lroTypes, s.Model.State.MessageByID[m.OperationInfo.ResponseTypeID])
			}
		}
	}
	serviceName := c.ServiceName(s)
	moduleName := toSnake(serviceName)
	ann := &serviceAnnotations{
		Name:              toPascal(serviceName),
		PackageModuleName: packageToModuleName(s.Package),
		ModuleName:        moduleName,
		DocLines: c.formatDocComments(
			s.Documentation, s.ID, s.Model.State, []string{s.ID, s.Package}),
		Methods:                   methods,
		DefaultHost:               s.DefaultHost,
		LROTypes:                  lroTypes,
		APITitle:                  s.Model.Title,
		PerServiceFeatures:        c.perServiceFeatures,
		HasVeneer:                 c.hasVeneer,
		Incomplete:                slices.ContainsFunc(s.Methods, func(m *api.Method) bool { return !c.generateMethod(m) }),
		DetailedTracingAttributes: c.detailedTracingAttributes,
	}
	s.Codec = ann
}

// annotateMessage annotates the message with basic or full annotations.
// When fully annotating a message, its fields, its nested messages, and its nested enums
// are also annotated.
// Basic annotations are useful for annotating external messages with information used in samples.
func (c *codec) annotateMessage(m *api.Message, model *api.API, full bool) {
	qualifiedName := fullyQualifiedMessageName(m, c.modulePath, model.PackageName, c.packageMapping)
	relativeName := strings.TrimPrefix(qualifiedName, c.modulePath+"::")
	nameInExamples := c.nameInExamplesFromQualifiedName(qualifiedName, model)
	annotations := &messageAnnotation{
		Name:              toPascal(m.Name),
		ModuleName:        toSnake(m.Name),
		QualifiedName:     qualifiedName,
		RelativeName:      relativeName,
		NameInExamples:    nameInExamples,
		PackageModuleName: packageToModuleName(m.Package),
		SourceFQN:         strings.TrimPrefix(m.ID, "."),
	}
	m.Codec = annotations

	if !full {
		// We have basic annotations, we are done.
		return
	}

	for _, f := range m.Fields {
		c.annotateField(f, m, model)
	}
	for _, o := range m.OneOfs {
		c.annotateOneOf(o, m, model)
	}
	for _, e := range m.Enums {
		c.annotateEnum(e, model, true)
	}
	for _, child := range m.Messages {
		c.annotateMessage(child, model, true)
	}
	basicFields := language.FilterSlice(m.Fields, func(f *api.Field) bool {
		return !f.IsOneOf
	})

	annotations.DocLines = c.formatDocComments(m.Documentation, m.ID, model.State, m.Scopes())
	annotations.HasNestedTypes = language.HasNestedTypes(m)
	annotations.BasicFields = basicFields
	annotations.Internal = slices.Contains(c.internalTypes, m.ID)
}

func (c *codec) annotateMethod(m *api.Method) {
	c.annotatePathInfo(m)
	for _, routing := range m.Routing {
		for _, variant := range routing.Variants {
			routingVariantAnnotations := &routingVariantAnnotations{
				FieldAccessors:   c.annotateRoutingAccessors(variant, m),
				PrefixSegments:   annotateSegments(variant.Prefix.Segments),
				MatchingSegments: annotateSegments(variant.Matching.Segments),
				SuffixSegments:   annotateSegments(variant.Suffix.Segments),
			}
			variant.Codec = routingVariantAnnotations
		}
	}
	returnType := c.methodInOutTypeName(m.OutputTypeID, m.Model.State, m.Model.PackageName)
	if m.ReturnsEmpty {
		returnType = "()"
	}
	serviceName := c.ServiceName(m.Service)
	annotation := &methodAnnotation{
		Name:                      toSnake(m.Name),
		NameNoMangling:            toSnakeNoMangling(m.Name),
		BuilderName:               toPascal(m.Name),
		Body:                      bodyAccessor(m),
		DocLines:                  c.formatDocComments(m.Documentation, m.ID, m.Model.State, m.Service.Scopes()),
		PathInfo:                  m.PathInfo,
		ServiceNameToPascal:       toPascal(serviceName),
		ServiceNameToCamel:        toCamel(serviceName),
		ServiceNameToSnake:        toSnake(serviceName),
		SystemParameters:          c.systemParameters,
		ReturnType:                returnType,
		HasVeneer:                 c.hasVeneer,
		RoutingRequired:           c.routingRequired,
		DetailedTracingAttributes: c.detailedTracingAttributes,
	}
	if annotation.Name == "clone" {
		// Some methods look too similar to standard Rust traits. Clippy makes
		// a recommendation that is not applicable to generated code.
		annotation.Attributes = []string{"#[allow(clippy::should_implement_trait)]"}
	}
	if m.OperationInfo != nil {
		metadataType := c.methodInOutTypeName(m.OperationInfo.MetadataTypeID, m.Model.State, m.Model.PackageName)
		responseType := c.methodInOutTypeName(m.OperationInfo.ResponseTypeID, m.Model.State, m.Model.PackageName)
		m.OperationInfo.Codec = &operationInfo{
			MetadataType:     metadataType,
			ResponseType:     responseType,
			PackageNamespace: c.packageNamespace(m.Model),
		}
	}
	if m.DiscoveryLro != nil {
		lroAnnotation := &discoveryLroAnnotations{
			MethodName: annotation.Name,
			ReturnType: returnType,
		}
		for _, p := range m.DiscoveryLro.PollingPathParameters {
			a := discoveryLroPathParameter{
				Name:       toSnake(p),
				SetterName: toSnakeNoMangling(p),
			}
			lroAnnotation.PollingPathParameters = append(lroAnnotation.PollingPathParameters, a)
		}
		m.DiscoveryLro.Codec = lroAnnotation
	}
	m.Codec = annotation
}

func (c *codec) annotateRoutingAccessors(variant *api.RoutingInfoVariant, m *api.Method) []string {
	return makeAccessors(variant.FieldPath, m)
}

func makeAccessors(fields []string, m *api.Method) []string {
	findField := func(name string, message *api.Message) *api.Field {
		for _, f := range message.Fields {
			if f.Name == name {
				return f
			}
		}
		return nil
	}
	var accessors []string
	message := m.InputType
	for _, name := range fields {
		field := findField(name, message)
		rustFieldName := toSnake(name)
		if field == nil {
			slog.Error("invalid routing/path field for request message", "field", rustFieldName, "message ID", message.ID)
			continue
		}
		if field.Optional {
			accessors = append(accessors, fmt.Sprintf(".and_then(|m| m.%s.as_ref())", rustFieldName))
		} else {
			accessors = append(accessors, fmt.Sprintf(".map(|m| &m.%s)", rustFieldName))
		}
		if field.Typez == api.STRING_TYPE {
			accessors = append(accessors, ".map(|s| s.as_str())")
		}
		if field.Typez == api.MESSAGE_TYPE {
			if fieldMessage, ok := m.Model.State.MessageByID[field.TypezID]; ok {
				message = fieldMessage
			}
		}
	}
	return accessors
}

func annotateSegments(segments []string) []string {
	var ann []string
	// The model may have multiple consecutive literal segments. We use this
	// buffer to consolidate them into a single literal segment.
	literalBuffer := ""
	flushBuffer := func() {
		if literalBuffer != "" {
			ann = append(ann, fmt.Sprintf(`Segment::Literal("%s")`, literalBuffer))
		}
		literalBuffer = ""
	}
	for index, segment := range segments {
		switch segment {
		case api.MultiSegmentWildcard:
			flushBuffer()
			if len(segments) == 1 {
				ann = append(ann, "Segment::MultiWildcard")
			} else if len(segments) != index+1 {
				ann = append(ann, "Segment::MultiWildcard")
			} else {
				ann = append(ann, "Segment::TrailingMultiWildcard")
			}
		case api.SingleSegmentWildcard:
			if index != 0 {
				literalBuffer += "/"
			}
			flushBuffer()
			ann = append(ann, "Segment::SingleWildcard")
		default:
			if index != 0 {
				literalBuffer += "/"
			}
			literalBuffer += segment
		}
	}
	flushBuffer()
	return ann
}

func makeBindingSubstitution(v *api.PathVariable, m *api.Method) bindingSubstitution {
	fieldAccessor := "Some(&req)"
	for _, a := range makeAccessors(v.FieldPath, m) {
		fieldAccessor += a
	}
	var rustNames []string
	for _, n := range v.FieldPath {
		rustNames = append(rustNames, toSnake(n))
	}
	return bindingSubstitution{
		FieldAccessor: fieldAccessor,
		FieldName:     strings.Join(rustNames, "."),
		Template:      v.Segments,
	}
}

func (c *codec) annotatePathBinding(b *api.PathBinding, m *api.Method) *pathBindingAnnotation {
	var subs []*bindingSubstitution
	for _, s := range b.PathTemplate.Segments {
		if s.Variable != nil {
			sub := makeBindingSubstitution(s.Variable, m)
			subs = append(subs, &sub)
		}
	}
	return &pathBindingAnnotation{
		PathFmt:                   httpPathFmt(b.PathTemplate),
		QueryParams:               language.QueryParams(m, b),
		Substitutions:             subs,
		DetailedTracingAttributes: c.detailedTracingAttributes,
	}
}

// annotatePathInfo annotates the `PathInfo` and all of its `PathBinding`s.
func (c *codec) annotatePathInfo(m *api.Method) {
	seen := make(map[string]bool)
	var uniqueParameters []*bindingSubstitution

	for _, b := range m.PathInfo.Bindings {
		ann := c.annotatePathBinding(b, m)

		// We need to keep track of unique path parameters to support
		// implicit routing over gRPC. This is go/aip/4222.
		for _, s := range ann.Substitutions {
			if _, ok := seen[s.FieldName]; !ok {
				uniqueParameters = append(uniqueParameters, s)
				seen[s.FieldName] = true
			}
		}

		// Annotate the `PathBinding`
		b.Codec = ann
	}

	// Annotate the `PathInfo`
	m.PathInfo.Codec = &pathInfoAnnotation{
		HasBody:          m.PathInfo.BodyFieldPath != "",
		UniqueParameters: uniqueParameters,
		IsIdempotent:     isIdempotent(m.PathInfo),
	}
}

func (c *codec) annotateOneOf(oneof *api.OneOf, message *api.Message, model *api.API) {
	scope := messageScopeName(message, "", c.modulePath, model.PackageName, c.packageMapping)
	enumName := c.OneOfEnumName(oneof)
	qualifiedName := fmt.Sprintf("%s::%s", scope, enumName)
	relativeEnumName := strings.TrimPrefix(qualifiedName, c.modulePath+"::")
	structQualifiedName := fullyQualifiedMessageName(message, c.modulePath, model.PackageName, c.packageMapping)
	nameInExamples := c.nameInExamplesFromQualifiedName(qualifiedName, model)

	bestField := slices.MaxFunc(oneof.Fields, func(f1 *api.Field, f2 *api.Field) int {
		if f1.Deprecated == f2.Deprecated {
			if f1.Map == f2.Map {
				if f1.Repeated == f2.Repeated {
					if f1.MessageType != nil && f2.MessageType == nil {
						return -1
					} else if f1.MessageType == nil && f2.MessageType != nil {
						return 1
					} else {
						return 0
					}
				} else if f1.Repeated {
					return -1
				} else {
					return 1
				}
			} else if f1.Map {
				return -1
			} else {
				return 1
			}
		} else if f1.Deprecated {
			return -1
		} else {
			return 1
		}
	})

	oneof.Codec = &oneOfAnnotation{
		FieldName:           toSnake(oneof.Name),
		SetterName:          toSnakeNoMangling(oneof.Name),
		EnumName:            enumName,
		QualifiedName:       qualifiedName,
		RelativeName:        relativeEnumName,
		StructQualifiedName: structQualifiedName,
		NameInExamples:      nameInExamples,
		FieldType:           fmt.Sprintf("%s::%s", scope, enumName),
		DocLines:            c.formatDocComments(oneof.Documentation, oneof.ID, model.State, message.Scopes()),
		ExampleField:        bestField,
	}
}

func (c *codec) primitiveSerdeAs(field *api.Field) string {
	switch field.Typez {
	case api.INT32_TYPE, api.SFIXED32_TYPE, api.SINT32_TYPE:
		return "wkt::internal::I32"
	case api.INT64_TYPE, api.SFIXED64_TYPE, api.SINT64_TYPE:
		return "wkt::internal::I64"
	case api.UINT32_TYPE, api.FIXED32_TYPE:
		return "wkt::internal::U32"
	case api.UINT64_TYPE, api.FIXED64_TYPE:
		return "wkt::internal::U64"
	case api.FLOAT_TYPE:
		return "wkt::internal::F32"
	case api.DOUBLE_TYPE:
		return "wkt::internal::F64"
	case api.BYTES_TYPE:
		if c.bytesUseUrlSafeAlphabet {
			return "serde_with::base64::Base64<serde_with::base64::UrlSafe>"
		}
		return "serde_with::base64::Base64"
	default:
		return ""
	}
}

func (c *codec) mapKeySerdeAs(field *api.Field) string {
	if field.Typez == api.BOOL_TYPE {
		return "serde_with::DisplayFromStr"
	}
	return c.primitiveSerdeAs(field)
}

func (c *codec) mapValueSerdeAs(field *api.Field) string {
	if field.Typez == api.MESSAGE_TYPE {
		return c.messageFieldSerdeAs(field)
	}
	return c.primitiveSerdeAs(field)
}

func (c *codec) messageFieldSerdeAs(field *api.Field) string {
	switch field.TypezID {
	case ".google.protobuf.BytesValue":
		if c.bytesUseUrlSafeAlphabet {
			return "serde_with::base64::Base64<serde_with::base64::UrlSafe>"
		}
		return "serde_with::base64::Base64"
	case ".google.protobuf.UInt64Value":
		return "wkt::internal::U64"
	case ".google.protobuf.Int64Value":
		return "wkt::internal::I64"
	case ".google.protobuf.UInt32Value":
		return "wkt::internal::U32"
	case ".google.protobuf.Int32Value":
		return "wkt::internal::I32"
	case ".google.protobuf.FloatValue":
		return "wkt::internal::F32"
	case ".google.protobuf.DoubleValue":
		return "wkt::internal::F64"
	case ".google.protobuf.BoolValue":
		return ""
	default:
		return ""
	}
}

func (c *codec) annotateField(field *api.Field, message *api.Message, model *api.API) {
	ann := &fieldAnnotations{
		FieldName:          toSnake(field.Name),
		SetterName:         toSnakeNoMangling(field.Name),
		FQMessageName:      fullyQualifiedMessageName(message, c.modulePath, model.PackageName, c.packageMapping),
		BranchName:         toPascal(field.Name),
		DocLines:           c.formatDocComments(field.Documentation, field.ID, model.State, message.Scopes()),
		FieldType:          fieldType(field, model.State, false, c.modulePath, model.PackageName, c.packageMapping),
		PrimitiveFieldType: fieldType(field, model.State, true, c.modulePath, model.PackageName, c.packageMapping),
		AddQueryParameter:  addQueryParameter(field),
		SerdeAs:            c.primitiveSerdeAs(field),
		SkipIfIsDefault:    field.Typez != api.STRING_TYPE && field.Typez != api.BYTES_TYPE,
		IsWktValue:         field.Typez == api.MESSAGE_TYPE && field.TypezID == ".google.protobuf.Value",
		IsWktNullValue:     field.Typez == api.ENUM_TYPE && field.TypezID == ".google.protobuf.NullValue",
	}
	if field.Recursive || (field.Typez == api.MESSAGE_TYPE && field.IsOneOf) {
		ann.IsBoxed = true
	}
	field.Codec = ann
	if field.Typez == api.MESSAGE_TYPE {
		if msg, ok := model.State.MessageByID[field.TypezID]; ok && msg.IsMap {
			if len(msg.Fields) != 2 {
				slog.Error("expected exactly two fields for map message", "field ID", field.ID, "map ID", field.TypezID)
			}
			ann.KeyField = msg.Fields[0]
			ann.KeyType = mapType(msg.Fields[0], model.State, c.modulePath, model.PackageName, c.packageMapping)
			ann.ValueField = msg.Fields[1]
			ann.ValueType = mapType(msg.Fields[1], model.State, c.modulePath, model.PackageName, c.packageMapping)
			key := c.mapKeySerdeAs(msg.Fields[0])
			value := c.mapValueSerdeAs(msg.Fields[1])
			if key != "" || value != "" {
				if key == "" {
					key = "serde_with::Same"
				}
				if value == "" {
					value = "serde_with::Same"
				}
				ann.SerdeAs = fmt.Sprintf("std::collections::HashMap<%s, %s>", key, value)
			}
		} else {
			ann.SerdeAs = c.messageFieldSerdeAs(field)
		}
	}
	if field.Group != nil {
		ann.OtherFieldsInGroup = language.FilterSlice(field.Group.Fields, func(f *api.Field) bool { return field != f })
	}
	ann.FieldTypeIsParentType = (field.MessageType == message || // Single or repeated field whose type is the same as the containing type.
		// Map field whose value type is the same as the conaining type.
		(ann.ValueField != nil && ann.ValueField.MessageType == message))
	if !ann.FieldTypeIsParentType && // When the type of the field is the same as the containing type we don't import twice. No alias needed.
		// Single or repeated field whose type unqualified name is the same as the containing message's.
		((field.MessageType != nil && field.MessageType.Name == message.Name) ||
			// Map field whose unqualified type unqualified name is the same as the containing message's.
			(ann.ValueField != nil && ann.ValueField.MessageType != nil && ann.ValueField.MessageType.Name == message.Name)) {
		ann.AliasInExamples = toPascal(field.Name)
	}
}

func (c *codec) annotateEnum(e *api.Enum, model *api.API, full bool) {
	for _, ev := range e.Values {
		c.annotateEnumValue(ev, model, full)
	}

	qualifiedName := fullyQualifiedEnumName(e, c.modulePath, model.PackageName, c.packageMapping)
	relativeName := strings.TrimPrefix(qualifiedName, c.modulePath+"::")
	nameInExamples := c.nameInExamplesFromQualifiedName(qualifiedName, model)

	// For BigQuery (and so far only BigQuery), the enum values conflict when
	// converted to the Rust style [1]. Basically, there are several enum values
	// in this service that differ only in case, such as `FULL` vs. `full`.
	//
	// We create a list with the duplicates removed to avoid conflicts in the
	// generated code.
	//
	// [1]: Both Rust and Protobuf use `SCREAMING_SNAKE_CASE` for these, but
	//      some services do not follow the Protobuf convention.
	seen := map[string]*api.EnumValue{}
	var unique []*api.EnumValue
	for _, ev := range e.Values {
		name := enumValueVariantName(ev)
		if existing, ok := seen[name]; ok {
			if existing.Number != ev.Number {
				slog.Warn("conflicting names for enum values", "enum.ID", e.ID)
			}
		} else {
			unique = append(unique, ev)
			seen[name] = ev
		}
	}

	// We try to pick some good enum values to show in examples.
	// - We pick from the already computed unique values, even though that applies to BigQuery only.
	// - We pick values that are not deprecated.
	// - We don't pick the default value.
	goodValues := language.FilterSlice(unique, func(ev *api.EnumValue) bool {
		return !ev.Deprecated && ev.Number != 0
	})
	// If we couldn't find any good enum values for examples, then we pick from all enum values.
	if len(goodValues) == 0 {
		goodValues = unique
	}
	// We pick at most 3 values as samples do not need to be exhaustive.
	goodValues = goodValues[:min(3, len(goodValues))]

	i := -1
	forExamples := language.MapSlice(goodValues, func(ev *api.EnumValue) *enumValueForExamples {
		i++
		return &enumValueForExamples{
			EnumValue: ev,
			Index:     i,
		}
	})

	annotations := &enumAnnotation{
		Name:              enumName(e),
		ModuleName:        toSnake(enumName(e)),
		QualifiedName:     qualifiedName,
		RelativeName:      relativeName,
		NameInExamples:    nameInExamples,
		IsWktNullValue:    nameInExamples == "wkt::NullValue",
		ValuesForExamples: forExamples,
	}
	e.Codec = annotations

	if !full {
		// We have basic annotations, we are done.
		return
	}

	annotations.DocLines = c.formatDocComments(e.Documentation, e.ID, model.State, e.Scopes())
	annotations.UniqueNames = unique
}

func (c *codec) annotateEnumValue(ev *api.EnumValue, model *api.API, full bool) {
	annotations := &enumValueAnnotation{
		Name:              enumValueName(ev),
		EnumType:          enumName(ev.Parent),
		VariantName:       enumValueVariantName(ev),
		SerializeAsString: c.serializeEnumsAsStrings,
	}
	ev.Codec = annotations

	if !full {
		// We have basic annotations, we are done.
		return
	}

	annotations.DocLines = c.formatDocComments(ev.Documentation, ev.ID, model.State, ev.Scopes())
}

// isIdempotent returns "true" if the method is idempotent by default, and "false", if not.
func isIdempotent(p *api.PathInfo) string {
	if len(p.Bindings) == 0 {
		return "false"
	}
	for _, b := range p.Bindings {
		if b.Verb == "POST" || b.Verb == "PATCH" {
			return "false"
		}
	}
	return "true"
}
