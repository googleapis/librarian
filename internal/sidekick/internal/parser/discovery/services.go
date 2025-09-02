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

func addServiceRecursive(model *api.API, resource *resource) error {
	if len(resource.Methods) != 0 {
		if err := addService(model, resource); err != nil {
			return err
		}
	}
	for _, child := range resource.Resources {
		if err := addServiceRecursive(model, child); err != nil {
			return err
		}
	}
	return nil
}

func addService(model *api.API, resource *resource) error {
	id := fmt.Sprintf(".%s.%s", model.PackageName, resource.Name)
	methods, err := makeServiceMethods(model, id, resource)
	if err != nil {
		return err
	}

	var service *api.Service
	if _, ok := model.State.ServiceByID[id]; !ok {
		service = &api.Service{
			ID:            id,
			Name:          resource.Name,
			Package:       model.PackageName,
			Documentation: fmt.Sprintf("Service for the `%s` resource.", resource.Name),
			Methods:       methods,
		}
		model.Services = append(model.Services, service)
		model.State.ServiceByID[id] = service
	}
	return nil
}
