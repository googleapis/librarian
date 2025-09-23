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
	for _, input := range resource.Methods {
		if err := makeMethod(model, service, doc, input); err != nil {
			return err
		}
	}

	return nil
}

func makeMethod(model *api.API, service *api.Service, doc *document, input *method) error {
	id := fmt.Sprintf("%s.%s", service.ID, input.Name)
	if input.MediaUpload != nil {
		return fmt.Errorf("media upload methods are not supported, id=%s", id)
	}
	bodyID, err := getMethodType(model, id, "request type", input.Request)
	if err != nil {
		return err
	}
	outputID, err := getMethodType(model, id, "response type", input.Response)
	if err != nil {
		return err
	}

	// Discovery doc methods get a synthetic request message.
	requestMessage := &api.Message{
		Name:          fmt.Sprintf("%sRequest", input.Name),
		ID:            fmt.Sprintf("%s.%sRequest", service.ID, input.Name),
		Package:       model.PackageName,
		Documentation: fmt.Sprintf("Synthetic request message for the [%s()][%s] method.", input.Name, id[1:]),
		Service:       service,
		// TODO(#2268) - deprecated if method is deprecated.
	}
	model.State.MessageByID[requestMessage.ID] = requestMessage

	var uriTemplate string
	if strings.HasSuffix(doc.ServicePath, "/") {
		uriTemplate = fmt.Sprintf("%s%s", doc.ServicePath, input.Path)
	} else {
		uriTemplate = fmt.Sprintf("%s/%s", doc.ServicePath, input.Path)
	}
	uriTemplate = strings.TrimPrefix(uriTemplate, "/")
	path, err := ParseUriTemplate(uriTemplate)
	if err != nil {
		return err
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
		field, err := makeField(fmt.Sprintf(requestMessage.ID, id), prop)
		if err != nil {
			return err
		}
		field.Synthetic = true
		field.Optional = !p.Required
		requestMessage.Fields = append(requestMessage.Fields, field)
		fieldNames[field.Name] = true
	}

	bodyPathField := ""
	if bodyID != ".google.protobuf.Empty" {
		name := bodyFieldName(fieldNames)
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
	model.State.MethodByID[id] = method
	service.Methods = append(service.Methods, method)
	return nil
}

func bodyFieldName(fieldNames map[string]bool) string {
	// Keep adding trailing `_` until there is no conflict. Most of the time
	// this returns `body` or `body_`
	name := "body"
	for count := 0; count != len(fieldNames); count += 1 {
		if _, ok := fieldNames[name]; !ok {
			return name
		}
		name = name + "_"
	}
	return name
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
