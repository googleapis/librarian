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

// Package php provides PHP specific functionality for librarian.
package php

import (
	"context"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/sources"
)

// Generate generates a PHP client library.
func Generate(ctx context.Context, cfg *config.Config, library *config.Library, src *sources.Sources) error {
	// TODO(https://github.com/googleapis/librarian/issues/6629): implement PHP generation
	return nil
}

// Format formats a generated PHP library.
func Format(ctx context.Context, library *config.Library) error {
	// TODO(https://github.com/googleapis/librarian/issues/6629): implement PHP formatting
	return nil
}

// Clean removes all generated code from beneath the given library's
// output directory.
func Clean(library *config.Library) error {
	// TODO(https://github.com/googleapis/librarian/issues/6629): implement PHP cleaning
	return nil
}
