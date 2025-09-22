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

	"github.com/googleapis/librarian/internal/sidekick/internal/api"
)

func makeServiceMethods(model *api.API, serviceID string, resource *resource) ([]*api.Method, error) {
	var methods []*api.Method
	for _, input := range resource.Methods {
		method, err := makeMethod(model, serviceID, input)
		if err != nil {
			return nil, err
		}
		methods = append(methods, method)
	}

	return methods, nil
}

func makeMethod(model *api.API, serviceID string, input *method) (*api.Method, error) {
	id := fmt.Sprintf("%s.%s", serviceID, input.Name)
	if input.MediaUpload != nil {
		return nil, fmt.Errorf("media upload methods are not supported, id=%s", id)
	}
	inputID, err := getMethodType(model, id, "request type", input.Request)
	if err != nil {
		return nil, err
	}
	outputID, err := getMethodType(model, id, "response type", input.Response)
	if err != nil {
		return nil, err
	}
	method := &api.Method{
		ID:            id,
		Name:          input.Name,
		Documentation: input.Description,
		// TODO(#1850) - handle deprecated methods
		// Deprecated: ...,
		InputTypeID:  inputID,
		OutputTypeID: outputID,
	}
	return method, nil
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
