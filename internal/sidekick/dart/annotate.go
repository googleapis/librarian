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

	"github.com/googleapis/librarian/internal/sidekick/api"
	"github.com/googleapis/librarian/internal/sidekick/language"
	"github.com/googleapis/librarian/internal/sidekick/license"
	"github.com/iancoleman/strcase"
)

var omitGeneration = map[string]string{
	".google.longrunning.Operation": "",
	".google.protobuf.Value":        "",
}

var defaultValues = map[api.Typez]struct {
	Value   string
	IsConst bool
}{
	api.BOOL_TYPE:     {"false", true},
	api.BYTES_TYPE:    {"Uint8List(0)", false},
	api.DOUBLE_TYPE:   {"0", true},
	api.FIXED32_TYPE:  {"0", true},
	api.FIXED64_TYPE:  {"BigInt.zero", false},
	api.FLOAT_TYPE:    {"0", true},
	api.INT32_TYPE:    {"0", true},
	api.INT64_TYPE:    {"0", true},
	api.SFIXED32_TYPE: {"0", true},
	api.SFIXED64_TYPE: {"0", true},
	api.SINT32_TYPE:   {"0", true},
	api.SINT64_TYPE:   {"0", true},
	api.STRING_TYPE:   {"''", true},
	api.UINT32_TYPE:   {"0", true},
	api.UINT64_TYPE:   {"BigInt.zero", false},
}

