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

// Package parser reads specifications and converts them into
// the `genclient.API` model.
package parser

import (
	"errors"
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/googleapis/librarian/internal/serviceconfig"
	"github.com/googleapis/librarian/internal/sidekick/api"
	"github.com/googleapis/librarian/internal/sidekick/parser/httprule"
	"github.com/googleapis/librarian/internal/sidekick/parser/svcconfig"
	"github.com/pb33f/libopenapi"
	"github.com/pb33f/libopenapi/datamodel/high/base"
	v3 "github.com/pb33f/libopenapi/datamodel/high/v3"
	"github.com/pb33f/libopenapi/orderedmap"
)

// ParseOpenAPI parses an OpenAPI specification and returns an API model.
func ParseOpenAPI(cfg *ModelConfig) (*api.API, error) {
	source := cfg.SpecificationSource
	contents, err := os.ReadFile(source)
	if err != nil {
		return nil, err
	}
	model, err := createDocModel(contents)
	if err != nil {
		return nil, err
	}
	serviceConfig, err := loadServiceConfig(cfg)
	if err != nil {
		return nil, err
	}
	return makeAPIForOpenAPI(serviceConfig, model)
}

func createDocModel(contents []byte) (*libopenapi.DocumentModel[v3.Document], error) {
	document, err := libopenapi.NewDocument(contents)
	if err != nil {
		return nil, err
	}
	docModel, errs := document.BuildV3Model()
	if len(errs) > 0 {
		return nil, fmt.Errorf("cannot convert document to OpenAPI V3 model: %w", errors.Join(errs...))
	}
	return docModel, nil
}

