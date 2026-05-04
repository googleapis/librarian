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
	"bytes"
	"embed"
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/template"

	"github.com/iancoleman/strcase"

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

// renderMain renders the main.go contents from the CLI model. The template
// output is run through go/format so the golden file is gofmt-stable.
func renderMain(model CLIModel) (string, error) {
	t, err := template.ParseFS(templates, "templates/package/cli.go.tmpl")
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, model); err != nil {
		return "", err
	}
	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		return "", fmt.Errorf("formatting generated main.go: %w", err)
	}
	return string(formatted), nil
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
			if subgroups[subgroupName] == nil {
				subgroups[subgroupName] = &Subgroup{
					Name:  subgroupName,
					Usage: fmt.Sprintf("Manage %s resources", subgroupName),
				}
			}

			commandName, _ := provider.GetCommandName(method)
			commandName = strcase.ToKebab(commandName)
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
// each component of the resource the method operates on, and (when the
// resource has any variables) the path is composed at runtime via
// [fmt.Sprintf].
func buildCommand(method *api.Method, model *api.API, commandName, subgroupName string) Command {
	segments := resourceSegments(method, model)
	cmd := Command{
		Name:  commandName,
		Usage: fmt.Sprintf("%s %s", commandName, subgroupName),
		Flags: pathFlagsFromSegments(segments),
	}
	if format := pathFormatFromSegments(segments); format != "" {
		cmd.PathFormat = format
		cmd.Args = pathArgsFromSegments(segments)
		cmd.PathLabel = pathLabel(method)
	}
	return cmd
}

// resourceSegments returns the resource pattern segments for a method, or
// nil when the method's resource cannot be resolved or has no pattern. For
// collection methods (List, Create, custom collection) the pattern is
// trimmed to the parent.
func resourceSegments(method *api.Method, model *api.API) []api.PathSegment {
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
	return segments
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

// pathFormatFromSegments returns a "/"-joined format string with literals
// as themselves and variables as "%s", or "" if there are no variables.
func pathFormatFromSegments(segments []api.PathSegment) string {
	hasVar := false
	var parts []string
	for _, seg := range segments {
		switch {
		case seg.Literal != nil:
			parts = append(parts, *seg.Literal)
		case seg.Variable != nil && len(seg.Variable.FieldPath) > 0:
			parts = append(parts, "%s")
			hasVar = true
		}
	}
	if !hasVar {
		return ""
	}
	return strings.Join(parts, "/")
}

// pathArgsFromSegments returns the variable names in segment order, one
// per "%s" position in the format string from pathFormatFromSegments.
func pathArgsFromSegments(segments []api.PathSegment) []string {
	var args []string
	for _, seg := range segments {
		if seg.Variable == nil || len(seg.Variable.FieldPath) == 0 {
			continue
		}
		args = append(args, seg.Variable.FieldPath[len(seg.Variable.FieldPath)-1])
	}
	return args
}

// pathLabel returns the local variable name used in the generated action
// to hold the composed path. Collection methods compose the parent path,
// so the label is "parent"; resource methods compose the resource name.
func pathLabel(method *api.Method) string {
	if provider.IsCollectionMethod(method) {
		return "parent"
	}
	return "name"
}