type modelAnnotations struct {
	Parent *api.API
	// The Dart package name (e.g. google_cloud_secretmanager).
	PackageName string
	// The version of the generated package.
	PackageVersion string
	// Name of the API in snake_format (e.g. secretmanager).
	MainFileNameWithExtension string
	SourcePackageName         string
	CopyrightYear             string
	BoilerPlate               []string
	DefaultHost               string
	DocLines                  []string
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
	// The default value for the string, e.g. "0" for an integer type.
	DefaultValue string
	// Whether the default value is constant or not, e.g. "0" is constant but "Uint8List(0)" is not.
	ConstDefault bool
	FromJson     string
	ToJson       string
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
		libraryPathOverride        string
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
		case key == "library-path-override":
			// library-path-override = "src/buffers.dart"
			// The path to use for the generated file, relative to the package's "lib/" directory.
			libraryPathOverride = definition
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

	// Add the import for ServiceClient and related functionality.
	if len(model.Services) > 0 {
		annotate.imports[serviceClientImport] = true
		annotate.imports[serviceExceptionImport] = true
	}

	// `protobuf.dart` defines `JsonEncodable`, which is needed by any API that defines an `enum` or `message`.
	annotate.imports[protobufImport] = true
	// `encoding.dart` defines primitive JSON encoding/decode methods, which are needed by any API that defines
	// an `enum` or `message`.
	annotate.imports[encodingImport] = true

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

	mainFileNameWithExtension := strcase.ToSnake(model.Name) + ".dart"
	if libraryPathOverride != "" {
		mainFileNameWithExtension = libraryPathOverride
	}

	slices.Sort(devDependencies)

	ann := &modelAnnotations{
		Parent:                    model,
		PackageName:               pkgName,
		PackageVersion:            packageVersion,
		MainFileNameWithExtension: mainFileNameWithExtension,
		CopyrightYear:             generationYear,
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
		Imports:                    calculateImports(annotate.imports, pkgName, mainFileNameWithExtension),
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

// calculateImports generates Dart import statements given a set of imports.
//
// For example:
// `{"dart:io": true, "package:http/http.dart as http": true}` to
// `{"import 'dart:io';", "", "import 'package:http/http.dart' as http;"}`.
func calculateImports(imports map[string]bool, curPkgName string, curFileName string) []string {
	var dartImports []string
	var packageImports []string
	var localImports []string

	sortedImports := make([]string, 0, len(imports))
	for imp := range imports {
		sortedImports = append(sortedImports, imp)
	}
	sort.Strings(sortedImports)

	for _, imp := range sortedImports {
		parts := strings.SplitN(imp, ":", 2)
		if len(parts) != 2 {
			continue
		}
		scheme := parts[0]
		body := parts[1]

		if scheme == "dart" {
			dartImports = append(dartImports, formatImport(imp))
			continue
		} else if scheme == "package" {
			if strings.HasPrefix(body, curPkgName+"/") {
				pathAndAlias := strings.TrimPrefix(body, curPkgName+"/")

				pathOnly := strings.Split(pathAndAlias, " ")[0]
				if pathOnly == curFileName {
					continue
				}

				localImports = append(localImports, formatImport(pathAndAlias))
			} else {
				packageImports = append(packageImports, formatImport(imp))
			}
		} else {
			panic("unknown import scheme: " + imp)
		}
	}

	var result []string
	if len(dartImports) > 0 {
		result = append(result, dartImports...)
	}

	if len(packageImports) > 0 {
		if len(result) > 0 {
			result = append(result, "")
		}
		result = append(result, packageImports...)
	}

	if len(localImports) > 0 {
		if len(result) > 0 {
			result = append(result, "")
		}
		result = append(result, localImports...)
	}

	return result
}

func formatImport(imp string) string {
	index := strings.IndexAny(imp, " ")
	if index != -1 {
		return fmt.Sprintf("import '%s'%s;", imp[0:index], imp[index:])
	}
	return fmt.Sprintf("import '%s';", imp)
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
	_, omit := omitGeneration[m.ID]

	if omit {
		m.Codec = &messageAnnotation{
			OmitGeneration: true,
		}
		return
	}

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
		RequestType:         annotate.resolveMessageName(state.MessageByID[method.InputTypeID], true),
		ResponseType:        annotate.resolveMessageName(state.MessageByID[method.OutputTypeID], true),
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
		ResponseType: annotate.resolveMessageName(response, false),
		MetadataType: annotate.resolveMessageName(metadata, false),
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

	// Calculate the default field value.
	defaultValue := ""
	constDefault := true
	fieldRequired := slices.Contains(field.Behavior, api.FIELD_BEHAVIOR_REQUIRED)
	if implicitPresence && !fieldRequired {
		switch {
		case field.Repeated:
			defaultValue = "const []"
		case field.Map:
			defaultValue = "const {}"
		case field.Typez == api.ENUM_TYPE:
			// The default value for enums are the generated MyEnum.$default field,
			// always set to the first value of that enum.
			typeName := annotate.resolveEnumName(annotate.state.EnumByID[field.TypezID])
			defaultValue = fmt.Sprintf("%s.$default", typeName)
		default:
			defaultValue = defaultValues[field.Typez].Value
			constDefault = defaultValues[field.Typez].IsConst
		}
	}
	state := annotate.state
	field.Codec = &fieldAnnotation{
		Name:                  fieldName(field),
		Type:                  annotate.fieldType(field),
		DocLines:              formatDocComments(field.Documentation, state),
		Required:              implicitPresence,
		Nullable:              !implicitPresence,
		FieldBehaviorRequired: fieldRequired,
		DefaultValue:          defaultValue,
		FromJson:              annotate.createFromJsonLine(field, state, implicitPresence),
		ToJson:                createToJsonLine(field, state),
		ConstDefault:          constDefault,
	}
}

func (annotate *annotateModel) decoder(typez api.Typez, typeid string, state *api.APIState) string {
	switch typez {
	case api.INT64_TYPE,
		api.SINT64_TYPE,
		api.SFIXED64_TYPE:
		return "decodeInt64"
	case api.FIXED64_TYPE,
		api.UINT64_TYPE:
		return "decodeUint64"
	case api.FLOAT_TYPE,
		api.DOUBLE_TYPE:
		return "decodeDouble"
	case api.INT32_TYPE,
		api.FIXED32_TYPE,
		api.SFIXED32_TYPE,
		api.SINT32_TYPE,
		api.UINT32_TYPE:
		return "decodeInt"
	case api.BOOL_TYPE:
		return "decodeBool"
	case api.STRING_TYPE:
		return "decodeString"
	case api.BYTES_TYPE:
		return "decodeBytes"
	case api.ENUM_TYPE:
		typeName := annotate.resolveEnumName(state.EnumByID[typeid])
		return fmt.Sprintf("%s.fromJson", typeName)
	case api.MESSAGE_TYPE:
		typeName := annotate.resolveMessageName(state.MessageByID[typeid], true)
		return fmt.Sprintf("%s.fromJson", typeName)
	default:
		panic(fmt.Sprintf("unsupported type: %d", typez))
	}
}

func (annotate *annotateModel) keyDecoder(typez api.Typez) string {
	// JSON objects can only contain string keys so non-String types need to use specialized decoders.
	// Supported key types are defined here:
	// https://protobuf.dev/programming-guides/proto3/#maps
	switch typez {
	case api.STRING_TYPE:
		return "decodeString"
	case api.INT32_TYPE, // Integer types that can be decoded as Dart `int`.
		api.FIXED32_TYPE,
		api.SFIXED32_TYPE,
		api.SINT32_TYPE,
		api.UINT32_TYPE,
		api.INT64_TYPE,
		api.SINT64_TYPE,
		api.SFIXED64_TYPE:
		return "decodeIntKey"
	case api.UINT64_TYPE,
		api.FIXED64_TYPE:
		return "decodeUint64Key"
	case api.BOOL_TYPE:
		return "decodeBoolKey"
	default:
		panic(fmt.Sprintf("unsupported key type: %d", typez))
	}
}

// encoder returns a string that encodes the given field name and a bool that indicates whether the
// field requires encoding.
//
// For example:
//
//	encoder(api.STRING_TYPE, "a_string_field") => ("a_string_field", false)
//	encoder(api.MESSAGE_TYPE, "a_message_field") => ("a_message_field.toJson()", true)
func encoder(typez api.Typez, name string) (string, bool) {
	switch typez {
	case api.INT64_TYPE,
		api.SINT64_TYPE,
		api.SFIXED64_TYPE,
		api.FIXED64_TYPE,
		api.UINT64_TYPE:
		// All 64-bit integer types are encoded as strings. In Dart, these may be
		// represented as `int` or `BigInt`.
		return fmt.Sprintf("%s.toString()", name), true
	case api.FLOAT_TYPE,
		api.DOUBLE_TYPE:
		// A special encoder is needed to handle NaN and Infinity.
		return fmt.Sprintf("encodeDouble(%s)", name), true
	case api.INT32_TYPE,
		api.FIXED32_TYPE,
		api.SFIXED32_TYPE,
		api.SINT32_TYPE,
		api.UINT32_TYPE:
		// All 32-bit integer types are encoded as JSON numbers.
		return name, false
	case api.BOOL_TYPE:
		return name, false
	case api.STRING_TYPE:
		return name, false
	case api.BYTES_TYPE:
		return fmt.Sprintf("encodeBytes(%s)", name), true
	case api.MESSAGE_TYPE, api.ENUM_TYPE:
		return fmt.Sprintf("%s.toJson()", name), true
	default:
		panic(fmt.Sprintf("unsupported type: %d", typez))
	}
}

// keyEncoder returns a string that encodes the given field name and a bool that indicates whether the
// field requires encoding
//
// For example:
//
//	keyEncoder(api.STRING_TYPE, "e.key") => ("e.key", false)
//	keyEncoder(api.INT32_TYPE, "e.key") => ("e.key.toString()", true)
func keyEncoder(typez api.Typez, name string) (string, bool) {
	// JSON objects can only contain string keys so non-String types need to be encoded as strings.
	// Supported key types are defined here:
	// https://protobuf.dev/programming-guides/proto3/#maps
	switch typez {
	case api.STRING_TYPE:
		return name, false
	case api.INT32_TYPE,
		api.FIXED32_TYPE,
		api.SFIXED32_TYPE,
		api.SINT32_TYPE,
		api.UINT32_TYPE,
		api.INT64_TYPE,
		api.SINT64_TYPE,
		api.SFIXED64_TYPE,
		api.UINT64_TYPE,
		api.FIXED64_TYPE:
		return fmt.Sprintf("%s.toString()", name), true
	case api.BOOL_TYPE:
		return fmt.Sprintf("%s.toString()", name), true
	default:
		panic(fmt.Sprintf("unsupported key type: %d", typez))
	}
}

func (annotate *annotateModel) createFromJsonLine(field *api.Field, state *api.APIState, required bool) string {
	data := fmt.Sprintf("json['%s']", field.JSONName)

	defaultValue := "null"
	if required {
		switch {
		case field.Repeated:
			defaultValue = "[]"
		case field.Map:
			defaultValue = "{}"
		case field.Typez == api.ENUM_TYPE:
			// 'ExecutableCode_Language.$default'
			typeName := annotate.resolveEnumName(annotate.state.EnumByID[field.TypezID])
			defaultValue = fmt.Sprintf("%s.$default", typeName)
		default:
			defaultValue = defaultValues[field.Typez].Value
		}
	}

	switch {
	// Value.NullValue is encoded as null in JSON so lists and map values must match on nullable objects.
	case field.Repeated:
		decoder := annotate.decoder(field.Typez, field.TypezID, state)
		return fmt.Sprintf(
			"switch (%s) { null => %s, List<Object?> $1 => [for (final i in $1) %s(i)], "+
				"_ => throw const FormatException('\"%s\" is not a list') }",
			data, defaultValue, decoder, field.JSONName)
	case field.Map:
		message := state.MessageByID[field.TypezID]
		keyType := message.Fields[0].Typez
		keyDecoder := annotate.keyDecoder(keyType)
		valueType := message.Fields[1].Typez
		valueTypeID := message.Fields[1].TypezID
		valueDecoder := annotate.decoder(valueType, valueTypeID, state)

		return fmt.Sprintf(
			"switch (%s) { null => %s, Map<String, Object?> $1 => {for (final e in $1.entries) %s(e.key): %s(e.value)}, "+
				"_ => throw const FormatException('\"%s\" is not an object') }",
			data, defaultValue, keyDecoder, valueDecoder, field.JSONName)
	}

	decoder := annotate.decoder(field.Typez, field.TypezID, state)
	return fmt.Sprintf("switch (%s) { null => %s, Object $1 => %s($1)}", data, defaultValue, decoder)
}

func createToJsonLine(field *api.Field, state *api.APIState) string {
	name := fieldName(field)

	switch {
	case field.Repeated:
		if encoder, encodingRequired := encoder(field.Typez, "i"); encodingRequired {
			return fmt.Sprintf(
				"[for (final i in %s) %s]",
				name, encoder)
		}
		return name
	case field.Map:
		message := state.MessageByID[field.TypezID]
		keyType := message.Fields[0].Typez
		keyEncoder, keyEncodingRequired := keyEncoder(keyType, "e.key")
		valueType := message.Fields[1].Typez
		valueEncoder, valueEncodingRequired := encoder(valueType, "e.value")

		if keyEncodingRequired || valueEncodingRequired {
			return fmt.Sprintf(
				"{for (final e in %s.entries) %s: %s}",
				name, keyEncoder, valueEncoder)
		}
		return name
	}

	enc, _ := encoder(field.Typez, name)
	return enc
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
		preable = fmt.Sprintf("if (%s case final $1?) '%s'", ref, param)
	} else {
		preable = fmt.Sprintf("if (%s case final $1 when $1.isNotDefault) '%s'", ref, param)
	}

	switch {
	case field.Repeated:
		// Handle lists; these should be lists of strings or other primitives.
		switch field.Typez {
		case api.STRING_TYPE:
			return append(result, fmt.Sprintf("%s: $1", preable))
		case api.ENUM_TYPE:
			return append(result, fmt.Sprintf("%s: $1.map((e) => e.value)", preable))
		case api.BOOL_TYPE, api.INT32_TYPE, api.UINT32_TYPE, api.SINT32_TYPE,
			api.FIXED32_TYPE, api.SFIXED32_TYPE, api.INT64_TYPE,
			api.UINT64_TYPE, api.SINT64_TYPE, api.FIXED64_TYPE, api.SFIXED64_TYPE,
			api.FLOAT_TYPE, api.DOUBLE_TYPE:
			return append(result, fmt.Sprintf("%s: $1.map((e) => '$e')", preable))
		case api.BYTES_TYPE:
			return append(result, fmt.Sprintf("%s: $1.map((e) => encodeBytes(e)!)", preable))
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

		_, hasCustomEncoding := usesCustomEncoding[field.TypezID]
		if hasCustomEncoding {
			// Example: 'fieldMask': fieldMask!.toJson()
			return append(result, fmt.Sprintf("%s: $1.toJson()", preable))
		}

		// Unroll the fields for messages.
		for _, field := range message.Fields {
			result = annotate.buildQueryLines(result, ref+deref, param+".", field, state)
		}
		return result

	case field.Typez == api.STRING_TYPE:
		return append(result, fmt.Sprintf("%s: $1", preable))
	case field.Typez == api.ENUM_TYPE:
		return append(result, fmt.Sprintf("%s: $1.value", preable))
	case field.Typez == api.BOOL_TYPE ||
		field.Typez == api.INT32_TYPE ||
		field.Typez == api.UINT32_TYPE || field.Typez == api.SINT32_TYPE ||
		field.Typez == api.FIXED32_TYPE || field.Typez == api.SFIXED32_TYPE ||
		field.Typez == api.INT64_TYPE ||
		field.Typez == api.UINT64_TYPE || field.Typez == api.SINT64_TYPE ||
		field.Typez == api.FIXED64_TYPE || field.Typez == api.SFIXED64_TYPE ||
		field.Typez == api.FLOAT_TYPE || field.Typez == api.DOUBLE_TYPE:
		return append(result, fmt.Sprintf("%s: '${$1}'", preable))
	case field.Typez == api.BYTES_TYPE:
		return append(result, fmt.Sprintf("%s: encodeBytes($1)!", preable))
	default:
		slog.Error("unhandled query param", "type", field.Typez)
		return append(result, fmt.Sprintf("/* unhandled query param type: %d */", field.Typez))
	}
}

func (annotate *annotateModel) annotateEnum(enum *api.Enum) {
	names := make(map[string]bool)
	enumValueToName := make(map[*api.EnumValue]string)
	useOriginalCase := false

	// Attempt to use Dart-style camelCase for enum values.
	// If there is a conflict, it must mean that there are enum values that differ only by case.
	// If there are values that differ only in case, use the original case for all enum values.
	for _, ev := range enum.Values {
		name := strcase.ToLowerCamel(ev.Name)
		if _, hasConflict := reservedNames[name]; hasConflict {
			name = name + deconflictChar
		}
		enumValueToName[ev] = name
		if _, ok := names[name]; ok {
			useOriginalCase = true
			break
		}
		names[name] = true
	}

	if useOriginalCase {
		for _, ev := range enum.Values {
			name := ev.Name
			if _, hasConflict := reservedNames[name]; hasConflict {
				name = name + deconflictChar
			}
			enumValueToName[ev] = name
		}
	}

	defaultValue := ""
	if len(enum.Values) > 0 {
		defaultValue = enumValueToName[enum.Values[0]]
	}

	for _, ev := range enum.Values {
		annotate.annotateEnumValue(ev, enumValueToName)
	}

	enum.Codec = &enumAnnotation{
		Name:         enumName(enum),
		DocLines:     formatDocComments(enum.Documentation, annotate.state),
		DefaultValue: defaultValue,
		Model:        annotate.model,
	}
}

func (annotate *annotateModel) annotateEnumValue(ev *api.EnumValue, enumValueToName map[*api.EnumValue]string) {
	ev.Codec = &enumValueAnnotation{
		Name:     enumValueToName[ev],
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
	case api.INT64_TYPE, api.SINT64_TYPE, api.SFIXED64_TYPE:
		out = "int"
	case api.FIXED64_TYPE, api.UINT64_TYPE:
		out = "BigInt"
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
			out = annotate.resolveMessageName(message, true)
		}
	case api.ENUM_TYPE:
		e, ok := annotate.state.EnumByID[f.TypezID]
		if !ok {
			slog.Error("unable to lookup type", "id", f.TypezID)
			return ""
		}
		out = annotate.resolveEnumName(e)
	default:
		slog.Error("unhandled fieldType", "type", f.Typez, "id", f.TypezID)
	}

	if f.Repeated {
		out = "List<" + out + ">"
	}

	return out
}

func (annotate *annotateModel) resolveEnumName(enum *api.Enum) string {
	annotate.updateUsedPackages(enum.Package)

	ref := enumName(enum)
	importPrefix, needsImportPrefix := annotate.packagePrefixes[enum.Package]
	if needsImportPrefix {
		ref = importPrefix + "." + ref
	}
	return ref
}

func (annotate *annotateModel) resolveMessageName(message *api.Message, returnVoidForEmpty bool) string {
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
	if strings.Contains(packageName, "rpc") {
		fmt.Println("debug") // Set a regular breakpoint here
	}
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
