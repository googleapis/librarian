// Copyright 2026 Google LLC
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

package swift

import (
	"github.com/googleapis/librarian/internal/sidekick/api"
)

type serviceAnnotations struct {
	CopyrightYear    string
	BoilerPlate      []string
	Name             string
	DocLines         []string
	RestMethods      []*api.Method
	PackageName      string
	QuickstartMethod *api.Method
}

func (codec *codec) annotateService(service *api.Service, model *modelAnnotations) error {
	docLines := codec.formatDocumentation(service.Documentation)
	var restMethods []*api.Method
	for _, method := range service.Methods {
		if isGeneratedMethod(method) {
			if err := codec.annotateMethod(method, model); err != nil {
				return err
			}
			restMethods = append(restMethods, method)
		}
	}
	var quickstartMethod *api.Method
	if service.QuickstartMethod != nil && isGeneratedMethod(service.QuickstartMethod) {
		quickstartMethod = service.QuickstartMethod
	}
	annotations := &serviceAnnotations{
		CopyrightYear:    model.CopyrightYear,
		BoilerPlate:      model.BoilerPlate,
		Name:             pascalCase(service.Name),
		DocLines:         docLines,
		RestMethods:      restMethods,
		PackageName:      codec.PackageName,
		QuickstartMethod: quickstartMethod,
	}
	service.Codec = annotations
	return nil
}

func isGeneratedMethod(method *api.Method) bool {
	return method.PathInfo != nil && len(method.PathInfo.Bindings) != 0
}
