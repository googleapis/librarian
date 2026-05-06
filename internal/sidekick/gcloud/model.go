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

// CLIModel is the data passed to the cmd/gcloud/main.go template.
type CLIModel struct {
	// ModulePath is the Go module path used to construct the import path
	// of each surface package, e.g. "cloud.google.com/go/gcloud" yields
	// imports like "cloud.google.com/go/gcloud/internal/generated/parallelstore".
	ModulePath string

	// Surfaces lists every surface registered under the gcloud root
	// command, in the order they should appear in the Commands slice.
	Surfaces []SurfaceRef
}

// SurfaceRef references a generated surface package from the top-level
// main.go.
type SurfaceRef struct {
	// PackageName is both the Go package name and the directory under
	// internal/generated/, e.g. "parallelstore".
	PackageName string
}

// SurfaceModel is the data passed to the surface_commands.go template. One
// surface model is rendered per generated package under internal/generated/.
type SurfaceModel struct {
	// PackageName is the Go package name and the directory under
	// internal/generated/.
	PackageName string

	// Imports holds the Go imports rendered into the surface's
	// commands.go, typically the GAPIC client and proto-Go packages used
	// by ClientCall actions. It is empty when no command in the surface
	// has a ClientCall.
	Imports []Import

	// Group is the surface's command tree: the root command for the
	// surface, with its subgroups and commands beneath.
	Group Group
}

// Import represents a Go import line in a generated file.
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

// constructSurfaceModel builds the data for a single surface's commands.go
// from the given API model. The returned model's PackageName matches the
// API's short name (e.g. "parallelstore"), which is also used as the
// directory name under internal/generated/.
func constructSurfaceModel(model *api.API) SurfaceModel {
	rootGroup := Group{
		Name:  model.Name,
		Usage: fmt.Sprintf("manage %s resources", model.Title),
	}

	var (
		imports       []Import
		goClient      = goClientPackage(model.PackageName)
		hasClientCall = false
		subgroups     = make(map[string]*Subgroup)
	)

	for _, service := range model.Services {
		for _, method := range service.Methods {
			subName, ok := subgroupName(method)
			if !ok {
				continue
			}
			commandName, _ := provider.GetCommandName(method)
			commandName = strcase.ToKebab(commandName)
			cmd := buildCommand(method, model, commandName, subName)
			if call := buildClientCall(method, goClient, cmd.HasPath()); call != nil {
				cmd.ClientCall = call
				hasClientCall = true
			}

			sg, ok := subgroups[subName]
			if !ok {
				sg = &Subgroup{
					Name:  subName,
					Usage: fmt.Sprintf("manage %s resources", subName),
				}
				subgroups[subName] = sg
			}
			sg.Commands = append(sg.Commands, cmd)
		}
	}

	if hasClientCall {
		imports = []Import{
			{Alias: goClient.Alias, Path: goClient.ClientPath},
			{Path: goClient.PbPath},
		}
	}

	keys := make([]string, 0, len(subgroups))
	for k := range subgroups {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		rootGroup.Subgroups = append(rootGroup.Subgroups, *subgroups[k])
	}

	return SurfaceModel{
		PackageName: model.Name,
		Imports:     imports,
		Group:       rootGroup,
	}
}
