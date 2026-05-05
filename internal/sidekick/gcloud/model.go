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
	"slices"
	"sort"

	"github.com/iancoleman/strcase"

	"github.com/googleapis/librarian/internal/sidekick/api"
	"github.com/googleapis/librarian/internal/sidekick/surfer/provider"
)

// CLIModel represents the data structure for the template.
type CLIModel struct {
	// Imports holds the Go imports rendered into the generated main.go.
	Imports []Import

	// Groups holds the top-level gcloud command groups rendered into main.go.
	Groups []Group
}

// Import represents a Go import line in the generated main.go.
type Import struct {
	// Alias is the optional package alias; empty when the import has no
	// alias. For example, "parallelstore" in:
	//
	//	parallelstore "cloud.google.com/go/parallelstore/apiv1"
	Alias string

	// Path is the import path of the package, for example
	// "cloud.google.com/go/parallelstore/apiv1".
	Path string
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

type commandWithSubgroup struct {
	Command  Command
	Subgroup string
}

func constructCLIModel(model *api.API) CLIModel {
	commands := buildCommands(model)
	return CLIModel{
		Imports: clientImports(model.PackageName, commands),
		Groups:  []Group{rootGroup(model, groupBySubgroup(commands))},
	}
}

func buildCommands(model *api.API) []commandWithSubgroup {
	goClient := goClientPackage(model.PackageName)
	var commands []commandWithSubgroup
	for _, service := range model.Services {
		for _, method := range service.Methods {
			subgroup, ok := subgroupName(method)
			if !ok {
				continue
			}
			commandName, _ := provider.GetCommandName(method)
			commandName = strcase.ToKebab(commandName)
			cmd := buildCommand(method, model, commandName, subgroup)
			if call := buildClientCall(method, goClient, cmd.HasPath()); call != nil {
				cmd.ClientCall = call
			}
			commands = append(commands, commandWithSubgroup{Command: cmd, Subgroup: subgroup})
		}
	}
	return commands
}

func groupBySubgroup(cmds []commandWithSubgroup) []Subgroup {
	bySubgroup := make(map[string]*Subgroup)
	for _, c := range cmds {
		sg, ok := bySubgroup[c.Subgroup]
		if !ok {
			sg = &Subgroup{
				Name:  c.Subgroup,
				Usage: fmt.Sprintf("Manage %s resources", c.Subgroup),
			}
			bySubgroup[c.Subgroup] = sg
		}
		sg.Commands = append(sg.Commands, c.Command)
	}
	keys := make([]string, 0, len(bySubgroup))
	for k := range bySubgroup {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	subgroups := make([]Subgroup, 0, len(keys))
	for _, k := range keys {
		subgroups = append(subgroups, *bySubgroup[k])
	}
	return subgroups
}

func clientImports(pkg string, cmds []commandWithSubgroup) []Import {
	hasCall := slices.ContainsFunc(cmds, func(c commandWithSubgroup) bool {
		return c.Command.ClientCall != nil
	})
	if !hasCall {
		return nil
	}
	goClient := goClientPackage(pkg)
	if goClient == nil {
		return nil
	}
	return []Import{
		{Alias: goClient.Alias, Path: goClient.ClientPath},
		{Path: goClient.PbPath},
	}
}

func rootGroup(model *api.API, subgroups []Subgroup) Group {
	return Group{
		Name:      model.Name,
		Usage:     fmt.Sprintf("manage %s resources", model.Title),
		Subgroups: subgroups,
	}
}
