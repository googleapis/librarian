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

package gcloud

import (
	"fmt"
	// "os"
	// "path/filepath"

	"github.com/googleapis/librarian/internal/sidekick/internal/api"
	"github.com/googleapis/librarian/internal/sidekick/internal/config"
	// "gopkg.in/yaml.v3"
)

// Generate generates gcloud commands from the model.
func Generate(model *api.API, outdir string, cfg *config.Config) error {
	for _, service := range model.Services {
		for _, method := range service.Methods {
			fmt.Printf("service: %v\n", service)
			fmt.Printf("method: %v\n", method)
		}
	}
	return nil
}
