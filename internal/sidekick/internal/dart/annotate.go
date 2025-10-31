// Copyright 2025 Google LLC
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

package dart

import (
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"sort"
	"strconv"
	"strings"

	"github.com/googleapis/librarian/internal/sidekick/internal/api"
	"github.com/googleapis/librarian/internal/sidekick/internal/language"
	"github.com/googleapis/librarian/internal/sidekick/internal/license"
	"github.com/iancoleman/strcase"
)

var omitGeneration = map[string]string{
	".google.longrunning.Operation": "",
	".google.protobuf.Value":        "",
}

type modelAnnotations struct {
	Parent *api.API
	// The Dart package name (e.g. google_cloud_secretmanager).
	PackageName string
	// The version of the generated package.
	PackageVersion string
	// Name of the API in snake_format (e.g. secretmanager).
	MainFileName      string
	SourcePackageName string
	CopyrightYear     string
	BoilerPlate       []string
	DefaultHost       string
	DocLines          []string
	// A reference to an optional hand-written part file.
	PartFileReference          string
	PackageDependencies        []packageDependency
	Imports                    []string
	DevDependencies            []string
	DoNotPublish               bool
	RepositoryURL              string
	ReadMeAfterTitleText       string
	ReadMeQuickstartText       string
	IssueTrackerURL            string
	ApiKeyEnvironmentVariables []string
	// Dart `export` statements e.g.
	// ["export 'package:google_cloud_gax/gax.dart' show Any", "export 'package:google_cloud_gax/gax.dart' show Status"]
	Exports     []string
	ProtoPrefix string
}

// HasServices returns true if the model has services.
func (m *modelAnnotations) HasServices() bool {
	return len(m.Parent.Services) > 0
}

// HasDependencies returns true if the model has package dependencies.
func (m *modelAnnotations) HasDependencies() bool {
	return len(m.PackageDependencies) > 0
}

// HasDevDependencies returns whether the generated package specified any dev_dependencies.
func (m *modelAnnotations) HasDevDependencies() bool {
	return len(m.DevDependencies) > 0
}

type serviceAnnotations struct {
	// The service name using Dart naming conventions.
	Name        string
	DocLines    []string
	Methods     []*api.Method
	FieldName   string
	StructName  string
	DefaultHost string
}

type messageAnnotation struct {
	Parent         *api.Message
	Name           string
	QualifiedName  string
	DocLines       []string
	OmitGeneration bool
	// A custom body for the message's constructor.
	ConstructorBody string
	ToStringLines   []string
	Model           *api.API
}

// HasFields returns true if the message has fields.
func (m *messageAnnotation) HasFields() bool {
	return len(m.Parent.Fields) > 0
}

// HasCustomEncoding returns true if the message has custom encoding.
func (m *messageAnnotation) HasCustomEncoding() bool {
	_, hasCustomEncoding := usesCustomEncoding[m.Parent.ID]
	return hasCustomEncoding
}

// HasToStringLines returns true if the message has toString lines.
func (m *messageAnnotation) HasToStringLines() bool {
	return len(m.ToStringLines) > 0
}

type methodAnnotation struct {
	Parent *api.Method
	// The method name using Dart naming conventions.
	Name                string
	RequestMethod       string
	RequestType         string
	ResponseType        string
	DocLines            []string
	ReturnsValue        bool
	BodyMessageName     string
	QueryLines          []string
	IsLROGetOperation   bool
	ServerSideStreaming bool // Whether the server supports streaming via server-sent events (SSE).
}

// HasBody returns true if the method has a body.
func (m *methodAnnotation) HasBody() bool {
	return m.Parent.PathInfo.BodyFieldPath != ""
}

// HasQueryLines returns true if the method has query lines.
func (m *methodAnnotation) HasQueryLines() bool {
	return len(m.QueryLines) > 0
}

type pathInfoAnnotation struct {
	PathFmt string
}

type oneOfAnnotation struct {
	Name     string
	DocLines []string
}

type operationInfoAnnotation struct {
	ResponseType string
	MetadataType string
}

type fieldAnnotation struct {
	Name                  string
	Type                  string
	DocLines              []string
	Required              bool
	Nullable              bool
	FieldBehaviorRequired bool
	DefaultValue          string
	FromJson              string
	ToJson                string
}

type enumAnnotation struct {
	Name         string
	DocLines     []string
	DefaultValue string
	Model        *api.API
}

type enumValueAnnotation struct {
	Name     string
	DocLines []string
}

type packageDependency struct {
	Name       string
	Constraint string
}

type annotateModel struct {
	// The API model we're annotating.
	model *api.API
	// Mappings from IDs to types.
	state *api.APIState
	// The set of required imports (e.g. "package:google_cloud_type/type.dart" or
	// "package:http/http.dart as http") that have been calculated.
	//
	// The keys of this map are used to determine what imports to include
	// in the generated Dart code and what dependencies to include in
	// pubspec.yaml.
	//
	// Every import must have a corresponding entry in .sidekick.toml to specify
	// its version constraints.
	imports map[string]bool
	// The mapping from protobuf packages to Dart import statements.
	packageMapping map[string]string
	// The protobuf packages that need to be imported with prefixes.
	packagePrefixes map[string]string
	// A mapping from a package name (e.g. "http") to its version constraint (e.g. "^1.3.0").
	dependencyConstraints map[string]string
}

