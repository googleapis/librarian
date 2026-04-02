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
	return &commandTreeBuilder{
		model:  model,
		config: config,
	}
}

func (b *commandTreeBuilder) build() (*CommandGroupsByTrack, error) {
	tree := &CommandGroupsByTrack{}

	for _, service := range b.model.Services {
		groupBuilder, err := newCommandGroupBuilder(b.model, service, b.config)
		if err != nil {
			return nil, err
		}

		track := strings.ToUpper(provider.InferTrackFromPackage(service.Package))
		root, err := b.root(tree, track, groupBuilder)
		if err != nil {
			return nil, err
		}

		for _, method := range service.Methods {
			if err := b.insert(root, groupBuilder, method); err != nil {
				return nil, err
			}
		}
	}

	return tree, nil
}

// insert traverses the tree and attaches a command leaf node. It resolves the
// literal path segments of the method and walks the tree, creating missing
// groups if they do not yet exist.
func (b *commandTreeBuilder) insert(root *CommandGroup, groupBuilder *commandGroupBuilder, method *api.Method) error {
	binding := provider.PrimaryBinding(method)
	if binding == nil || provider.IsSingletonFromSegments(binding.PathTemplate.Segments) {
		return nil
	}

	segments := provider.GetLiteralSegments(binding.PathTemplate.Segments)
	if len(segments) == 0 {
		return nil
	}

	curr := root
	for i, seg := range segments {
		if b.isTerminatedSegment(seg) {
			return nil
		}
		isLeaf := i == len(segments)-1
		if b.isFlattenedSegment(seg) && !isLeaf {
			continue
		}

		if curr.Groups[seg] == nil {
			curr.Groups[seg] = groupBuilder.build(segments, i)
		}
		curr = curr.Groups[seg]
	}

	cmd, err := newCommandBuilder(method, b.config, b.model, groupBuilder.service).build()
	if err != nil {
		return err
	}

	curr.Commands[cmd.Name] = cmd
	return nil
}

func (b *commandTreeBuilder) root(tree *CommandGroupsByTrack, track string, groupBuilder *commandGroupBuilder) (*CommandGroup, error) {
	switch track {
	case "GA":
		if tree.GA == nil {
			tree.GA = groupBuilder.buildRoot()
		}
		return tree.GA, nil
	case "BETA":
		if tree.BETA == nil {
			tree.BETA = groupBuilder.buildRoot()
		}
		return tree.BETA, nil
	case "ALPHA":
		if tree.ALPHA == nil {
			tree.ALPHA = groupBuilder.buildRoot()
		}
		return tree.ALPHA, nil
	default:
		return nil, fmt.Errorf("unknown track %q", track)
	}
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
	return lit == "operations" && !provider.ShouldGenerateOperations(b.config)
}
