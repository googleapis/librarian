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

package gcloud

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/googleapis/librarian/internal/license"
	"github.com/googleapis/librarian/internal/sidekick/api"
	"github.com/googleapis/librarian/internal/sidekick/surfer"
	"github.com/googleapis/librarian/internal/sidekick/surfer/provider"
	"github.com/iancoleman/strcase"
)

// templateData is the top-level value passed to the templates.
type templateData struct {
	LicenseHeader string
	ModulePath    string
	RootName      string
	RootHelp      string
	Subgroups     []*subgroup
}

// subgroup describes a top-level group under the root command.
type subgroup struct {
	Name       string
	GoFuncName string
	HelpText   string
	Commands   []*command
}

// command describes a leaf gcloud command.
type command struct {
	Name       string
	GoFuncName string
	HelpText   string
	HTTPMethod string
	URLPattern string
	Flags      []*flag
	Positional *flag
	IsAsync    bool
}

// flag describes a single CLI flag or positional argument.
type flag struct {
	CLIName    string
	GoVar      string
	Type       string
	APIField   string
	Help       string
	Required   bool
	Repeated   bool
	Choices    []string
	EnumValues []string
}

// annotate walks the command tree and the underlying model to build templateData.
func annotate(root *surfer.CommandGroup, model *api.API, modulePath string) (*templateData, error) {
	if root == nil {
		return nil, fmt.Errorf("gcloud: command tree has no root")
	}

	methodIndex, err := indexMethods(model)
	if err != nil {
		return nil, err
	}
	rootName := root.ClassName
	if modulePath == "" {
		modulePath = "example.com/" + rootName
	}

	data := &templateData{
		LicenseHeader: licenseHeader(),
		ModulePath:    modulePath,
		RootName:      rootName,
		RootHelp:      sanitizeHelp(root.HelpText),
	}

	for _, name := range sortedKeys(root.Groups) {
		g := root.Groups[name]
		sg, err := buildSubgroup(name, g, methodIndex)
		if err != nil {
			return nil, err
		}
		data.Subgroups = append(data.Subgroups, sg)
	}

	for _, name := range sortedKeys(root.Commands) {
		// Commands directly under the root are unusual but we surface them as
		// a synthetic "default" subgroup so generated code remains uniform.
		c := root.Commands[name]
		if len(data.Subgroups) == 0 || data.Subgroups[0].Name != "root" {
			data.Subgroups = append([]*subgroup{{
				Name:       "root",
				GoFuncName: "Root",
				HelpText:   "Root commands.",
			}}, data.Subgroups...)
		}
		cmd, err := buildCommand("", name, c, methodIndex)
		if err != nil {
			return nil, err
		}
		data.Subgroups[0].Commands = append(data.Subgroups[0].Commands, cmd)
	}

	return data, nil
}

func buildSubgroup(name string, g *surfer.CommandGroup, methodIndex map[string]*api.Method) (*subgroup, error) {
	cliName := snakeToKebab(name)
	sg := &subgroup{
		Name:       cliName,
		GoFuncName: kebabToUpperCamel(cliName),
		HelpText:   sanitizeHelp(g.HelpText),
	}

	for _, cname := range sortedKeys(g.Commands) {
		c := g.Commands[cname]
		cmd, err := buildCommand(name, cname, c, methodIndex)
		if err != nil {
			return nil, err
		}
		sg.Commands = append(sg.Commands, cmd)
	}

	// Flatten nested subgroups by prefixing the group name onto each leaf
	// command's CLI name. parallelstore does not exercise this branch, but
	// keeping it guards against a panic for deeper hierarchies.
	for _, gname := range sortedKeys(g.Groups) {
		nested := g.Groups[gname]
		nestedCommands, err := flattenGroup([]string{gname}, nested, methodIndex)
		if err != nil {
			return nil, err
		}
		sg.Commands = append(sg.Commands, nestedCommands...)
	}
	return sg, nil
}

func flattenGroup(prefix []string, g *surfer.CommandGroup, methodIndex map[string]*api.Method) ([]*command, error) {
	var out []*command
	for _, cname := range sortedKeys(g.Commands) {
		c := g.Commands[cname]
		flatName := strings.Join(append(prefix, snakeToKebab(cname)), "-")
		cmd, err := buildCommand("", flatName, c, methodIndex)
		if err != nil {
			return nil, err
		}
		out = append(out, cmd)
	}
	for _, gname := range sortedKeys(g.Groups) {
		nested, err := flattenGroup(append(prefix, gname), g.Groups[gname], methodIndex)
		if err != nil {
			return nil, err
		}
		out = append(out, nested...)
	}
	return out, nil
}

