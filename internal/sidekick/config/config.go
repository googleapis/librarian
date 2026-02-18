// Copyright 2024 Google LLC
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

// Package config provides functionality for working with the sidekick.toml
// configuration file.
package config

// DocumentationOverride describes overrides for the documentation of a single element.
//
// This should be used sparingly. Generally we should prefer updating the
// comments upstream, and then getting a new version of the services'
// specification. The exception may be when the fixes take a long time, or are
// specific to one language.
type DocumentationOverride struct {
	ID      string `toml:"id"`
	Match   string `toml:"match"`
	Replace string `toml:"replace"`
}

// PaginationOverride describes overrides for pagination config of a method.
type PaginationOverride struct {
	// The method ID.
	ID string `toml:"id"`
	// The name of the field used for `items`.
	ItemField string `toml:"item-field"`
}

// Config is the main configuration struct.
type Config struct {
	Discovery           *Discovery              `toml:"discovery,omitempty"`
	CommentOverrides    []DocumentationOverride `toml:"documentation-overrides,omitempty"`
	PaginationOverrides []PaginationOverride    `toml:"pagination-overrides,omitempty"`
}
