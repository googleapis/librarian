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

// Package gcloud provides a code generator for gcloud commands.
package gcloud

import (
	"github.com/googleapis/librarian/internal/sidekick/api"
	"github.com/googleapis/librarian/internal/sidekick/gcloud/provider"
)

// Generate builds a gcloud command tree from the parsed API model and
// overrides and writes the resulting command groups into output under
// baseModule.
func Generate(model *api.API, overrides *provider.Config, output, baseModule string) error {
	tree, err := newSurfaceBuilder(model, overrides).build()
	if err != nil {
		return err
	}
	return writeCommandGroupTree(output, baseModule, tree)
}