func newAnnotateModel(model *api.API) *annotateModel {
	return &annotateModel{
		model:                 model,
		state:                 model.State,
		imports:               map[string]bool{},
		packageMapping:        map[string]string{},
		packagePrefixes:       map[string]string{},
		dependencyConstraints: map[string]string{},
	}
}

// annotateModel creates a struct used as input for Mustache templates.
// Fields and methods defined in this struct directly correspond to Mustache
// tags. For example, the Mustache tag {{#Services}} uses the
// [Template.Services] field.
func (annotate *annotateModel) annotateModel(options map[string]string) error {
	var (
		packageNameOverride        string
		generationYear             string
		packageVersion             string
		partFileReference          string
		doNotPublish               bool
		dependencies               = []string{}
		devDependencies            = []string{}
		repositoryURL              string
		readMeAfterTitleText       string
		readMeQuickstartText       string
		issueTrackerURL            string
		apiKeyEnvironmentVariables = []string{}
		exports                    = []string{}
		protobufPrefix             string
		pkgName                    string
	)

	for key, definition := range options {
		switch {
		case key == "api-keys-environment-variables":
			// api-keys-environment-variables = "GOOGLE_API_KEY,GEMINI_API_KEY"
			// A comma-separated list of environment variables to look for searching for
			// a API key.
			apiKeyEnvironmentVariables = strings.Split(definition, ",")
			for i := range apiKeyEnvironmentVariables {
				apiKeyEnvironmentVariables[i] = strings.TrimSpace(apiKeyEnvironmentVariables[i])
			}
		case key == "package-name-override":
			packageNameOverride = definition
		case key == "copyright-year":
			generationYear = definition
		case key == "issue-tracker-url":
			// issue-tracker-url = "http://www.example.com/issues"
			// A link to the issue tracker for the service.
			issueTrackerURL = definition
		case key == "version":
			packageVersion = definition
		case key == "part-file":
			partFileReference = definition
		case key == "extra-exports":
			// extra-export = "export 'package:google_cloud_gax/gax.dart' show Any; export 'package:google_cloud_gax/gax.dart' show Status;"
			// Dart `export` statements that should be appended after any imports.
			exports = strings.FieldsFunc(definition, func(c rune) bool { return c == ';' })
			for i := range exports {
				exports[i] = strings.TrimSpace(exports[i])
			}
		case key == "dependencies":
			// dependencies = "http, googleapis_auth"
			// A list of dependencies to add to pubspec.yaml. This can be used to add dependencies for hand-written code.
			dependencies = strings.Split(definition, ",")
			for i := range dependencies {
				dependencies[i] = strings.TrimSpace(dependencies[i])
			}
		case key == "dev-dependencies":
			devDependencies = strings.Split(definition, ",")
		case key == "not-for-publication":
			value, err := strconv.ParseBool(definition)
			if err != nil {
				return fmt.Errorf(
					"cannot convert `not-for-publication` value %q to boolean: %w",
					definition,
					err,
				)
			}
			doNotPublish = value
		case key == "readme-after-title-text":
			// Markdown that will be inserted into the README.md after the title section.
			readMeAfterTitleText = definition
		case key == "readme-quickstart-text":
			// Markdown that will appear as a "Quickstart" section of README.md. Does not include
			// the section title, i.e., you probably want it to start with "## Getting Started"`
			// or similar.
			readMeQuickstartText = definition
		case key == "repository-url":
			repositoryURL = definition
		case strings.HasPrefix(key, "proto:"):
			// "proto:google.protobuf" = "package:google_cloud_protobuf/protobuf.dart"
			keys := strings.Split(key, ":")
			if len(keys) != 2 {
				return fmt.Errorf("key should be in the format proto:<proto-package>, got=%q", key)
			}
			protoPackage := keys[1]
			annotate.packageMapping[protoPackage] = definition
		case strings.HasPrefix(key, "prefix:"):
			// 'prefix:google.protobuf' = 'protobuf'
			keys := strings.Split(key, ":")
			if len(keys) != 2 {
				return fmt.Errorf("key should be in the format prefix:<proto-package>, got=%q", key)
			}
			protoPackage := keys[1]
			annotate.packagePrefixes[protoPackage] = definition
		case strings.HasPrefix(key, "package:"):
			// Version constraints for a package.
			//
			// Expressed as: 'package:<package name>' = '<version constraint>'
			// For example: 'package:http' = '^1.3.0'
			//
			// If the package is needed as a dependency, then this constract is used.
			annotate.dependencyConstraints[strings.TrimPrefix(key, "package:")] = definition
		}
	}

	// Register any missing WKTs.
	registerMissingWkt(annotate.state)

	model := annotate.model

	// Traverse and annotate the enums defined in this API.
	for _, e := range model.Enums {
		annotate.annotateEnum(e)
	}

	// Traverse and annotate the messages defined in this API.
	for _, m := range model.Messages {
		annotate.annotateMessage(m)
	}

	for _, s := range model.Services {
		annotate.annotateService(s)
	}

	// Remove our package self-reference.
	delete(annotate.imports, model.PackageName)

	// Add a dev dependency on package:lints.
	devDependencies = append(devDependencies, "lints")

	// Add the import for ServiceClient and related functionality.
	if len(model.Services) > 0 {
		annotate.imports[serviceClientImport] = true
	}

	// `google.protobuf` defines `JsonEncodable`, which is needed by any package that defines a
	// `message` or `enum`, i.e., all of them.
	protobufPrefix = annotate.packagePrefixes["google.protobuf"]
	if protobufPrefix == "" {
		annotate.imports[protobufImport] = true
	} else {
		annotate.imports[protobufImport+" as "+protobufPrefix] = true
		protobufPrefix += "."
	}

	if len(model.Services) > 0 && len(apiKeyEnvironmentVariables) == 0 {
		return errors.New("all packages that define a service must define 'api-keys-environment-variables'")
	}

	if issueTrackerURL == "" {
		return errors.New("all packages must define 'issue-tracker-url'")
	}

	pkgName = packageName(model, packageNameOverride)
	importedPackages := calculatePubPackages(annotate.imports)
	for _, d := range dependencies {
		importedPackages[d] = true
	}

	packageDependencies, err := calculateDependencies(importedPackages, annotate.dependencyConstraints, pkgName)
	if err != nil {
		return err
	}

	ann := &modelAnnotations{
		Parent:         model,
		PackageName:    pkgName,
		PackageVersion: packageVersion,
		MainFileName:   strcase.ToSnake(model.Name),
		CopyrightYear:  generationYear,
		BoilerPlate: append(license.LicenseHeaderBulk(),
			"",
			" Code generated by sidekick. DO NOT EDIT."),
		DefaultHost: func() string {
			if len(model.Services) > 0 {
				return model.Services[0].DefaultHost
			}
			return ""
		}(),
		DocLines:                   formatDocComments(model.Description, model.State),
		Imports:                    calculateImports(annotate.imports),
		PartFileReference:          partFileReference,
		PackageDependencies:        packageDependencies,
		DevDependencies:            devDependencies,
		DoNotPublish:               doNotPublish,
		RepositoryURL:              repositoryURL,
		IssueTrackerURL:            issueTrackerURL,
		ReadMeAfterTitleText:       readMeAfterTitleText,
		ReadMeQuickstartText:       readMeQuickstartText,
		ApiKeyEnvironmentVariables: apiKeyEnvironmentVariables,
		Exports:                    exports,
		ProtoPrefix:                protobufPrefix,
	}

	model.Codec = ann
	return nil
}

