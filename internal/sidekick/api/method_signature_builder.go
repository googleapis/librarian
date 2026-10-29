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

// MethodSignatureBuilder builds a [MethodSignature].
type MethodSignatureBuilder struct {
	s MethodSignature
}

// NewMethodSignatureBuilder creates a new [MethodSignatureBuilder].
func NewMethodSignatureBuilder() *MethodSignatureBuilder {
	return &MethodSignatureBuilder{}
}

// WithNames sets the list of field names included in the signature.
func (b *MethodSignatureBuilder) WithNames(names ...string) *MethodSignatureBuilder {
	b.s.names = names
	return b
}

// WithFields sets the list of resolved fields included in the signature.
func (b *MethodSignatureBuilder) WithFields(fields ...*Field) *MethodSignatureBuilder {
	b.s.fields = fields
	return b
}

// Build returns the constructed [MethodSignature].
func (b *MethodSignatureBuilder) Build() *MethodSignature {
	s := b.s
	return &s
}
