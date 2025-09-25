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

	"github.com/googleapis/librarian/internal/sidekick/internal/api"
)

func makeServiceMethods(model *api.API, service *api.Service, doc *document, resource *resource) error {
	// It is Okay to reuse the ID, sidekick uses different the namespaces
	// for messages vs. services.
	parent := &api.Message{
		Name:               service.Name,
		ID:                 service.ID,
		Package:            service.Package,
		Documentation:      fmt.Sprintf("Synthetic messages for the [%s][%s] service", service.Name, service.ID[1:]),
		ServicePlaceholder: true,
	}
	model.State.MessageByID[parent.ID] = parent
	model.Messages = append(model.Messages, parent)
	for _, input := range resource.Methods {
		method, err := makeMethod(model, parent, doc, input)
		if err != nil {
			return err
		}
		model.State.MethodByID[method.ID] = method
		service.Methods = append(service.Methods, method)
	}

	return nil
}

func makeMethod(model *api.API, parent *api.Message, doc *document, input *method) (*api.Method, error) {
	id := fmt.Sprintf("%s.%s", parent.ID, input.Name)
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
	requestMessage := &api.Message{
		Name:             fmt.Sprintf("%sRequest", input.Name),
		ID:               fmt.Sprintf("%s.%sRequest", parent.ID, input.Name),
		Package:          model.PackageName,
		SyntheticRequest: true,
		Documentation:    fmt.Sprintf("Synthetic request message for the [%s()][%s] method.", input.Name, id[1:]),
		Parent:           parent,
		// TODO(#2268) - deprecated if method is deprecated.
	}
	model.State.MessageByID[requestMessage.ID] = requestMessage
	parent.Messages = append(parent.Messages, requestMessage)

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

	binding := &api.PathBinding{
		Verb:            input.HTTPMethod,
		PathTemplate:    path,
		QueryParameters: map[string]bool{},
	}
	fieldNames := map[string]bool{}
	for _, p := range input.Parameters {
		if p.Location != "path" {
			binding.QueryParameters[p.Name] = true
		}
		prop := &property{
			Name:   p.Name,
			Schema: &p.schema,
		}
		field, err := makeField(model, requestMessage, prop)
		if err != nil {
			return nil, err
		}
		field.Optional = !p.Required
		requestMessage.Fields = append(requestMessage.Fields, field)
		fieldNames[field.Name] = true
	}

	bodyPathField := ""
	if bodyID != ".google.protobuf.Empty" {
		name, err := bodyFieldName(fieldNames)
		if err != nil {
			return nil, err
		}
		body := &api.Field{
			Documentation: fmt.Sprintf("Synthetic request body field for the [%s()][%s] method.", input.Name, id[1:]),
			Name:          name,
			JSONName:      name,
			ID:            fmt.Sprintf("%s.%s", requestMessage.ID, name),
			Typez:         api.MESSAGE_TYPE,
			TypezID:       bodyID,
			Optional:      true,
		}
		requestMessage.Fields = append(requestMessage.Fields, body)
		bodyPathField = name
	}

	method := &api.Method{
		ID:            id,
		Name:          input.Name,
		Documentation: input.Description,
		// TODO(#2268) - handle deprecated methods
		// Deprecated: ...,
		InputTypeID:  requestMessage.ID,
		OutputTypeID: outputID,
		PathInfo: &api.PathInfo{
			Bindings:      []*api.PathBinding{binding},
			BodyFieldPath: bodyPathField,
		},
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