// calculatePubPackages returns a set of package names (e.g. "http"), given a
// set of imports (e.g. "package:http/http.dart as http").
func calculatePubPackages(imports map[string]bool) map[string]bool {
	packages := map[string]bool{}
	for imp := range imports {
		if name, hadPrefix := strings.CutPrefix(imp, "package:"); hadPrefix {
			name = strings.Split(name, "/")[0]
			packages[name] = true
		}
	}
	return packages
}

// calculateDependencies calculates package dependencies given a set of
// package names (e.g. "http") and version constraints (e.g. {"http": "^1.2.3"}).
//
// Excludes packages that match the current package.
func calculateDependencies(packages map[string]bool, constraints map[string]string, curPkgName string) ([]packageDependency, error) {
	deps := []packageDependency{}

	for name := range packages {
		constraint := constraints[name]
		if name != curPkgName {
			if len(constraint) == 0 {
				return nil, fmt.Errorf("unknown version constraint for package %q (did you forget to add it to .sidekick.toml?)", name)
			}
			deps = append(deps, packageDependency{Name: name, Constraint: constraint})
		}
	}
	sort.SliceStable(deps, func(i, j int) bool {
		return deps[i].Name < deps[j].Name
	})

	return deps, nil
}

func calculateImports(imports map[string]bool) []string {
	var dartImports []string
	for imp := range imports {
		dartImports = append(dartImports, imp)
	}
	sort.Strings(dartImports)

	previousImportType := ""
	var results []string

	for _, imp := range dartImports {
		// Emit a blank line when changing between 'dart:' and 'package:' imports.
		importType := strings.Split(imp, ":")[0]
		if previousImportType != "" && previousImportType != importType {
			results = append(results, "")
		}
		previousImportType = importType

		// Wrap the first part of the import (or the whole import) in single quotes.
		index := strings.IndexAny(imp, " ")
		if index != -1 {
			imp = "'" + imp[0:index] + "'" + imp[index:]
		} else {
			imp = "'" + imp + "'"
		}

		results = append(results, fmt.Sprintf("import %s;", imp))
	}

	return results
}

