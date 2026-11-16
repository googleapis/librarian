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

// SampleInfoBuilder builds a [SampleInfo] instance.
type SampleInfoBuilder struct {
	s SampleInfo
}

// NewSampleInfoBuilder creates a new [SampleInfoBuilder].
func NewSampleInfoBuilder() *SampleInfoBuilder {
	return &SampleInfoBuilder{}
}

// WithResourceNameField sets the field containing the resource name or parent resource name.
func (b *SampleInfoBuilder) WithResourceNameField(field *Field) *SampleInfoBuilder {
	b.s.resourceNameField = field
	return b
}

// WithResourceIDField sets the field containing the resource ID.
func (b *SampleInfoBuilder) WithResourceIDField(field *Field) *SampleInfoBuilder {
	b.s.resourceIDField = field
	return b
}

// WithMessageField sets the field containing the message body to be created or updated.
func (b *SampleInfoBuilder) WithMessageField(field *Field) *SampleInfoBuilder {
	b.s.messageField = field
	return b
}

// WithUpdateMaskField sets the field containing the update mask.
func (b *SampleInfoBuilder) WithUpdateMaskField(field *Field) *SampleInfoBuilder {
	b.s.updateMaskField = field
	return b
}

// WithIsRequestResourceName sets whether the main resource name is a field of the request object itself.
func (b *SampleInfoBuilder) WithIsRequestResourceName(isRequestResourceName bool) *SampleInfoBuilder {
	b.s.isRequestResourceName = isRequestResourceName
	return b
}

// WithIsMessageResourceName sets whether the main resource name is a field of the message body.
func (b *SampleInfoBuilder) WithIsMessageResourceName(isMessageResourceName bool) *SampleInfoBuilder {
	b.s.isMessageResourceName = isMessageResourceName
	return b
}

// Build returns the constructed [SampleInfo].
func (b *SampleInfoBuilder) Build() *SampleInfo {
	s := b.s
	return &s
}
