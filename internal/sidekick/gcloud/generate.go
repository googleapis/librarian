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
	Flags []Flag
	Name  string
	Usage string
}

// HasFlags reports whether the command has any flags.
func (c Command) HasFlags() bool { return len(c.Flags) > 0 }

// Generate is the package entry point. It builds the model, renders main.go,
// writes it, then renders any other generated files via
// language.GenerateFromModel.
func Generate(model *api.API, outdir string) error {
	cliModel := constructCLIModel(model)
	contents, err := renderMain(cliModel)
	if err != nil {
		return err
	}
	if err := writeMain(outdir, contents); err != nil {
		return err
	}
	return renderReadme(outdir, model)
}

// renderMain renders the main.go contents from the CLI model.
func renderMain(model CLIModel) (string, error) {
	templateContents, err := templates.ReadFile("templates/package/cli.go.mustache")
	if err != nil {
		return "", err
	}
	return mustache.Render(string(templateContents), model)
}

func writeMain(outdir, contents string) error {
	destination := filepath.Join(outdir, "main.go")
	if err := os.MkdirAll(filepath.Dir(destination), 0755); err != nil {
		return err
	}
	return os.WriteFile(destination, []byte(contents), 0666)
}

// renderReadme renders README.md via language.GenerateFromModel.
func renderReadme(outdir string, model *api.API) error {
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

			subgroupName := strcase.ToKebab(segments[len(segments)-1])

			commandName, _ := provider.GetCommandName(method)
			commandName = strcase.ToKebab(commandName)

			if subgroups[subgroupName] == nil {
				subgroups[subgroupName] = &Subgroup{
					Name:  subgroupName,
					Usage: fmt.Sprintf("Manage %s resources", subgroupName),
				}
			}

			cmd := buildCommand(method, model, commandName, subgroupName)
			subgroups[subgroupName].Commands = append(subgroups[subgroupName].Commands, cmd)
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

// buildCommand constructs a Command for a method. The command's flags name
// each component of the resource the method operates on.
func buildCommand(method *api.Method, model *api.API, commandName, subgroupName string) Command {
	return Command{
		Name:  commandName,
		Usage: fmt.Sprintf("%s %s", commandName, subgroupName),
		Flags: pathFlagsForMethod(method, model),
	}
}

// pathFlagsForMethod returns the required flags that name each component of
// the resource the method operates on. For collection methods (List,
// Create, custom collection) it walks up to the parent of the resource
// pattern; for resource methods it uses the resource pattern directly.
//
// The flag name is the last component of each variable segment's
// FieldPath, per AIP-123. Resources without a pattern, or methods whose
// resource cannot be resolved, contribute no flags.
func pathFlagsForMethod(method *api.Method, model *api.API) []Flag {
	resource := provider.GetResourceForMethod(method, model)
	if resource == nil || len(resource.Patterns) == 0 {
		return nil
	}
	segments := resource.Patterns[0]
	if provider.IsCollectionMethod(method) {
		if parent := provider.GetParentFromSegments(segments); parent != nil {
			segments = parent
		}
	}
	return pathFlagsFromSegments(segments)
}

// pathFlagsFromSegments returns one required string flag for each variable
// segment in the pattern, named after the variable's last FieldPath
// component. Duplicates (same FieldPath) are skipped.
func pathFlagsFromSegments(segments []api.PathSegment) []Flag {
	var flags []Flag
	seen := map[string]bool{}
	for _, seg := range segments {
		if seg.Variable == nil || len(seg.Variable.FieldPath) == 0 {
			continue
		}
		name := seg.Variable.FieldPath[len(seg.Variable.FieldPath)-1]
		if seen[name] {
			continue
		}
		seen[name] = true
		flags = append(flags, pathFlag(name))
	}
	return flags
}
