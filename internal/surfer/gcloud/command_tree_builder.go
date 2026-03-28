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
	"strings"

	"github.com/googleapis/librarian/internal/sidekick/api"
	"github.com/googleapis/librarian/internal/surfer/gcloud/provider"
)

type commandTreeBuilder struct {
	model  *api.API
	config *provider.Config
}

func newCommandTreeBuilder(model *api.API, config *provider.Config) *commandTreeBuilder {
	return &commandTreeBuilder{model: model, config: config}
}

func (b *commandTreeBuilder) build() (*CommandGroup, error) {
	if len(b.model.Services) == 0 {
		return nil, fmt.Errorf("no services found in the provided protos")
	}

	root := &CommandGroup{
		Name:     b.name(),
		Tracks:   []string{b.track()},
		HelpText: fmt.Sprintf("Manage %s resources.", b.name()),
		Commands: make(map[string]*Command),
		Groups:   make(map[string]*CommandGroup),
	}

	for _, service := range b.model.Services {
		for _, method := range service.Methods {
			builder := newCommandGroupBuilder(method, b.model, b.config)
			group, err := builder.build()
			if err != nil {
				return nil, err
			}
			if group != nil {
				b.mergeGroup(root, group)
			}
		}
	}

	return root, nil
}

func (b *commandTreeBuilder) mergeGroup(dest, src *CommandGroup) {
	if src == nil {
		return
	}
	if dest.Groups == nil {
		dest.Groups = make(map[string]*CommandGroup)
	}

	existing, ok := dest.Groups[src.Name]
	if !ok {
		dest.Groups[src.Name] = src
		return
	}

	// Merge commands
	for k, v := range src.Commands {
		if existing.Commands == nil {
			existing.Commands = make(map[string]*Command)
		}
		existing.Commands[k] = v
	}

	// Recurse for sub-groups
	for _, g := range src.Groups {
		b.mergeGroup(existing, g)
	}
}

func (b *commandTreeBuilder) name() string {
	return b.model.Name
}

func (b *commandTreeBuilder) track() string {
	return strings.ToUpper(provider.InferTrackFromPackage(b.model.PackageName))
}
