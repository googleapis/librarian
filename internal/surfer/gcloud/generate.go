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

// Package gcloud orchestrates the generation of gcloud surface configurations.
package gcloud

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/googleapis/librarian/internal/sidekick/api"
	"github.com/googleapis/librarian/internal/surfer/provider"
	"github.com/googleapis/librarian/internal/yaml"
)

// partialsHeader is the directive that tells gcloud to look in the `_partials` directory
// for command definitions. This allows for sharing definitions across release tracks.
const partialsHeader = "_PARTIALS_: true\n"

// Generate generates gcloud commands for a service.
func Generate(_ context.Context, googleapis, gcloudconfig, output, includeList string) error {
	overrides, err := provider.ReadGcloudConfig(gcloudconfig)
	if err != nil {
		return err
	}

	model, err := provider.CreateAPIModel(googleapis, includeList)
	if err != nil {
		return err
	}

	if len(model.Services) == 0 {
		return fmt.Errorf("no services found in the provided protos")
	}

	var methods []*provider.MethodAdapter
	for _, service := range model.Services {
		for _, method := range service.Methods {
			methods = append(methods, &provider.MethodAdapter{Method: method})
		}
	}

	for _, service := range model.Services {
		// 1. Generate the Surface Tree from the methods belonging to this service
		var serviceMethods []*provider.MethodAdapter
		for _, m := range methods {
			// Find methods for this service.
			// Currently simplified; typically you'd filter by m.Method.Service.Package == service.Package.
			if m.Method.Service.Package == service.Package {
				serviceMethods = append(serviceMethods, m)
			}
		}

		tree := buildSurfaceTree(service, serviceMethods, model, output)

		// 2. Render the Tree via DFS node traversal
		if err := renderTree(tree, overrides, model, service); err != nil {
			return err
		}
	}

	return nil
}

// renderTree traverses the SurfaceNode hierarchy. Group nodes emit __init__.py,
// leaf methods construct commandBuilder rules and emit YAMLs.
func renderTree(node *SurfaceNode, overrides *provider.Config, model *api.API, service *api.Service) error {
	// Root or intermediate group node.
	groupBuilder := NewCommandGroupBuilder(node)
	if err := groupBuilder.Build(); err != nil {
		return err
	}

	if len(node.Methods) > 0 {
		partialsDir := filepath.Join(node.Path, "_partials")
		if err := os.MkdirAll(partialsDir, 0755); err != nil {
			return fmt.Errorf("failed to create partials directory for %q: %w", node.Name, err)
		}

		for _, method := range node.Methods {
			verb, err := method.GetCommandName()
			if err != nil {
				continue
			}

			cmd, err := NewCommand(method, overrides, model, service)
			if err != nil {
				return err
			}

			cmdList := []*YAMLCommand{EmitCommand(cmd)}
			mainCmdPath := filepath.Join(node.Path, fmt.Sprintf("%s.yaml", verb))
			if err := os.WriteFile(mainCmdPath, []byte(partialsHeader), 0644); err != nil {
				return fmt.Errorf("failed to write main command file for %q: %w", method.Method.Name, err)
			}

			for _, track := range cmd.ReleaseTracks {
				trackName := strings.ToLower(track)
				partialFileName := fmt.Sprintf("_%s_%s.yaml", verb, trackName)
				partialCmdPath := filepath.Join(partialsDir, partialFileName)

				b, err := yaml.Marshal(cmdList)
				if err != nil {
					return fmt.Errorf("failed to marshal partial command for %q: %w", method.Method.Name, err)
				}
				if err := os.WriteFile(partialCmdPath, b, 0644); err != nil {
					return fmt.Errorf("failed to write partial command file for %q: %w", method.Method.Name, err)
				}
			}
		}
	}

	// Recurse children.
	for _, child := range node.Children {
		if err := renderTree(child, overrides, model, service); err != nil {
			return err
		}
	}

	return nil
}
