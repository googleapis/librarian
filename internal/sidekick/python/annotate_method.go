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

package python

import (
	"github.com/googleapis/librarian/internal/sidekick/api"
)

// methodAnnotations decorates api.Method with Python-specific metadata.
type methodAnnotations struct {
	Service        *serviceAnnotations
	Method         *api.Method
	Name           string
	ProtoName      string
	DocLines       []string
	InputTypeName  string
	OutputTypeName string
	IsStreaming    bool
	IsPaged        bool
	IsLRO          bool
	IsMixin        bool
	Pager          *pagerAnnotations
	RestMethod     *restMethodAnnotation
}

func (c *codec) annotateMethod(method *api.Method, service *serviceAnnotations) error {
	docLines := formatDocLines(method.Documentation)
	inputTypeName := ""
	if method.InputType != nil {
		inputTypeName = method.InputType.Name
	}
	outputTypeName := ""
	if method.OutputType != nil {
		outputTypeName = method.OutputType.Name
	}
	var svc *api.Service
	if service != nil {
		svc = service.Service
	}
	isMixin := isMixin(method, svc)
	ann := &methodAnnotations{
		Service:        service,
		Method:         method,
		Name:           pythonIdentifier(snakeCase(method.Name)),
		ProtoName:      method.Name,
		DocLines:       docLines,
		InputTypeName:  inputTypeName,
		OutputTypeName: outputTypeName,
		IsStreaming:    method.ServerSideStreaming || method.ClientSideStreaming || method.IsStreaming,
		IsPaged:        method.Pagination != nil && !isMixin,
		IsLRO:          method.OperationInfo != nil || method.IsLRO,
		IsMixin:        isMixin,
		RestMethod:     annotateRestMethod(method),
	}
	method.Codec = ann
	return nil
}

// isMixin returns true if m originates from a different service than the host service.
func isMixin(m *api.Method, service *api.Service) bool {
	return m != nil && m.SourceServiceID != "" && service != nil && m.SourceServiceID != service.ID
}
