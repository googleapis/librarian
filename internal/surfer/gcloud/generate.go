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
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/googleapis/librarian/internal/sidekick/api"
	"github.com/googleapis/librarian/internal/surfer/gcloud/utils"
	"gopkg.in/yaml.v3"
)

// Generate generates gcloud commands for a service.
func Generate(ctx context.Context, googleapis, gcloudconfig, output, includeList string) error {
	overrides, err := readGcloudConfig(gcloudconfig)
	if err != nil {
		return err
	}

	model, err := createAPIModel(googleapis, includeList)
	if err != nil {
		return err
	}

	if len(model.Services) == 0 {
		return fmt.Errorf("no services found in the provided protos")
	}

	for _, service := range model.Services {
		// TODO(https://github.com/googleapis/librarian/issues/3291): Ensure output directories don't collide if multiple services share a name.
		if err := generateService(service, overrides, model, output); err != nil {
			return fmt.Errorf("failed to generate commands for service %q: %w", service.Name, err)
		}
	}
	return nil
}

func generateService(service *api.Service, overrides *Config, model *api.API, output string) error {
	// Determine short service name for directory structure.
	// The `shortServiceName` is derived from `service.DefaultHost` (e.g., "parallelstore.googleapis.com" -> "parallelstore").
	// `service.DefaultHost`  matches the name field in the service config file
	// (e.g., `default_host` for parallelstore is derived from `parallelstore_v1.yaml` name field).
	shortServiceName := ""
	hostParts := strings.Split(service.DefaultHost, ".")
	if len(hostParts) > 0 {
		shortServiceName = hostParts[0]
	}

	if shortServiceName == "" {
		return fmt.Errorf("failed to determine short service name for service %q: default_host is empty", service.Name)
	}

	// The final output will be placed in a directory structure like:
	// `{outdir}/{shortServiceName}/`
	surfaceDir := filepath.Join(output, shortServiceName)

	// gcloud commands are resource-centric commands (e.g., `gcloud parallelstore instances create`),
	// so we first need to group all the API methods by the resource they operate on.
	// We'll create a map where the key is the resource's collection ID (e.g., "instances")
	// and the value is a list of methods that act on that resource.
	methodsByResource := make(map[string][]*api.Method)

	for _, method := range service.Methods {
		// For each method, we determine the plural name of the resource it operates on.
		// This plural name (e.g., "instances") will serve as our collection ID.
		// Example: For the `CreateInstance` method, this will return "instances".
		collectionID := utils.GetPluralResourceNameForMethod(method, model)

		// If a collection ID is found, we add the method to our map.
		if collectionID != "" {
			methodsByResource[collectionID] = append(methodsByResource[collectionID], method)
		}
	}

	// Now that we have grouped the methods by resource, we can generate the
	// command files for each resource.
	for collectionID, methods := range methodsByResource {
		// The `generateResourceCommands` function will handle the creation of the
		// directory structure and YAML files for this specific resource.
		err := generateResourceCommands(collectionID, methods, surfaceDir, overrides, model, service)
		if err != nil {
			return err
		}
	}
	return nil
}

// generateResourceCommands creates the directory structure and YAML files for a
// single resource's commands (e.g., create, delete, list).
//
// For a given collectionID like "instances", this function will create a directory
// `instances/` and populate it with `create.yaml`, `delete.yaml`, etc.
func generateResourceCommands(collectionID string, methods []*api.Method, baseDir string, overrides *Config, model *api.API, service *api.Service) error {
	// The main directory for the resource is named after its collection ID.
	// Example: `{baseDir}/instances`
	resourceDir := filepath.Join(baseDir, collectionID)

	// Gcloud commands are defined in a `_partials` directory. This allows
	// for sharing command definitions across different release tracks (GA, Beta, Alpha).
	partialsDir := filepath.Join(resourceDir, "_partials")
	if err := os.MkdirAll(partialsDir, 0755); err != nil {
		return fmt.Errorf("failed to create partials directory for %q: %w", collectionID, err)
	}

	// We iterate through each method associated with this resource.
	for _, method := range methods {
		// We map the API method name to a standard gcloud command verb.
		// Example: `CreateInstance` -> "create"
		verb, err := utils.GetVerb(method.Name)
		if err != nil {
			// Continue to the next method if we can't determine a verb,
			// logging the issue might be useful here in the future.
			continue
		}

		// We construct the complete command definition from the API method.
		// This involves generating all the arguments, help text, and request details.
		cmd, err := NewCommand(method, overrides, model, service)
		if err != nil {
			return err
		}

		// in gcloud convention, the final YAML file must contain a list of commands,
		// even if there is only one.
		cmdList := []*Command{cmd}

		// We create the main command file (e.g., `create.yaml`). This file doesn't
		// contain the command definition itself, but rather a directive that tells
		// gcloud to look in the `_partials` directory.
		mainCmdPath := filepath.Join(resourceDir, fmt.Sprintf("%s.yaml", verb))
		if err := os.WriteFile(mainCmdPath, []byte("_PARTIALS_: true\n"), 0644); err != nil {
			return fmt.Errorf("failed to write main command file for %q: %w", method.Name, err)
		}

		// Generate a partial file for each release track.
		for _, track := range cmd.ReleaseTracks {
			trackName := strings.ToLower(track)
			partialFileName := fmt.Sprintf("_%s_%s.yaml", verb, trackName)
			partialCmdPath := filepath.Join(partialsDir, partialFileName)

			// We marshal the command definition struct into YAML format.
			b, err := yaml.Marshal(cmdList)
			if err != nil {
				return fmt.Errorf("failed to marshal partial command for %q: %w", method.Name, err)
			}

			// Finally, we write the generated YAML to the partial file.
			if err := os.WriteFile(partialCmdPath, b, 0644); err != nil {
				return fmt.Errorf("failed to write partial command file for %q: %w", method.Name, err)
			}
		}
	}
	return nil
}
