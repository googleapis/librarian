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

	"github.com/googleapis/librarian/internal/sidekick/config"
)

// ModelOverrides contains overrides for the API model.
type ModelOverrides struct {
	Name                string
	Title               string
	Description         string
	SkippedIDs          []string
	IncludedIDs         []string
	IncludeList         []string
	ExcludeList         []string
	CommentOverrides    []config.DocumentationOverride
	PaginationOverrides []config.PaginationOverride
	Discovery           *config.Discovery
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
	if val, ok := source["include-list"]; ok {
		m.IncludeList = strings.Split(val, ",")
	}
	if val, ok := source["exclude-list"]; ok {
		m.ExcludeList = strings.Split(val, ",")
	}
	return m
}
