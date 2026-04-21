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
	Path           string
	HTTPMethod     string
	HasBody        bool
	IsBodyWildcard bool
	BodyField      string
	QueryParams    []*api.Field
}

// HasQueryParams returns true if the method's default binding has query parameters
//
// The mustache templates use this to (1) use a `var query` vs. `let query` for the collection of
// query parameters, and (2) generate the query parameter encoder only once, and only if needed.
func (ann *methodAnnotations) HasQueryParams() bool {
	return len(ann.QueryParams) != 0
}

func (codec *codec) annotateMethod(method *api.Method, model *modelAnnotations) (*api.Method, error) {
	if method.InputType != nil {
		if _, err := codec.annotateMessage(method.InputType, model); err != nil {
			return nil, err
		}
	}
	if method.OutputType != nil {
		if _, err := codec.annotateMessage(method.OutputType, model); err != nil {
			return nil, err
		}
	}
	docLines := codec.formatDocumentation(method.Documentation)
	binding := method.PathInfo.Bindings[0]
	path := formatPath(binding.PathTemplate)
	hasBody := method.PathInfo.BodyFieldPath != ""
	isBodyWildcard := method.PathInfo.BodyFieldPath == "*"
	var bodyField string
	if hasBody && !isBodyWildcard {
		bodyField = camelCase(method.PathInfo.BodyFieldPath)
	}
	method.Codec = &methodAnnotations{
		Name:           camelCase(method.Name),
		DocLines:       docLines,
		Path:           path,
		HTTPMethod:     binding.Verb,
		HasBody:        hasBody,
		IsBodyWildcard: isBodyWildcard,
		BodyField:      bodyField,
		QueryParams:    language.QueryParams(method, binding),
	}
	return method, nil
}
