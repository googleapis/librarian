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

func constructCLIModel(model *api.API) CLIModel {
	rootGroup := Group{
		Name:  model.Name,
		Usage: fmt.Sprintf("manage %s resources", model.Title),
	}

	var (
		cliModel      CLIModel
		goClient      = goClientPackage(model.PackageName)
		hasClientCall = false
		subgroups     = make(map[string]*Subgroup)
	)
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

			if call := buildClientCall(method, goClient, cmd.HasPath()); call != nil {
				cmd.ClientCall = call
				hasClientCall = true
			}
			subgroups[subgroupName].Commands = append(subgroups[subgroupName].Commands, cmd)
		}
	}

	if hasClientCall {
		cliModel.Imports = []Import{
			{Alias: goClient.Alias, Path: goClient.ClientPath},
			{Path: goClient.PbPath},
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
	cliModel.Groups = []Group{rootGroup}
	return cliModel
}
