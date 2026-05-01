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

// Package gcloud provides a code generator for gcloud commands.
package gcloud

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/iancoleman/strcase"

	"github.com/cbroglie/mustache"
	"github.com/googleapis/librarian/internal/sidekick/api"
	"github.com/googleapis/librarian/internal/sidekick/language"
	"github.com/googleapis/librarian/internal/sidekick/surfer/provider"
)

//go:embed all:templates
var templates embed.FS

// CLIModel represents the data structure for the template.
type CLIModel struct {
	Groups []Group
}

// Group represents a gcloud command group.
type Group struct {
	Name      string
	Usage     string
	Subgroups []Subgroup
	Commands  []Command
}

// Subgroup represents a nested command group.
type Subgroup struct {
	Name     string
	Usage    string
	Commands []Command
}

// Command represents a leaf command.
type Command struct {
	Name  string
	Usage string
}

// Generate generates code from the model.
func Generate(model *api.API, outdir string) error {
	cliModel := constructCLIModel(model)

	templateContents, err := templates.ReadFile("templates/package/cli.go.mustache")
	if err != nil {
		return err
	}

	s, err := mustache.Render(string(templateContents), cliModel)
	if err != nil {
		return err
	}

	destination := filepath.Join(outdir, "main.go")
	if err := os.MkdirAll(filepath.Dir(destination), 0755); err != nil {
		return err
	}
	if err := os.WriteFile(destination, []byte(s), 0666); err != nil {
		return err
	}

	provider := func(name string) (string, error) {
		contents, err := templates.ReadFile(name)
		if err != nil {
			return "", err
		}
		return string(contents), nil
	}
	generatedFiles := []language.GeneratedFile{
		{TemplatePath: "templates/package/README.md.mustache", OutputPath: "README.md"},
	}
	return language.GenerateFromModel(outdir, model, provider, generatedFiles)
}

func constructCLIModel(model *api.API) CLIModel {
	rootGroup := Group{
		Name:  model.Name,
		Usage: fmt.Sprintf("manage %s resources", model.Title),
	}

	subgroups := make(map[string]*Subgroup)

	for _, service := range model.Services {
		for _, method := range service.Methods {
			binding := provider.PrimaryBinding(method)
			if binding == nil {
				continue
			}
			segments := provider.GetLiteralSegments(binding.PathTemplate.Segments)
			if len(segments) == 0 {
				continue
			}

			subgroupName := "default"
			if len(segments) >= 1 {
				subgroupName = strcase.ToKebab(segments[len(segments)-1])
			}

			commandName, _ := provider.GetCommandName(method)
			commandName = strcase.ToKebab(commandName)

			if subgroups[subgroupName] == nil {
				subgroups[subgroupName] = &Subgroup{
					Name:  subgroupName,
					Usage: fmt.Sprintf("Manage %s resources", subgroupName),
				}
			}

			subgroups[subgroupName].Commands = append(subgroups[subgroupName].Commands, Command{
				Name:  commandName,
				Usage: fmt.Sprintf("%s %s", commandName, subgroupName),
			})
		}
	}

	var keys []string
	for k := range subgroups {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		rootGroup.Subgroups = append(rootGroup.Subgroups, *subgroups[k])
	}

	return CLIModel{
		Groups: []Group{rootGroup},
	}
}