func makeAPIForOpenAPI(serviceConfig *serviceconfig.Service, model *libopenapi.DocumentModel[v3.Document]) (*api.API, error) {
	result := &api.API{
		Name:        "",
		Title:       model.Model.Info.Title,
		Description: model.Model.Info.Description,
		Messages:    make([]*api.Message, 0),
	}
	// OpenAPI (for Google) uses some well-known types inspired by Protobuf.
	// With protoc these types are automatically included via `import`
	// statements. In the OpenAPI JSON inputs, these types are not automatically
	// included.
	result.LoadWellKnownTypes()

	if serviceConfig != nil {
		result.Name = strings.TrimSuffix(serviceConfig.Name, ".googleapis.com")
		result.Title = serviceConfig.Title
		if serviceConfig.Documentation != nil {
			result.Description = serviceConfig.Documentation.Summary
		}
	}

	// OpenAPI does not define a service name. The service config may provide
	// one. In tests, the service config is typically `nil`.
	serviceName := "Service"
	packageName := ""
	names := svcconfig.ExtractPackageName(serviceConfig)
	if names != nil {
		serviceName, packageName = names.ServiceName, names.PackageName
		result.PackageName = packageName
	}

	for name, msg := range model.Model.Components.Schemas.FromOldest() {
		id := fmt.Sprintf(".%s.%s", packageName, name)
		schema, err := msg.BuildSchema()
		if err != nil {
			return nil, err
		}
		fields, err := makeMessageFields(result, packageName, name, schema)
		if err != nil {
			return nil, err
		}
		message := &api.Message{
			Name:          name,
			ID:            id,
			Package:       packageName,
			Deprecated:    msg.Schema().Deprecated != nil && *msg.Schema().Deprecated,
			Documentation: msg.Schema().Description,
			Fields:        fields,
		}

		result.Messages = append(result.Messages, message)
		result.AddMessage(message)
	}

	err := makeServices(result, model, serviceConfig, packageName, serviceName)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func makeServices(a *api.API, model *libopenapi.DocumentModel[v3.Document], serviceConfig *serviceconfig.Service, packageName, serviceName string) error {
	// It is hard to imagine an OpenAPI specification without at least some
	// RPCs, but we can simplify the tests if we support specifications without
	// paths or without any useful methods in the paths.
	if model.Model.Paths == nil {
		return nil
	}
	sID := fmt.Sprintf(".%s.%s", packageName, serviceName)
	methods, err := makeMethods(a, model, serviceConfig, packageName, serviceName, sID)
	if err != nil {
		return err
	}
	service := &api.Service{
		Name:          serviceName,
		ID:            sID,
		Package:       packageName,
		Documentation: a.Description,
		DefaultHost:   defaultHost(model),
		Methods:       methods,
	}
	a.Services = append(a.Services, service)
	a.AddService(service)
	return nil
}

func defaultHost(model *libopenapi.DocumentModel[v3.Document]) string {
	defaultHost := ""
	for _, server := range model.Model.Servers {
		if defaultHost == "" {
			defaultHost = server.URL
		} else if len(defaultHost) > len(server.URL) {
			defaultHost = server.URL
		}
	}
	// The Go template adds https:// because Protobuf does not include
	// the scheme.
	return strings.TrimPrefix(defaultHost, "https://")
}

func makeMethods(a *api.API, model *libopenapi.DocumentModel[v3.Document], serviceConfig *serviceconfig.Service, packageName, serviceName, serviceID string) ([]*api.Method, error) {
	if model.Model.Paths == nil {
		// The method has no path, there is nothing to generate.
		return nil, nil
	}

	// It is Okay to reuse the ID, sidekick uses different the namespaces
	// for messages vs. services.
	parent := &api.Message{
		Name:               serviceName,
		ID:                 serviceID,
		Package:            packageName,
		Documentation:      fmt.Sprintf("Synthetic messages for the [%s][%s] service.", serviceName, serviceID[1:]),
		ServicePlaceholder: true,
	}
	a.AddMessage(parent)
	a.Messages = append(a.Messages, parent)

	var methods []*api.Method
	for pattern, item := range model.Model.Paths.PathItems.FromOldest() {
		pathTemplate, err := httprule.ParseSegments(pattern)
		if err != nil {
			return nil, err
		}

		type NamedOperation struct {
			Verb      string
			Operation *v3.Operation
		}
		operations := []NamedOperation{
			{Verb: "GET", Operation: item.Get},
			{Verb: "PUT", Operation: item.Put},
			{Verb: "POST", Operation: item.Post},
			{Verb: "DELETE", Operation: item.Delete},
			{Verb: "OPTIONS", Operation: item.Options},
			{Verb: "HEAD", Operation: item.Head},
			{Verb: "PATCH", Operation: item.Patch},
			{Verb: "TRACE", Operation: item.Trace},
		}
		for _, op := range operations {
			if op.Operation == nil {
				continue
			}
			requestMessage, bodyFieldPath, err := makeRequestMessage(a, parent, op.Operation, packageName, pattern)
			if err != nil {
				return nil, err
			}
			responseMessage, err := makeResponseMessage(a, op.Operation, packageName)
			if err != nil {
				return nil, err
			}
			queryParameters := makeQueryParameters(op.Operation)
			pathInfo := api.NewPathInfoBuilder().
				WithBindings(
					api.NewPathBindingBuilder().
						WithVerb(op.Verb).
						WithPathTemplate(pathTemplate).
						WithQueryParameterMap(queryParameters).
						Build(),
				).
				WithBodyFieldPath(bodyFieldPath).
				Build()
			mID := fmt.Sprintf("%s.%s", serviceID, op.Operation.OperationId)
			m := &api.Method{
				Name:          op.Operation.OperationId,
				ID:            mID,
				Deprecated:    op.Operation.Deprecated != nil && *op.Operation.Deprecated,
				Documentation: op.Operation.Description,
				InputTypeID:   requestMessage.ID,
				OutputTypeID:  responseMessage.ID,
				PathInfo:      pathInfo,
				AutoPopulated: findAutoPopulatedFields(serviceConfig, mID, requestMessage),
			}
			a.AddMethod(m)
			methods = append(methods, m)
		}
	}
	return methods, nil
}

// makeRequestMessage creates (if needed) the request message for `operation`. Returns the message
// and the body field path (if any) for the request.
func makeRequestMessage(a *api.API, parent *api.Message, operation *v3.Operation, packageName, template string) (*api.Message, string, error) {
	messageName := fmt.Sprintf("%sRequest", operation.OperationId)
	id := fmt.Sprintf("%s.%s", parent.ID, messageName)
	methodID := fmt.Sprintf("%s.%s", parent.ID, operation.OperationId)
	message := &api.Message{
		Name:             messageName,
		ID:               id,
		Package:          packageName,
		Documentation:    fmt.Sprintf("Synthetic request message for the [%s()][%s] method.", operation.OperationId, methodID[1:]),
		SyntheticRequest: true,
		Parent:           parent,
	}
	fieldNames := map[string]bool{}
	for _, p := range operation.Parameters {
		schema, err := p.Schema.BuildSchema()
		if err != nil {
			return nil, "", fmt.Errorf("error building schema for parameter %s: %w", p.Name, err)
		}
		typez, typezID, err := scalarType(messageName, p.Name, schema)
		if err != nil {
			return nil, "", err
		}
		documentation := p.Description
		if len(documentation) == 0 {
			// In Google's OpenAPI v3 specifications the parameters often lack
			// any documentation. Create a synthetic document in this case.
			documentation = fmt.Sprintf(
				"The `{%s}` component of the target path.\n"+
					"\n"+
					"The full target path will be in the form `%s`.", p.Name, template)
		}
		field := api.NewFieldBuilder().
			WithName(p.Name).
			WithID(fmt.Sprintf("%s.%s", message.ID, p.Name)).
			WithJSONName(p.Name). // OpenAPI fields are already camelCase
			WithDocumentation(documentation).
			WithDeprecated(p.Deprecated).
			WithOptional(openapiFieldIsOptional(p)).
			WithTypez(typez).
			WithTypezID(typezID).
			WithAutoPopulated(openapiIsAutoPopulated(typez, schema, p)).
			WithBehavior(openapiParameterBehavior(p)...).
			Build()
		message.Fields = append(message.Fields, field)
		fieldNames[p.Name] = true
	}

	bodyFieldPath := ""
	if operation.RequestBody != nil {
		reference, err := findReferenceInContentMap(operation.RequestBody.Content)
		if err != nil {
			return nil, "", err
		}
		bid := fmt.Sprintf(".%s.%s", packageName, strings.TrimPrefix(reference, "#/components/schemas/"))
		if a.Message(bid) == nil {
			return nil, "", fmt.Errorf("cannot find referenced type (%s) in API messages", reference)
		}
		name, err := openapiBodyFieldName(fieldNames)
		if err != nil {
			return nil, "", err
		}
		bodyFieldPath = name
		field := api.NewFieldBuilder().
			WithName(name).
			WithID(fmt.Sprintf("%s.%s", message.ID, name)).
			WithJSONName(name).
			WithDocumentation("The request body.").
			WithTypez(api.TypezMessage).
			WithTypezID(bid).
			WithOptional(true).
			Build()
		message.Fields = append(message.Fields, field)
	}
	// Add the message to the symbol table and the parent.
	parent.Messages = append(parent.Messages, message)
	a.AddMessage(message)

	return message, bodyFieldPath, nil
}

func openapiBodyFieldName(fieldNames map[string]bool) (string, error) {
	if _, ok := fieldNames["body"]; ok {
		return "", fmt.Errorf("body is a request or path parameter")
	}
	return "body", nil
}

func openapiFieldIsOptional(p *v3.Parameter) bool {
	return p.Required == nil || !*p.Required
}

func openapiIsAutoPopulated(typez api.Typez, schema *base.Schema, p *v3.Parameter) bool {
	return typez == api.TypezString && schema.Format == "uuid" && openapiFieldIsOptional(p)
}

func makeResponseMessage(api *api.API, operation *v3.Operation, packageName string) (*api.Message, error) {
	if operation.Responses == nil {
		return nil, fmt.Errorf("missing Responses in specification for operation %s", operation.OperationId)
	}
	if operation.Responses.Default == nil {
		// Google's OpenAPI v3 specifications only include the "default"
		// response. In the future we may want to support more than this.
		return nil, fmt.Errorf("expected Default response for operation %s", operation.OperationId)
	}
	// TODO(#1590) - support a missing `Content` as an indication of `void`.
	reference, err := findReferenceInContentMap(operation.Responses.Default.Content)
	if err != nil {
		return nil, err
	}
	id := fmt.Sprintf(".%s.%s", packageName, strings.TrimPrefix(reference, "#/components/schemas/"))
	if message := api.Message(id); message != nil {
		return message, nil
	}
	return nil, fmt.Errorf("cannot find response message ref=%s", reference)
}

func findReferenceInContentMap(content *orderedmap.Map[string, *v3.MediaType]) (string, error) {
	for pair := content.Oldest(); pair != nil; pair = pair.Next() {
		if pair.Key != "application/json" {
			continue
		}
		return pair.Value.Schema.GetReference(), nil
	}
	return "", fmt.Errorf("cannot find an application/json content type")
}

func makeQueryParameters(operation *v3.Operation) map[string]bool {
	queryParameters := map[string]bool{}
	for _, p := range operation.Parameters {
		if p.In != "query" {
			continue
		}
		queryParameters[p.Name] = true
	}
	return queryParameters
}

func makeMessageFields(model *api.API, packageName, messageName string, message *base.Schema) ([]*api.Field, error) {
	var fields []*api.Field
	for name, f := range message.Properties.FromOldest() {
		schema, err := f.BuildSchema()
		if err != nil {
			return nil, err
		}
		optional := !slices.Contains(message.Required, name)
		field, err := makeField(model, packageName, messageName, name, optional, schema)
		if err != nil {
			return nil, err
		}
		fields = append(fields, field)
	}
	return fields, nil
}

func makeField(model *api.API, packageName, messageName, name string, optional bool, field *base.Schema) (*api.Field, error) {
	if len(field.AllOf) != 0 {
		// Simple object fields name an AllOf attribute, but no `Type` attribute.
		return makeObjectField(model, packageName, messageName, name, false, field)
	}
	if len(field.Type) == 0 {
		return nil, fmt.Errorf("missing field type for field %s.%s", messageName, name)
	}
	switch field.Type[0] {
	case "boolean", "integer", "number", "string":
		return makeScalarField(packageName, messageName, name, field, optional, false, field)
	case "object":
		return makeObjectField(model, packageName, messageName, name, false, field)
	case "array":
		return makeArrayField(model, packageName, messageName, name, field)
	default:
		return nil, fmt.Errorf("unknown type for field %q", name)
	}
}

func makeScalarField(packageName, messageName, name string, schema *base.Schema, optional, repeated bool, field *base.Schema) (*api.Field, error) {
	typez, typezID, err := scalarType(messageName, name, schema)
	if err != nil {
		return nil, err
	}
	return api.NewFieldBuilder().
		WithName(name).
		WithID(fmt.Sprintf(".%s.%s.%s", packageName, messageName, name)).
		WithJSONName(name). // OpenAPI field names are always camelCase
		WithDocumentation(field.Description).
		WithTypez(typez).
		WithTypezID(typezID).
		WithDeprecated(field.Deprecated != nil && *field.Deprecated).
		WithOptional(!repeated && (optional || (typez == api.TypezMessage))).
		WithRepeated(repeated).
		Build(), nil
}

func makeObjectField(model *api.API, packageName, messageName, name string, repeated bool, field *base.Schema) (*api.Field, error) {
	if len(field.AllOf) != 0 {
		return makeObjectFieldAllOf(packageName, messageName, name, repeated, field)
	}
	if field.AdditionalProperties != nil && field.AdditionalProperties.IsA() {
		// This indicates we have a map<K, T> field. In OpenAPI, these are
		// simply JSON objects, maybe with a restrictive value type.
		schema, err := field.AdditionalProperties.A.BuildSchema()
		if err != nil {
			return nil, fmt.Errorf("cannot build schema for field %s.%s: %w", messageName, name, err)
		}

		if len(schema.Type) == 0 {
			// Untyped message fields are .google.protobuf.Any
			return api.NewFieldBuilder().
				WithName(name).
				WithID(fmt.Sprintf(".%s.%s.%s", packageName, messageName, name)).
				WithJSONName(name). // OpenAPI field names are always camelCase
				WithDocumentation(field.Description).
				WithDeprecated(field.Deprecated != nil && *field.Deprecated).
				WithTypez(api.TypezMessage).
				WithTypezID(".google.protobuf.Any").
				WithOptional(!repeated).
				WithRepeated(repeated).
				Build(), nil
		}
		mapEntry, err := makeMap(model, messageName, name, schema)
		if err != nil {
			return nil, err
		}
		return api.NewFieldBuilder().
			WithName(name).
			WithID(fmt.Sprintf(".%s.%s.%s", packageName, messageName, name)).
			WithJSONName(name). // OpenAPI field names are always camelCase
			WithDocumentation(field.Description).
			WithDeprecated(field.Deprecated != nil && *field.Deprecated).
			WithTypez(api.TypezMap).
			WithTypezID(mapEntry.ID()).
			WithOptional(false).
			WithRepeated(repeated).
			WithMap(!repeated).
			Build(), nil
	}
	if field.Items != nil && field.Items.IsA() {
		proxy := field.Items.A
		typezID := fmt.Sprintf(".%s.%s", packageName, strings.TrimPrefix(proxy.GetReference(), "#/components/schemas/"))
		return api.NewFieldBuilder().
			WithName(name).
			WithID(fmt.Sprintf(".%s.%s.%s", packageName, messageName, name)).
			WithJSONName(name). // OpenAPI field names are always camelCase
			WithDocumentation(field.Description).
			WithDeprecated(field.Deprecated != nil && *field.Deprecated).
			WithTypez(api.TypezMessage).
			WithTypezID(typezID).
			WithOptional(!repeated).
			WithRepeated(repeated).
			Build(), nil
	}
	return nil, fmt.Errorf("unknown object field type for field %s.%s", messageName, name)
}

func makeArrayField(model *api.API, packageName, messageName, name string, field *base.Schema) (*api.Field, error) {
	if !field.Items.IsA() {
		return nil, fmt.Errorf("cannot handle arrays without an `Items` field for %s.%s", messageName, name)
	}
	reference := field.Items.A.GetReference()
	schema, err := field.Items.A.BuildSchema()
	if err != nil {
		return nil, fmt.Errorf("cannot build items schema for %s.%s error=%q", messageName, name, err)
	}
	if len(schema.Type) != 1 {
		return nil, fmt.Errorf("the items for field  %s.%s should have a single type", messageName, name)
	}
	switch schema.Type[0] {
	case "boolean", "integer", "number", "string":
		return makeScalarField(packageName, messageName, name, schema, false, true, field)
	case "object":
		if reference != "" {
			typezID := fmt.Sprintf(".%s.%s", packageName, strings.TrimPrefix(reference, "#/components/schemas/"))
			return api.NewFieldBuilder().
				WithName(name).
				WithID(fmt.Sprintf(".%s.%s.%s", packageName, messageName, name)).
				WithJSONName(name). // OpenAPI field names are always camelCase
				WithDocumentation(field.Description).
				WithDeprecated(field.Deprecated != nil && *field.Deprecated).
				WithTypez(api.TypezMessage).
				WithTypezID(typezID).
				WithRepeated(true).
				Build(), nil
		}
		return makeObjectField(model, packageName, messageName, name, true, schema)
	default:
		return nil, fmt.Errorf("unknown array field type for %s.%s %q", messageName, name, schema.Type[0])
	}
}

func makeObjectFieldAllOf(packageName, messageName, name string, repeated bool, field *base.Schema) (*api.Field, error) {
	for _, proxy := range field.AllOf {
		typezID := fmt.Sprintf(".%s.%s", packageName, strings.TrimPrefix(proxy.GetReference(), "#/components/schemas/"))
		return api.NewFieldBuilder().
			WithName(name).
			WithID(fmt.Sprintf(".%s.%s.%s", packageName, messageName, name)).
			WithJSONName(name). // OpenAPI field names are always camelCase
			WithDocumentation(field.Description).
			WithDeprecated(field.Deprecated != nil && *field.Deprecated).
			WithTypez(api.TypezMessage).
			WithTypezID(typezID).
			WithOptional(!repeated).
			WithRepeated(repeated).
			Build(), nil
	}
	return nil, fmt.Errorf("cannot build any AllOf schema for field %s.%s", messageName, name)
}

func makeMap(model *api.API, messageName, name string, schema *base.Schema) (*api.Map, error) {
	valueTypez, valueID, err := scalarType(messageName, name, schema)
	if err != nil {
		return nil, err
	}
	id := fmt.Sprintf("$map<string, %s>", valueID)
	mapEntry := model.Map(id)
	if mapEntry == nil {
		// The map was not found, insert the type.
		key := api.NewFieldBuilder().
			WithName("key").
			WithID(id + ".key").
			WithTypez(api.TypezString).
			WithTypezID("string").
			Build()
		value := api.NewFieldBuilder().
			WithName("value").
			WithID(valueID).
			WithTypez(valueTypez).
			WithTypezID(valueID).
			Build()
		placeholder := api.NewMapBuilder().
			WithID(id).
			WithKey(key).
			WithValue(value).
			Build()
		model.AddMap(placeholder)
		mapEntry = placeholder
	}
	return mapEntry, nil
}

func scalarType(messageName, name string, schema *base.Schema) (api.Typez, string, error) {
	for _, type_name := range schema.Type {
		switch type_name {
		case "boolean":
			return api.TypezBool, "bool", nil
		case "integer":
			return scalarTypeForIntegerFormats(messageName, name, schema)
		case "number":
			return scalarTypeForNumberFormats(messageName, name, schema)
		case "string":
			return scalarTypeForStringFormats(messageName, name, schema)
		}
	}
	return 0, "", fmt.Errorf("expected a scalar type for field %s.%s", messageName, name)
}

func scalarTypeForIntegerFormats(messageName, name string, schema *base.Schema) (api.Typez, string, error) {
	switch schema.Format {
	case "int32":
		if schema.Minimum != nil && *schema.Minimum == 0 {
			return api.TypezUint32, "uint32", nil
		}
		return api.TypezInt32, "int32", nil
	case "int64":
		if schema.Minimum != nil && *schema.Minimum == 0 {
			return api.TypezUint64, "uint64", nil
		}
		return api.TypezInt64, "int64", nil
	}
	return 0, "", fmt.Errorf("unknown integer format (%s) for field %s.%s", schema.Format, messageName, name)
}

func scalarTypeForNumberFormats(messageName, name string, schema *base.Schema) (api.Typez, string, error) {
	switch schema.Format {
	case "float":
		return api.TypezFloat, "float", nil
	case "double":
		return api.TypezDouble, "double", nil
	}
	return 0, "", fmt.Errorf("unknown number format (%s) for field %s.%s", schema.Format, messageName, name)
}

func scalarTypeForStringFormats(messageName, name string, schema *base.Schema) (api.Typez, string, error) {
	switch schema.Format {
	case "":
		return api.TypezString, "string", nil
	case "uuid":
		return api.TypezString, "string", nil
	case "byte":
		return api.TypezBytes, "bytes", nil
	case "int32":
		if schema.Minimum != nil && *schema.Minimum == 0 {
			return api.TypezUint32, "uint32", nil
		}
		return api.TypezInt32, "int32", nil
	case "int64":
		if schema.Minimum != nil && *schema.Minimum == 0 {
			return api.TypezUint64, "uint64", nil
		}
		return api.TypezInt64, "int64", nil
	case "google-duration":
		return api.TypezMessage, ".google.protobuf.Duration", nil
	case "date-time":
		return api.TypezMessage, ".google.protobuf.Timestamp", nil
	case "google-fieldmask":
		return api.TypezMessage, ".google.protobuf.FieldMask", nil
	}
	return 0, "", fmt.Errorf("unknown string format (%s) for field %s.%s", schema.Format, messageName, name)
}