func buildCommand(groupName, cmdKey string, c *surfer.Command, methodIndex map[string]*api.Method) (*command, error) {
	cliName := snakeToKebab(cmdKey)
	goName := kebabToUpperCamel(snakeToKebab(groupName)) + kebabToUpperCamel(cliName)
	cmd := &command{
		Name:       cliName,
		GoFuncName: goName,
		HelpText:   sanitizeHelp(c.HelpText.Brief),
		IsAsync:    c.Async != nil,
	}

	method := methodIndex[c.Name+"|"+joinCollection(c.Collection)]
	if method == nil {
		method = methodIndex[c.Name]
	}
	if method != nil {
		cmd.HTTPMethod, cmd.URLPattern = httpMethodAndURL(method)
	}

	usedFlagNames := make(map[string]bool)
	for i := range c.Arguments {
		arg := &c.Arguments[i]
		f := buildFlag(arg, usedFlagNames)
		if f == nil {
			continue
		}
		if arg.IsPositional {
			if cmd.Positional == nil {
				cmd.Positional = f
			} else {
				// More than one positional is unusual for gcloud; treat
				// extras as flags so we do not silently drop them.
				cmd.Flags = append(cmd.Flags, f)
			}
			continue
		}
		cmd.Flags = append(cmd.Flags, f)
	}
	return cmd, nil
}

func buildFlag(arg *surfer.Argument, used map[string]bool) *flag {
	cliName := strcase.ToKebab(arg.ArgName)
	if cliName == "" {
		return nil
	}
	if used[cliName] {
		return nil
	}
	used[cliName] = true

	f := &flag{
		CLIName:  cliName,
		GoVar:    kebabToLowerCamel(cliName),
		APIField: strings.Join(arg.APIField, "."),
		Help:     sanitizeHelp(arg.HelpText),
		Required: arg.Required,
		Repeated: arg.Repeated,
	}

	switch {
	case arg.IsPositional:
		f.Type = "string"
	case len(arg.Spec) > 0:
		f.Type = "map"
	case len(arg.Choices) > 0:
		f.Type = "string"
		for _, ch := range arg.Choices {
			f.Choices = append(f.Choices, ch.ArgValue)
			f.EnumValues = append(f.EnumValues, ch.EnumValue)
		}
	default:
		f.Type = mapArgType(arg.Type, arg.Repeated)
	}
	return f
}

func mapArgType(t string, repeated bool) string {
	if repeated {
		return "stringSlice"
	}
	switch t {
	case "long", "int":
		return "int"
	case "double", "float":
		return "float"
	case "bool":
		return "bool"
	case "str", "bytes", "":
		return "string"
	case "arg_object":
		return "string"
	default:
		return "string"
	}
}

// indexMethods returns a map keyed by gcloud command name (with optional
// disambiguation by collection) to the underlying api.Method.
func indexMethods(model *api.API) (map[string]*api.Method, error) {
	out := make(map[string]*api.Method)
	if model == nil {
		return out, nil
	}
	for _, svc := range model.Services {
		for _, m := range svc.Methods {
			name, err := provider.GetCommandName(m)
			if err != nil {
				continue
			}
			out[name] = m
			binding := provider.PrimaryBinding(m)
			if binding != nil {
				out[name+"|"+pathBindingKey(binding)] = m
			}
		}
	}
	return out, nil
}

func pathBindingKey(binding *api.PathBinding) string {
	if binding == nil || binding.PathTemplate == nil {
		return ""
	}
	var parts []string
	for _, seg := range binding.PathTemplate.Segments {
		switch {
		case seg.Literal != nil:
			parts = append(parts, *seg.Literal)
		case seg.Variable != nil:
			parts = append(parts, "{"+strings.Join(seg.Variable.FieldPath, ".")+"}")
		}
	}
	return strings.Join(parts, "/")
}

// joinCollection joins a Command.Collection slice for use as a lookup key.
func joinCollection(collection []string) string {
	return strings.Join(collection, ",")
}

func httpMethodAndURL(m *api.Method) (string, string) {
	binding := provider.PrimaryBinding(m)
	if binding == nil || binding.PathTemplate == nil {
		return "", ""
	}
	method := strings.ToUpper(binding.Verb)
	var sb strings.Builder
	for _, seg := range binding.PathTemplate.Segments {
		sb.WriteString("/")
		switch {
		case seg.Literal != nil:
			sb.WriteString(*seg.Literal)
		case seg.Variable != nil:
			sb.WriteString("{")
			sb.WriteString(strings.Join(seg.Variable.FieldPath, "."))
			sb.WriteString("}")
		}
	}
	if binding.PathTemplate.Verb != nil {
		sb.WriteString(":")
		sb.WriteString(*binding.PathTemplate.Verb)
	}
	return method, sb.String()
}

// sanitizeHelp normalizes help text so it is safe to embed in a Go string
// literal in the generated source.
func sanitizeHelp(help string) string {
	help = strings.TrimSpace(help)
	if help == "" {
		return ""
	}
	help = strings.ReplaceAll(help, "\r", "")
	help = strings.ReplaceAll(help, "\n", " ")
	help = strings.ReplaceAll(help, "\t", " ")
	for strings.Contains(help, "  ") {
		help = strings.ReplaceAll(help, "  ", " ")
	}
	return help
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// licenseHeader returns the Apache 2.0 license header as a comment block,
// suitable for prepending to a generated source file.
func licenseHeader() string {
	year := strconv.Itoa(time.Now().Year())
	lines := license.Header(year)
	var sb strings.Builder
	for _, line := range lines {
		sb.WriteString("//")
		sb.WriteString(line)
		sb.WriteString("\n")
	}
	return sb.String()
}
