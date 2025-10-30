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
	"os"
	"path/filepath"
	"strings"

	"github.com/googleapis/librarian/internal/sidekick/internal/api"
	"github.com/googleapis/librarian/internal/sidekick/internal/config"
	"github.com/googleapis/librarian/internal/sidekick/internal/config/gcloudyaml"
	"gopkg.in/yaml.v3"
)

// Generate generates gcloud commands from the model.
func Generate(model *api.API, outdir string, cfg *config.Config) error {
	if cfg.Gcloud == nil {
		return fmt.Errorf("gcloud config is missing")
	}
	serviceNameParts := strings.Split(cfg.Gcloud.ServiceName, ".")
	if len(serviceNameParts) == 0 {
		return fmt.Errorf("invalid service name in gcloud.yaml: %s", cfg.Gcloud.ServiceName)
	}
	shortServiceName := serviceNameParts[0]
	surfaceDir := filepath.Join(outdir, shortServiceName, "surface")

	methodsByResource := make(map[string][]*api.Method)
	for _, service := range model.Services {
		for _, method := range service.Methods {
			collectionID := getCollectionID(method, model)
			fmt.Printf("DEBUG: Method: %s, CollectionID: %s\n", method.Name, collectionID)
			if collectionID != "" {
				methodsByResource[collectionID] = append(methodsByResource[collectionID], method)
			}
		}
	}

	for collectionID, methods := range methodsByResource {
		err := generateResourceCommands(collectionID, methods, surfaceDir, cfg)
		if err != nil {
			return err
		}
	}
	return nil
}

func generateResourceCommands(collectionID string, methods []*api.Method, baseDir string, cfg *config.Config) error {
	resourceDir := filepath.Join(baseDir, collectionID)
	if err := os.MkdirAll(resourceDir, 0755); err != nil {
		return fmt.Errorf("failed to create resource directory for %q: %w", collectionID, err)
	}

	for _, method := range methods {
		verb := getVerb(method.Name)
		cmd := newCommand(method, cfg)

		b, err := yaml.Marshal(cmd)
		if err != nil {
			return fmt.Errorf("failed to marshal command for %q: %w", method.Name, err)
		}

		filename := fmt.Sprintf("%s.yaml", verb)
		path := filepath.Join(resourceDir, filename)
		if err := os.WriteFile(path, b, 0644); err != nil {
			return fmt.Errorf("failed to write command file for %q: %w", method.Name, err)
		}
	}
	return nil
}

func newCommand(method *api.Method, cfg *config.Config) *Command {
	rule := findHelpTextRule(method, cfg)
	apiDef := findAPI(method, cfg)
	cmd := &Command{}
	if rule != nil {
		cmd.HelpText = &CommandHelpText{
			Brief:       rule.HelpText.Brief,
			Description: rule.HelpText.Description,
		}
	}
	if apiDef != nil {
		cmd.ReleaseTracks = apiDef.ReleaseTracks
	}
	cmd.Arguments = &Arguments{}
	return cmd
}

func getCollectionID(method *api.Method, model *api.API) string {
	var resourceType string
	if method.InputType != nil {
		for _, field := range method.InputType.Fields {
			if (field.Name == "name" || field.Name == "parent") && field.ResourceReference != nil {
				resourceType = field.ResourceReference.Type
				if resourceType == "" {
					resourceType = field.ResourceReference.ChildType
				}
				break
			}
			// Handle update methods where the resource is in a field named after the resource.
			if msg := field.MessageType; msg != nil && msg.Resource != nil {
				if strings.ToLower(msg.Name) == field.Name {
					resourceType = msg.Resource.Type
					break
				}
			}
		}
	}

	if resourceType == "" {
		return ""
	}

	for _, msg := range model.Messages {
		if msg.Resource != nil && msg.Resource.Type == resourceType {
			return msg.Resource.Plural
		}
	}

	return ""
}

func getVerb(methodName string) string {
	switch {
	case strings.HasPrefix(methodName, "Get"):
		return "describe"
	case strings.HasPrefix(methodName, "List"):
		return "list"
	case strings.HasPrefix(methodName, "Create"):
		return "create"
	case strings.HasPrefix(methodName, "Update"):
		return "update"
	case strings.HasPrefix(methodName, "Delete"):
		return "delete"
	default:
		return ToSnakeCase(methodName)
	}
}

func findAPI(method *api.Method, cfg *config.Config) *gcloudyaml.API {
	if cfg.Gcloud == nil || cfg.Gcloud.APIs == nil {
		return nil
	}
	// TODO: support more than one API.
	if len(cfg.Gcloud.APIs) > 0 {
		return &cfg.Gcloud.APIs[0]
	}
	return nil
}

func findHelpTextRule(method *api.Method, cfg *config.Config) *gcloudyaml.HelpTextRule {
	if cfg.Gcloud == nil || cfg.Gcloud.APIs == nil {
		return nil
	}
	for _, api := range cfg.Gcloud.APIs {
		if api.HelpText == nil {
			continue
		}
		for _, rule := range api.HelpText.MethodRules {
			if rule.Selector == method.MethodFullName() {
				return rule
			}
		}
	}
	return nil
}