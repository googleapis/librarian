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

	"github.com/googleapis/librarian/internal/sidekick/api"
	"github.com/googleapis/librarian/internal/surfer/gcloud/provider"
)

type commandTreeBuilder struct {
	model        *api.API
	config       *provider.Config
	groupBuilder *commandGroupBuilder
}

func newCommandTreeBuilder(model *api.API, config *provider.Config) *commandTreeBuilder {
	return &commandTreeBuilder{
		model:        model,
		config:       config,
		groupBuilder: newCommandGroupBuilder(model, config),
	}
}

func (b *commandTreeBuilder) build() (*CommandGroup, error) {
	root := b.groupBuilder.buildRoot()

	for _, service := range b.model.Services {
		if service.DefaultHost == "" {
			return nil, fmt.Errorf("service %q has empty default host", service.Name)
		}
		for _, method := range service.Methods {
			if err := b.insert(root, method, service); err != nil {
				return nil, err
			}
		}
	}

	return root, nil
}

// insert traverses the trie and attaches a command leaf node. It resolves the
// literal path segments of the method and walks the tree, creating missing
// groups if they do not yet exist.
func (b *commandTreeBuilder) insert(root *CommandGroup, method *api.Method, service *api.Service) error {
	segments := b.getLiteralSegments(method)
	if len(segments) == 0 {
		return nil
	}

	curr := root
	for i, seg := range segments {
		if b.isTerminatedSegment(seg) {
			return nil
		}
		if b.isFlattenedSegment(seg) {
			continue
		}

		// TODO: https://github.com/googleapis/librarian/issues/4530 - handle conflicting release tracks.
		// Right now the first set of release tracks wins.
		if curr.Groups[seg] == nil {
			curr.Groups[seg] = b.groupBuilder.build(segments, i)
		}
		curr = curr.Groups[seg]
	}

	cmd, err := newCommandBuilder(method, b.config, b.model, service).build()
	if err != nil {
		return err
	}
	curr.Commands[cmd.Name] = cmd

	return nil
}

func (b *commandTreeBuilder) getLiteralSegments(method *api.Method) []string {
	if method.PathInfo == nil || len(method.PathInfo.Bindings) == 0 {
		return nil
	}
	// TODO: https://github.com/googleapis/librarian/issues/3415 - handle multiple bindings
	return provider.GetLiteralSegments(method.PathInfo.Bindings[0].PathTemplate.Segments)
}

var flattenedSegments = map[string]bool{
	"projects":      true,
	"locations":     true,
	"zones":         true,
	"regions":       true,
	"folders":       true,
	"organizations": true,
}

func (b *commandTreeBuilder) isFlattenedSegment(lit string) bool {
	return flattenedSegments[lit]
}

func (b *commandTreeBuilder) isTerminatedSegment(lit string) bool {
	return lit == "operations" && b.config != nil && !b.config.GenerateOperations
}
