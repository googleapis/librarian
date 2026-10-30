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

func addServiceRecursive(model *api.APIBuilder, doc *document, resource *resource) error {
	if len(resource.Methods) != 0 {
		if err := addService(model, doc, resource); err != nil {
			return err
		}
	}
	for _, child := range resource.Resources {
		if err := addServiceRecursive(model, doc, child); err != nil {
			return err
		}
	}
	return nil
}

func addService(model *api.APIBuilder, doc *document, resource *resource) error {
	id := fmt.Sprintf(".%s.%s", model.PackageName(), resource.Name)
	sb := api.NewServiceBuilder().
		WithID(id).
		WithName(resource.Name).
		WithPackage(model.PackageName()).
		WithDocumentation(fmt.Sprintf("Service for the `%s` resource.", resource.Name)).
		WithDefaultHost(strings.TrimSuffix(strings.TrimPrefix(doc.RootURL, "https://"), "/")).
		WithDeprecated(resource.Deprecated)
	if err := makeServiceMethods(model, sb, doc, resource); err != nil {
		return err
	}
	service := sb.Build()
	model.AppendService(service)
	model.AddService(service)
	return nil
}
