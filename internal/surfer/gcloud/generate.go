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
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/googleapis/librarian/internal/sidekick/api"
	"github.com/googleapis/librarian/internal/surfer/provider"
	"github.com/googleapis/librarian/internal/yaml"
	"github.com/iancoleman/strcase"
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

	for _, service := range model.Services {
		// TODO(https://github.com/googleapis/librarian/issues/3291): Ensure output directories don't collide if multiple services share a name.
		if err := generateService(service, overrides, model, output); err != nil {
			return fmt.Errorf("failed to generate commands for service %q: %w", service.Name, err)
		}
	}
	return nil
}

func generateService(service *api.Service, overrides *provider.Config, model *api.API, output string) error {
	shortServiceName, _, found := strings.Cut(service.DefaultHost, ".")
	if !found {
		return fmt.Errorf("failed to determine short service name for service %q: default_host is empty", service.Name)
	}

	surfaceDir := filepath.Join(output, shortServiceName)

	if err := os.MkdirAll(surfaceDir, 0755); err != nil {
		return fmt.Errorf("failed to create surface directory for %q: %w", shortServiceName, err)
	}

	track := strings.ToUpper(provider.InferTrackFromPackage(service.Package))
	data := commandGroupData{
		ServiceTitle:    provider.GetServiceTitle(model, shortServiceName),
		ClassNamePrefix: strcase.ToCamel(shortServiceName),
		Tracks:          []string{track},
	}

	if err := writeCommandGroupFile(surfaceDir, data); err != nil {
		return fmt.Errorf("failed to write command group file for service %q: %w", shortServiceName, err)
	}

	// gcloud commands are resource-centric, so group methods by the resource
	// they operate on.
	methodsByResource := make(map[string][]*api.Method)

	for _, method := range service.Methods {
		collectionID := provider.GetPluralResourceNameForMethod(method, model)

		if collectionID != "" {
			methodsByResource[collectionID] = append(methodsByResource[collectionID], method)
		}
	}

	for collectionID, methods := range methodsByResource {
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
func generateResourceCommands(collectionID string, methods []*api.Method, baseDir string, overrides *provider.Config, model *api.API, service *api.Service) error {
	if len(methods) == 0 {
		return nil
	}

	resourceDir := filepath.Join(baseDir, collectionID)

	if err := os.MkdirAll(resourceDir, 0755); err != nil {
		return fmt.Errorf("failed to create resource directory for %q: %w", collectionID, err)
	}

	singular := provider.GetSingularResourceNameForMethod(methods[0], model)

	shortServiceName, _, _ := strings.Cut(service.DefaultHost, ".")

	track := strings.ToUpper(provider.InferTrackFromPackage(service.Package))
	data := commandGroupData{
		ServiceTitle:     provider.GetServiceTitle(model, shortServiceName),
		ResourceSingular: singular,
		ClassNamePrefix:  strcase.ToCamel(collectionID),
		Tracks:           []string{track},
	}

	if err := writeCommandGroupFile(resourceDir, data); err != nil {
		return fmt.Errorf("failed to write command group file for resource %q: %w", collectionID, err)
	}

	// Partials allow sharing command definitions across release tracks.
	partialsDir := filepath.Join(resourceDir, "_partials")
	if err := os.MkdirAll(partialsDir, 0755); err != nil {
		return fmt.Errorf("failed to create partials directory for %q: %w", collectionID, err)
	}

	for _, method := range methods {
		verb, err := (&provider.MethodAdapter{Method: method}).GetCommandName()
		if err != nil {
			// Continue to the next method if we can't determine a command name,
			// logging the issue might be useful here in the future.
			continue
		}

		cmd, err := NewCommand(method, overrides, model, service)
		if err != nil {
			return err
		}

		// In gcloud convention, the final YAML file must contain a list of commands,
		// even if there is only one.
		cmdList := []*Command{cmd}

		mainCmdPath := filepath.Join(resourceDir, fmt.Sprintf("%s.yaml", verb))
		if err := os.WriteFile(mainCmdPath, []byte(partialsHeader), 0644); err != nil {
			return fmt.Errorf("failed to write main command file for %q: %w", method.Name, err)
		}

		// Generate a partial file for each release track.
		for _, track := range cmd.ReleaseTracks {
			trackName := strings.ToLower(track)
			partialFileName := fmt.Sprintf("_%s_%s.yaml", verb, trackName)
			partialCmdPath := filepath.Join(partialsDir, partialFileName)

			b, err := yaml.Marshal(cmdList)
			if err != nil {
				return fmt.Errorf("failed to marshal partial command for %q: %w", method.Name, err)
			}

			if err := os.WriteFile(partialCmdPath, b, 0644); err != nil {
				return fmt.Errorf("failed to write partial command file for %q: %w", method.Name, err)
			}
		}
	}
	return nil
}

func writeCommandGroupFile(dir string, data commandGroupData) error {
	var buf bytes.Buffer
	if err := commandGroupTemplate.Execute(&buf, data); err != nil {
		return err
	}
	path := filepath.Join(dir, "__init__.py")
	return os.WriteFile(path, buf.Bytes(), 0644)
}

var commandGroupTemplate = template.Must(template.New("__init__.py").Funcs(template.FuncMap{
	"toCamel": strcase.ToCamel,
}).Parse(`# NOTE: This file is autogenerated and should not be edited by hand.
"""Manage {{.ServiceTitle}}{{if .ResourceSingular}} {{.ResourceSingular}}{{end}} resources."""

from googlecloudsdk.calliope import base


{{range .Tracks}}@base.ReleaseTracks(base.ReleaseTrack.{{.}})
@base.Autogenerated
@base.Hidden
class {{$.ClassNamePrefix}}{{. | toCamel}}(base.Group):
  """Manage {{$.ServiceTitle}}{{if $.ResourceSingular}} {{$.ResourceSingular}}{{end}} resources."""


{{end}}`))

type commandGroupData struct {
	ServiceTitle     string
	ResourceSingular string
	ClassNamePrefix  string
	Tracks           []string
}
