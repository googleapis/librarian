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

package discovery

import (
	"fmt"
	"strings"

	"github.com/googleapis/librarian/internal/sidekick/api"
)

func makeServiceMethods(model *api.API, service *api.Service, doc *document, resource *resource) error {
	// It is Okay to reuse the ID, sidekick uses different the namespaces
	// for messages vs. services.
	parentBuilder := api.NewMessageBuilder().
		WithName(service.Name).
		WithID(service.ID).
		WithPackage(service.Package).
		WithDocumentation(fmt.Sprintf("Synthetic messages for the [%s][%s] service", service.Name, service.ID[1:])).
		WithServicePlaceholder(true)
	parent := parentBuilder.Build()
	model.AddMessage(parent)
	model.Messages = append(model.Messages, parent)
	for _, input := range resource.Methods {
		method, err := makeMethod(model, parentBuilder, doc, input)
		if err != nil {
			return err
		}
		model.AddMethod(method)
		service.Methods = append(service.Methods, method)
	}

	return nil
}

func makeMethod(model *api.API, parentBuilder *api.MessageBuilder, doc *document, input *method) (*api.Method, error) {
	parent := parentBuilder.Build()
	id := fmt.Sprintf("%s.%s", parent.ID(), input.Name)
	if input.MediaUpload != nil {
		return nil, fmt.Errorf("media upload methods are not supported, id=%s", id)
	}
	bodyID, err := getMethodType(model, id, "request type", input.Request)
	if err != nil {
		return nil, err
	}
	outputID, err := getMethodType(model, id, "response type", input.Response)
	if err != nil {
		return nil, err
	}

	// Discovery doc methods get a synthetic request message.
	requestMessageID := fmt.Sprintf("%s.%sRequest", parent.ID(), input.Name)
	requestBuilder := api.NewMessageBuilder().
		WithName(fmt.Sprintf("%sRequest", input.Name)).
		WithID(requestMessageID).
		WithPackage(model.PackageName).
		WithSyntheticRequest(true).
		WithDocumentation(fmt.Sprintf("Synthetic request message for the [%s()][%s] method.", input.Name, id[1:])).
		WithParent(parent).
		WithDeprecated(input.Deprecated)
	requestMessage := requestBuilder.Build()
	model.AddMessage(requestMessage)
	parentBuilder.AddMessage(requestMessage)

	var uriTemplate string
	if strings.HasSuffix(doc.ServicePath, "/") {
		uriTemplate = fmt.Sprintf("%s%s", doc.ServicePath, input.Path)
	} else {
		uriTemplate = fmt.Sprintf("%s/%s", doc.ServicePath, input.Path)
	}
	uriTemplate = strings.TrimPrefix(uriTemplate, "/")
	path, err := ParseUriTemplate(uriTemplate)
	if err != nil {
		return nil, err
	}

	bindingBuilder := api.NewPathBindingBuilder().
		WithVerb(input.HTTPMethod).
		WithPathTemplate(path).
		WithQueryParameters()
	fieldNames := map[string]bool{}
	for _, p := range input.Parameters {
		if p.Location != "path" {
			bindingBuilder.WithQueryParameter(p.Name)
		}
		prop := &property{
			Name:   p.Name,
			Schema: &p.schema,
		}
		field, err := makeField(model, requestBuilder, prop, !p.Required)
		if err != nil {
			return nil, err
		}
		requestBuilder.AddField(field)
		fieldNames[field.Name()] = true
	}
	binding := bindingBuilder.Build()

	var signatureNames []string
	if len(input.ParameterOrder) != 0 {
		signatureNames = append(signatureNames, input.ParameterOrder...)
	}

	bodyPathField := ""
	if bodyID != ".google.protobuf.Empty" {
		name, err := bodyFieldName(fieldNames)
		if err != nil {
			return nil, err
		}
		body := api.NewFieldBuilder().
			WithDocumentation(fmt.Sprintf("Synthetic request body field for the [%s()][%s] method.", input.Name, id[1:])).
			WithName(name).
			WithJSONName(name).
			WithID(fmt.Sprintf("%s.%s", requestMessageID, name)).
			WithTypez(api.TypezMessage).
			WithTypezID(bodyID).
			WithOptional(true).
			Build()
		requestBuilder.AddField(body)
		bodyPathField = name
		if len(signatureNames) != 0 {
			// The body, if present, is required for signature requests.
			signatureNames = append(signatureNames, name)
		}
	}

	// Methods without path parameters don't get an overload. Typically one of
	// the query parameter is "required" for them. Otherwise the request is some
	// kind of singleton, these are sufficiently rare that missing the
	// convenience overload is not a big problem.
	signatures := []*api.MethodSignature{}
	if len(signatureNames) != 0 {
		signatures = append(signatures, api.NewMethodSignatureBuilder().WithNames(signatureNames...).Build())
	}
	method := &api.Method{
		ID:            id,
		Name:          input.Name,
		Documentation: input.Description,
		Deprecated:    input.Deprecated,
		InputTypeID:   requestMessageID,
		OutputTypeID:  outputID,
		ReturnsEmpty:  outputID == ".google.protobuf.Empty",
		PathInfo: api.NewPathInfoBuilder().
			WithBindings(binding).
			WithBodyFieldPath(bodyPathField).
			Build(),
		Signatures: signatures,
		APIVersion: input.APIVersion,
	}
	return method, nil
}

func bodyFieldName(fieldNames map[string]bool) (string, error) {
	if _, ok := fieldNames["body"]; ok {
		return "", fmt.Errorf("body is a request or path parameter")
	}
	return "body", nil
}

func getMethodType(model *api.API, methodID, name string, typez *schema) (string, error) {
	if typez == nil {
		return ".google.protobuf.Empty", nil
	}
	if typez.Ref == "" {
		return "", fmt.Errorf("expected a ref-like schema for %s in method %s", name, methodID)
	}
	return fmt.Sprintf(".%s.%s", model.PackageName, typez.Ref), nil
}