func (annotate *annotateModel) annotateService(s *api.Service) {
	// Add a package:http import if we're generating a service.
	annotate.imports[httpImport] = true

	// Some methods are skipped.
	methods := language.FilterSlice(s.Methods, func(m *api.Method) bool {
		return shouldGenerateMethod(m)
	})

	for _, m := range methods {
		annotate.annotateMethod(m)
	}
	ann := &serviceAnnotations{
		Name:        s.Name,
		DocLines:    formatDocComments(s.Documentation, annotate.state),
		Methods:     methods,
		FieldName:   strcase.ToLowerCamel(s.Name),
		StructName:  s.Name,
		DefaultHost: s.DefaultHost,
	}
	s.Codec = ann
}

func (annotate *annotateModel) annotateMessage(m *api.Message) {
	// Add the import for the common JSON helpers.

	annotate.imports[encodingImport] = true

	for _, f := range m.Fields {
		annotate.annotateField(f)
	}
	for _, o := range m.OneOfs {
		annotate.annotateOneOf(o)
	}
	for _, e := range m.Enums {
		annotate.annotateEnum(e)
	}
	for _, m := range m.Messages {
		annotate.annotateMessage(m)
	}

	constructorBody := ";"
	_, needsValidation := needsCtorValidation[m.ID]
	if needsValidation {
		constructorBody = " {\n" +
			"    _validate();\n" +
			"  }"
	}

	toStringLines := createToStringLines(m)

	_, omit := omitGeneration[m.ID]

	m.Codec = &messageAnnotation{
		Parent:          m,
		Name:            messageName(m),
		QualifiedName:   qualifiedName(m),
		DocLines:        formatDocComments(m.Documentation, annotate.state),
		OmitGeneration:  omit || m.IsMap,
		ConstructorBody: constructorBody,
		ToStringLines:   toStringLines,
		Model:           annotate.model,
	}
}

func createToStringLines(message *api.Message) []string {
	lines := []string{}

	for _, field := range message.Fields {
		codec := field.Codec.(*fieldAnnotation)
		name := codec.Name

		// Don't generate toString() entries for lists, maps, or messages.
		if field.Repeated || field.Typez == api.MESSAGE_TYPE {
			continue
		}

		var value string
		if strings.Contains(name, "$") {
			value = "${" + name + "}"
		} else {
			value = "$" + name
		}

		if codec.Required {
			// 'name=$name',
			lines = append(lines, fmt.Sprintf("'%s=%s',", field.JSONName, value))
		} else {
			// if (name != null) 'name=$name',
			lines = append(lines,
				fmt.Sprintf("if (%s != null) '%s=%s',", name, field.JSONName, value))
		}
	}

	return lines
}

func (annotate *annotateModel) annotateMethod(method *api.Method) {
	// Ignore imports added from the input and output messages.
	if method.InputType.Codec == nil {
		annotate.annotateMessage(method.InputType)
	}
	if method.OutputType.Codec == nil {
		annotate.annotateMessage(method.OutputType)
	}

	pathInfoAnnotation := &pathInfoAnnotation{
		PathFmt: httpPathFmt(method.PathInfo),
	}
	method.PathInfo.Codec = pathInfoAnnotation

	bodyMessageName := method.PathInfo.BodyFieldPath
	if bodyMessageName == "*" {
		bodyMessageName = "request"
	} else if bodyMessageName != "" {
		bodyMessageName = "request." + strcase.ToLowerCamel(bodyMessageName)
	}

	state := annotate.state

	// For 'GetOperation' mixins, we augment the method generation with
	// additional generic type parameters.
	isGetOperation := method.Name == "GetOperation" &&
		method.OutputTypeID == ".google.longrunning.Operation"
	if method.ID == ".google.longrunning.Operations.GetOperation" {
		isGetOperation = false
	}

	if method.OperationInfo != nil {
		annotate.annotateOperationInfo(method.OperationInfo)
	}

	queryParams := language.QueryParams(method, method.PathInfo.Bindings[0])
	queryLines := []string{}
	for _, field := range queryParams {
		queryLines = annotate.buildQueryLines(queryLines, "request.", "", field, state)
	}

	annotation := &methodAnnotation{
		Parent:              method,
		Name:                strcase.ToLowerCamel(method.Name),
		RequestMethod:       strings.ToLower(method.PathInfo.Bindings[0].Verb),
		RequestType:         annotate.resolveTypeName(state.MessageByID[method.InputTypeID], true),
		ResponseType:        annotate.resolveTypeName(state.MessageByID[method.OutputTypeID], true),
		DocLines:            formatDocComments(method.Documentation, state),
		ReturnsValue:        !method.ReturnsEmpty,
		BodyMessageName:     bodyMessageName,
		QueryLines:          queryLines,
		IsLROGetOperation:   isGetOperation,
		ServerSideStreaming: method.ServerSideStreaming,
	}
	method.Codec = annotation
}

