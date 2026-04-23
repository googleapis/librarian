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
	"github.com/googleapis/librarian/internal/sidekick/language"
)

type methodAnnotations struct {
	Name           string
	DocLines       []string
	PathVariables  []*pathVariable
	PathExpression string
	HTTPMethod     string
	HasBody        bool
	IsBodyWildcard bool
	BodyField      string
	QueryParams    []*api.Field
}

// pathVariable describes a variable used to build a request URL path.
//
// Most services have a single path variable, something like `request.parent` or `request.name`,
// where the field is a (required) string.
//
// In general they can take more complex forms, including:
//   - `request.secret.name` where `secret` is optional and name is a string, typically a full
//     resource name.
//   - `request.name` where `name` is an optional string (common in OpenAPI and discovery docs).
//   - `request.value` where the value is some enum, or integer field.
//   - `request.project` and `request.resource` where each is a string and are combined to construct
//     the path (again, common in OpenAPI and discovery docs).
//
// And of course all of these can be combined, such as nested fields that point to enums or nested
// fields that point to nested fields.
type pathVariable struct {
	Name       string
	Expression string
	Test       string
}

// HasQueryParams returns true if the method's default binding has query parameters
//
// The mustache templates use this to (1) use a `var query` vs. `let query` for the collection of
// query parameters, and (2) generate the query parameter encoder only once, and only if needed.
func (ann *methodAnnotations) HasQueryParams() bool {
	return len(ann.QueryParams) != 0
}

func (c *codec) annotateMethod(method *api.Method, modelAnn *modelAnnotations) error {
	if method.InputType != nil {
		if err := c.annotateMessage(method.InputType, modelAnn); err != nil {
			return err
		}
	}
	if method.OutputType != nil {
		if err := c.annotateMessage(method.OutputType, modelAnn); err != nil {
			return err
		}
	}
	docLines := c.formatDocumentation(method.Documentation)
	binding := method.PathInfo.Bindings[0]
	hasBody := method.PathInfo.BodyFieldPath != ""
	isBodyWildcard := method.PathInfo.BodyFieldPath == "*"
	var bodyField string
	if hasBody && !isBodyWildcard {
		bodyField = camelCase(method.PathInfo.BodyFieldPath)
	}
	pathVariables, err := c.pathVariables(method.InputType, binding.PathTemplate)
	if err != nil {
		return err
	}
	method.Codec = &methodAnnotations{
		Name:           camelCase(method.Name),
		DocLines:       docLines,
		PathExpression: pathExpression(binding.PathTemplate),
		PathVariables:  pathVariables,
		HTTPMethod:     binding.Verb,
		HasBody:        hasBody,
		IsBodyWildcard: isBodyWildcard,
		BodyField:      bodyField,
		QueryParams:    language.QueryParams(method, binding),
	}
	return nil
}
