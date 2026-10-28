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

package api

// PathBindingBuilder builds a [PathBinding].
type PathBindingBuilder struct {
	verb            string
	pathTemplate    *PathTemplate
	queryParameters map[string]bool
	targetResource  *TargetResource
}

// NewPathBindingBuilder creates a new [PathBindingBuilder].
func NewPathBindingBuilder() *PathBindingBuilder {
	return &PathBindingBuilder{}
}

// WithVerb sets the HTTP verb for the binding.
func (b *PathBindingBuilder) WithVerb(verb string) *PathBindingBuilder {
	b.verb = verb
	return b
}

// WithPathTemplate sets the path template for the binding.
func (b *PathBindingBuilder) WithPathTemplate(pt *PathTemplate) *PathBindingBuilder {
	b.pathTemplate = pt
	return b
}

// WithQueryParameters sets the query parameters for the binding.
func (b *PathBindingBuilder) WithQueryParameters(params ...string) *PathBindingBuilder {
	b.queryParameters = make(map[string]bool, len(params))
	for _, p := range params {
		b.queryParameters[p] = true
	}
	return b
}

// WithQueryParameter adds a single query parameter to the binding.
func (b *PathBindingBuilder) WithQueryParameter(name string) *PathBindingBuilder {
	if b.queryParameters == nil {
		b.queryParameters = make(map[string]bool)
	}
	b.queryParameters[name] = true
	return b
}

// WithQueryParameterMap sets the query parameter map for the binding.
func (b *PathBindingBuilder) WithQueryParameterMap(params map[string]bool) *PathBindingBuilder {
	b.queryParameters = params
	return b
}

// WithTargetResource sets the target resource for the binding.
func (b *PathBindingBuilder) WithTargetResource(tr *TargetResource) *PathBindingBuilder {
	b.targetResource = tr
	return b
}

// Build constructs a [PathBinding] from the builder.
func (b *PathBindingBuilder) Build() *PathBinding {
	return &PathBinding{
		verb:            b.verb,
		pathTemplate:    b.pathTemplate,
		queryParameters: b.queryParameters,
		targetResource:  b.targetResource,
	}
}