func (annotate *annotateModel) annotateOperationInfo(operationInfo *api.OperationInfo) {
	response := annotate.state.MessageByID[operationInfo.ResponseTypeID]
	metadata := annotate.state.MessageByID[operationInfo.MetadataTypeID]

	operationInfo.Codec = &operationInfoAnnotation{
		ResponseType: annotate.resolveTypeName(response, false),
		MetadataType: annotate.resolveTypeName(metadata, false),
	}
}

func (annotate *annotateModel) annotateOneOf(oneof *api.OneOf) {
	oneof.Codec = &oneOfAnnotation{
		Name:     strcase.ToLowerCamel(oneof.Name),
		DocLines: formatDocComments(oneof.Documentation, annotate.state),
	}
}

func (annotate *annotateModel) annotateField(field *api.Field) {
	// Here, we calculate the nullability / required status of a field. For this
	// we use the proto field presence information.
	//
	// For edification of our readers:
	//   - proto 3 fields default to implicit presence
	//   - the 'optional' keyword changes a field to explicit presence
	//   - types like lists (repeated) and maps are always implicit presence
	//
	// Explicit presence means that you can know whether the user set a value or
	// not. Implicit presence means you can always retrieve a value; if one had
	// not been set, you'll see the default value for that type.
	//
	// We translate explicit presence (a optional annotation) to using a nullable
	// type for that field. We translate implicit presence (always returning some
	// value) to a non-null type.
	//
	// Some short-hand:
	//   - optional == explicit == nullable
	//   - implicit == non-nullable
	//   - lists and maps == implicit == non-nullable
	//   - singular message == explicit == nullable
	//
	// See also https://protobuf.dev/programming-guides/field_presence/.

	var implicitPresence bool

	if field.Repeated || field.Map {
		// Repeated fields and maps have implicit presence (non-nullable).
		implicitPresence = true
	} else if field.Typez == api.MESSAGE_TYPE {
		// In proto3, singular message fields have explicit presence and are nullable.
		implicitPresence = false
	} else {
		if field.IsOneOf {
			// If this field is part of a oneof, it may or may not have a value; we
			// translate that as nullable (explicit presence).
			implicitPresence = false
		} else if field.Optional {
			// The optional keyword makes the field have explicit presence (nullable).
			implicitPresence = false
		} else {
			// Proto3 does not track presence for basic types (implicit presence).
			implicitPresence = true
		}
	}

	// We interpret proto implicit presence as non-nullable for Dart.
	required := implicitPresence

	// We can't make 'bytes' required, as UInt8List in Dart can't be const.
	if field.Typez == api.BYTES_TYPE {
		required = false
	}

	// Calculate the default field value.
	defaultValues := map[api.Typez]string{
		api.BOOL_TYPE:     "false",
		api.DOUBLE_TYPE:   "0",
		api.FIXED32_TYPE:  "0",
		api.FIXED64_TYPE:  "0",
		api.FLOAT_TYPE:    "0",
		api.INT32_TYPE:    "0",
		api.INT64_TYPE:    "0",
		api.SFIXED32_TYPE: "0",
		api.SFIXED64_TYPE: "0",
		api.SINT32_TYPE:   "0",
		api.SINT64_TYPE:   "0",
		api.STRING_TYPE:   "''",
		api.UINT32_TYPE:   "0",
		api.UINT64_TYPE:   "0",
	}
	defaultValue := ""
	if required {
		switch {
		case field.Repeated:
			defaultValue = "const []"
		case field.Map:
			defaultValue = "const {}"
		case field.Typez == api.ENUM_TYPE:
			// The default value for enums are the generated MyEnum.$default field,
			// always set to the first value of that enum.
			typeName := enumName(annotate.state.EnumByID[field.TypezID])
			defaultValue = fmt.Sprintf("%s.$default", typeName)
		default:
			defaultValue = defaultValues[field.Typez]
		}
	}

	state := annotate.state

	field.Codec = &fieldAnnotation{
		Name:                  fieldName(field),
		Type:                  annotate.fieldType(field),
		DocLines:              formatDocComments(field.Documentation, state),
		Required:              required,
		Nullable:              !required,
		FieldBehaviorRequired: slices.Contains(field.Behavior, api.FIELD_BEHAVIOR_REQUIRED),
		DefaultValue:          defaultValue,
		FromJson:              annotate.createFromJsonLine(field, state, required),
		ToJson:                createToJsonLine(field, state, required),
	}
}

