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

package parser

import (
	"strings"
)

// ModelOverrides contains overrides for the API model.
type ModelOverrides struct {
	// Name overrides the package name.
	Name string
	// Title overrides the API title.
	Title string
	// Description overrides the API description.
	Description string
	// SkippedIDs is a list of element IDs to skip.
	SkippedIDs []string
	// IncludedIDs is a list of element IDs to include.
	IncludedIDs []string
}

// NewModelOverridesFromSource creates a new ModelOverrides from a source map.
// This is used for backward compatibility with the CLI and legacy configurations
// where overrides are mixed with source roots in a map[string]string.
func NewModelOverridesFromSource(source map[string]string) *ModelOverrides {
	m := &ModelOverrides{}
	if val, ok := source["name-override"]; ok {
		m.Name = val
	}
	if val, ok := source["title-override"]; ok {
		m.Title = val
	}
	if val, ok := source["description-override"]; ok {
		m.Description = val
	}
	if val, ok := source["skipped-ids"]; ok {
		m.SkippedIDs = strings.Split(val, ",")
	}
	if val, ok := source["included-ids"]; ok {
		m.IncludedIDs = strings.Split(val, ",")
	}

	return m
}