func (annotate *annotateModel) createFromJsonLine(field *api.Field, state *api.APIState, required bool) string {
	message := state.MessageByID[field.TypezID]

	data := fmt.Sprintf("json['%s']", field.JSONName)

	bang := ""
	if required {
		switch {
		case field.Repeated:
			bang = " ?? []"
		case field.Map:
			bang = " ?? {}"
		case field.Typez == api.ENUM_TYPE:
			// 'ExecutableCode_Language.$default'
			typeName := enumName(annotate.state.EnumByID[field.TypezID])
			bang = fmt.Sprintf(" ?? %s.$default", typeName)
		default:
			defaultValues := map[api.Typez]string{
				api.BOOL_TYPE:     "false",
				api.BYTES_TYPE:    "Uint8List()",
				api.DOUBLE_TYPE:   "0",
				api.FIXED32_TYPE:  "0",
				api.FIXED64_TYPE:  "0",
				api.FLOAT_TYPE:    "0",
				api.INT32_TYPE:    "0",
				api.INT64_TYPE:    "0",
				api.SFIXED32_TYPE: "0",
				api.SFIXED64_TYPE: "0",
				api.SINT32_TYPE:   "0",
				api.SINT64_TYPE:   "0",
				api.STRING_TYPE:   "''",
				api.UINT32_TYPE:   "0",
				api.UINT64_TYPE:   "0",
			}
			bang = fmt.Sprintf(" ?? %s", defaultValues[field.Typez])
		}
	}

	switch {
	case field.Repeated:
		switch field.Typez {
		case api.BYTES_TYPE:
			return fmt.Sprintf("decodeListBytes(%s)%s", data, bang)
		case api.ENUM_TYPE:
			typeName := enumName(state.EnumByID[field.TypezID])
			return fmt.Sprintf("decodeListEnum(%s, %s.fromJson)%s", data, typeName, bang)
		case api.MESSAGE_TYPE:
			_, hasCustomEncoding := usesCustomEncoding[field.TypezID]
			typeName := annotate.resolveTypeName(state.MessageByID[field.TypezID], true)
			if hasCustomEncoding {
				return fmt.Sprintf("decodeListMessageCustom(%s, %s.fromJson)%s", data, typeName, bang)
			} else {
				return fmt.Sprintf("decodeListMessage(%s, %s.fromJson)%s", data, typeName, bang)
			}
		default:
			return fmt.Sprintf("decodeList(%s)%s", data, bang)
		}
	case field.Map:
		valueField := message.Fields[1]

		switch valueField.Typez {
		case api.BYTES_TYPE:
			return fmt.Sprintf("decodeMapBytes(%s)%s", data, bang)
		case api.ENUM_TYPE:
			typeName := enumName(state.EnumByID[valueField.TypezID])
			return fmt.Sprintf("decodeMapEnum(%s, %s.fromJson)%s", data, typeName, bang)
		case api.MESSAGE_TYPE:
			_, hasCustomEncoding := usesCustomEncoding[valueField.TypezID]
			typeName := annotate.resolveTypeName(state.MessageByID[valueField.TypezID], true)
			if hasCustomEncoding {
				return fmt.Sprintf("decodeMapMessageCustom(%s, %s.fromJson)%s", data, typeName, bang)
			} else {
				return fmt.Sprintf("decodeMapMessage(%s, %s.fromJson)%s", data, typeName, bang)
			}
		default:
			return fmt.Sprintf("decodeMap(%s)%s", data, bang)
		}
	case field.Typez == api.INT64_TYPE ||
		field.Typez == api.UINT64_TYPE || field.Typez == api.SINT64_TYPE ||
		field.Typez == api.FIXED64_TYPE || field.Typez == api.SFIXED64_TYPE:
		return fmt.Sprintf("decodeInt64(%s)%s", data, bang)
	case field.Typez == api.FLOAT_TYPE || field.Typez == api.DOUBLE_TYPE:
		return fmt.Sprintf("decodeDouble(%s)%s", data, bang)
	case field.Typez == api.INT32_TYPE || field.Typez == api.FIXED32_TYPE ||
		field.Typez == api.SFIXED32_TYPE || field.Typez == api.SINT32_TYPE ||
		field.Typez == api.UINT32_TYPE ||
		field.Typez == api.BOOL_TYPE ||
		field.Typez == api.STRING_TYPE:
		return fmt.Sprintf("%s%s", data, bang)
	case field.Typez == api.BYTES_TYPE:
		return fmt.Sprintf("decodeBytes(%s)%s", data, bang)
	case field.Typez == api.ENUM_TYPE:
		typeName := enumName(state.EnumByID[field.TypezID])
		return fmt.Sprintf("decodeEnum(%s, %s.fromJson)%s", data, typeName, bang)
	case field.Typez == api.MESSAGE_TYPE:
		_, hasCustomEncoding := usesCustomEncoding[field.TypezID]
		typeName := annotate.resolveTypeName(state.MessageByID[field.TypezID], true)
		if hasCustomEncoding {
			return fmt.Sprintf("decodeCustom(%s, %s.fromJson)", data, typeName)
		} else {
			return fmt.Sprintf("decode(%s, %s.fromJson)", data, typeName)
		}
	}

	// No decoding necessary.
	return data
}

func createToJsonLine(field *api.Field, state *api.APIState, required bool) string {
	name := fieldName(field)
	message := state.MessageByID[field.TypezID]

	isList := field.Repeated
	isMap := message != nil && message.IsMap

	bang := "!"
	if required {
		bang = ""
	}

	switch {
	case isList:
		switch field.Typez {
		case api.BYTES_TYPE:
			return fmt.Sprintf("encodeListBytes(%s)", name)
		case api.MESSAGE_TYPE, api.ENUM_TYPE:
			return fmt.Sprintf("encodeList(%s)", name)
		default:
			// identity
			return name
		}
	case isMap:
		valueField := message.Fields[1]

		switch valueField.Typez {
		case api.BYTES_TYPE:
			return fmt.Sprintf("encodeMapBytes(%s)", name)
		case api.MESSAGE_TYPE, api.ENUM_TYPE:
			return fmt.Sprintf("encodeMap(%s)", name)
		default:
			// identity
			return name
		}
	case field.Typez == api.MESSAGE_TYPE || field.Typez == api.ENUM_TYPE:
		return fmt.Sprintf("%s%s.toJson()", name, bang)
	case field.Typez == api.BYTES_TYPE:
		return fmt.Sprintf("encodeBytes(%s)", name)
	case field.Typez == api.INT64_TYPE ||
		field.Typez == api.UINT64_TYPE || field.Typez == api.SINT64_TYPE ||
		field.Typez == api.FIXED64_TYPE || field.Typez == api.SFIXED64_TYPE:
		return fmt.Sprintf("encodeInt64(%s)", name)
	case field.Typez == api.FLOAT_TYPE || field.Typez == api.DOUBLE_TYPE:
		return fmt.Sprintf("encodeDouble(%s)", name)
	default:
	}

	// No encoding necessary.
	return name
}

// buildQueryLines builds a string or strings representing query parameters for the given field.
//
// Docs on the format are at
// https://github.com/googleapis/googleapis/blob/master/google/api/http.proto.
//
// Generally:
//   - primitives, lists of primitives and enums are supported
//   - repeated fields are passed as lists
//   - messages need to be unrolled and fields passed individually
func (annotate *annotateModel) buildQueryLines(
	result []string, refPrefix string, paramPrefix string,
	field *api.Field, state *api.APIState,
) []string {
	message := state.MessageByID[field.TypezID]
	isMap := message != nil && message.IsMap

	if field.Codec == nil {
		annotate.annotateField(field)
	}
	codec := field.Codec.(*fieldAnnotation)

	ref := fmt.Sprintf("%s%s", refPrefix, fieldName(field))
	param := fmt.Sprintf("%s%s", paramPrefix, field.JSONName)

	var preable string
	if codec.Nullable {
		preable = fmt.Sprintf("if (%s != null) '%s'", ref, param)
	} else {
		preable = fmt.Sprintf("if (%s.isNotDefault) '%s'", ref, param)
	}

	switch {
	case field.Repeated:
		// Handle lists; these should be lists of strings or other primitives.
		switch field.Typez {
		case api.STRING_TYPE:
			return append(result, fmt.Sprintf("%s: %s", preable, ref))
		case api.ENUM_TYPE:
			return append(result, fmt.Sprintf("%s: %s!.map((e) => e.value)", preable, ref))
		case api.BOOL_TYPE, api.INT32_TYPE, api.UINT32_TYPE, api.SINT32_TYPE,
			api.FIXED32_TYPE, api.SFIXED32_TYPE, api.INT64_TYPE,
			api.UINT64_TYPE, api.SINT64_TYPE, api.FIXED64_TYPE, api.SFIXED64_TYPE,
			api.FLOAT_TYPE, api.DOUBLE_TYPE:
			return append(result, fmt.Sprintf("%s: %s!.map((e) => '$e')", preable, ref))
		case api.BYTES_TYPE:
			return append(result, fmt.Sprintf("%s: %s!.map((e) => encodeBytes(e)!)", preable, ref))
		default:
			slog.Error("unhandled list query param", "type", field.Typez)
			return append(result, fmt.Sprintf("/* unhandled list query param type: %d */", field.Typez))
		}

	case isMap:
		// Maps are not supported.
		slog.Error("unhandled query param", "type", "map")
		return append(result, fmt.Sprintf("/* unhandled query param type: %d */", field.Typez))

	case field.Typez == api.MESSAGE_TYPE:
		deref := "."
		if codec.Nullable {
			deref = "!."
		}

		// Unroll the fields for messages.
		for _, field := range message.Fields {
			result = annotate.buildQueryLines(result, ref+deref, param+".", field, state)
		}
		return result

	case field.Typez == api.STRING_TYPE:
		deref := ""
		if codec.Nullable {
			deref = "!"
		}
		return append(result, fmt.Sprintf("%s: %s%s", preable, ref, deref))
	case field.Typez == api.ENUM_TYPE:
		return append(result, fmt.Sprintf("%s: %s.value", preable, ref))
	case field.Typez == api.BOOL_TYPE ||
		field.Typez == api.INT32_TYPE ||
		field.Typez == api.UINT32_TYPE || field.Typez == api.SINT32_TYPE ||
		field.Typez == api.FIXED32_TYPE || field.Typez == api.SFIXED32_TYPE ||
		field.Typez == api.INT64_TYPE ||
		field.Typez == api.UINT64_TYPE || field.Typez == api.SINT64_TYPE ||
		field.Typez == api.FIXED64_TYPE || field.Typez == api.SFIXED64_TYPE ||
		field.Typez == api.FLOAT_TYPE || field.Typez == api.DOUBLE_TYPE:
		return append(result, fmt.Sprintf("%s: '${%s}'", preable, ref))
	case field.Typez == api.BYTES_TYPE:
		return append(result, fmt.Sprintf("%s: encodeBytes(%s)!", preable, ref))
	default:
		slog.Error("unhandled query param", "type", field.Typez)
		return append(result, fmt.Sprintf("/* unhandled query param type: %d */", field.Typez))
	}
}

func (annotate *annotateModel) annotateEnum(enum *api.Enum) {
	for _, ev := range enum.Values {
		annotate.annotateEnumValue(ev)
	}

	defaultValue := ""
	if len(enum.Values) > 0 {
		defaultValue = enumValueName(enum.Values[0])
	}

	enum.Codec = &enumAnnotation{
		Name:         enumName(enum),
		DocLines:     formatDocComments(enum.Documentation, annotate.state),
		DefaultValue: defaultValue,
		Model:        annotate.model,
	}
}

func (annotate *annotateModel) annotateEnumValue(ev *api.EnumValue) {
	ev.Codec = &enumValueAnnotation{
		Name:     enumValueName(ev),
		DocLines: formatDocComments(ev.Documentation, annotate.state),
	}
}

func (annotate *annotateModel) fieldType(f *api.Field) string {
	var out string

	switch f.Typez {
	case api.BOOL_TYPE:
		out = "bool"
	case api.INT32_TYPE, api.UINT32_TYPE, api.SINT32_TYPE,
		api.FIXED32_TYPE, api.SFIXED32_TYPE:
		out = "int"
	case api.INT64_TYPE, api.UINT64_TYPE, api.SINT64_TYPE,
		api.FIXED64_TYPE, api.SFIXED64_TYPE:
		out = "int"
	case api.FLOAT_TYPE, api.DOUBLE_TYPE:
		out = "double"
	case api.STRING_TYPE:
		out = "String"
	case api.BYTES_TYPE:
		out = "Uint8List"
	case api.MESSAGE_TYPE:
		message, ok := annotate.state.MessageByID[f.TypezID]
		if !ok {
			slog.Error("unable to lookup type", "id", f.TypezID)
			return ""
		}
		if message.IsMap {
			key := annotate.fieldType(message.Fields[0])
			val := annotate.fieldType(message.Fields[1])
			out = "Map<" + key + ", " + val + ">"
		} else {
			out = annotate.resolveTypeName(message, true)
		}
	case api.ENUM_TYPE:
		e, ok := annotate.state.EnumByID[f.TypezID]
		if !ok {
			slog.Error("unable to lookup type", "id", f.TypezID)
			return ""
		}
		annotate.updateUsedPackages(e.Package)
		out = enumName(e)
		importPrefix, needsImportPrefix := annotate.packagePrefixes[e.Package]
		if needsImportPrefix {
			out = importPrefix + "." + out
		}
	default:
		slog.Error("unhandled fieldType", "type", f.Typez, "id", f.TypezID)
	}

	if f.Repeated {
		out = "List<" + out + ">"
	}

	return out
}

func (annotate *annotateModel) resolveTypeName(message *api.Message, returnVoidForEmpty bool) string {
	if message == nil {
		slog.Error("unable to lookup type")
		return ""
	}

	if message.ID == ".google.protobuf.Empty" && returnVoidForEmpty {
		return "void"
	}

	annotate.updateUsedPackages(message.Package)

	ref := messageName(message)
	importPrefix, needsImportPrefix := annotate.packagePrefixes[message.Package]
	if needsImportPrefix {
		ref = importPrefix + "." + ref
	}
	return ref
}

func (annotate *annotateModel) updateUsedPackages(packageName string) {
	selfReference := annotate.model.PackageName == packageName
	if !selfReference {
		// Use the packageMapping info to add any necessary import.
		dartImport, ok := annotate.packageMapping[packageName]
		if ok {
			importPrefix, needsImportPrefix := annotate.packagePrefixes[packageName]
			if needsImportPrefix {
				dartImport += " as " + importPrefix
			}
			annotate.imports[dartImport] = true
		}
	}
}

func registerMissingWkt(state *api.APIState) {
	// If these definitions weren't provided by protoc then provide our own
	// placeholders.
	for _, message := range []struct {
		ID      string
		Name    string
		Package string
	}{
		{".google.protobuf.Any", "Any", "google.protobuf"},
		{".google.protobuf.Empty", "Empty", "google.protobuf"},
	} {
		_, ok := state.MessageByID[message.ID]
		if !ok {
			state.MessageByID[message.ID] = &api.Message{
				ID:      message.ID,
				Name:    message.Name,
				Package: message.Package,
			}
		}
	}
}
