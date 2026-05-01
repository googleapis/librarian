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

package surfer

import (
	"github.com/googleapis/librarian/internal/sidekick/api"
	"github.com/googleapis/librarian/internal/sidekick/surfer/provider"
)

func buildSurface(model *api.API, config *provider.Config) (*Surface, error) {
	var root *CommandGroup
	providerTracks := provider.Tracks(provider.APIVersionFromModel(model))
	var tracks []ReleaseTrack
	for _, t := range providerTracks {
		tracks = append(tracks, ReleaseTrack(t))
	}

	for _, service := range model.Services {
		gb := newGroupBuilder(model, service, config)

		if root == nil {
			root = gb.buildRoot()
		}

		for _, method := range service.Methods {
			if err := insert(root, gb, method); err != nil {
				return nil, err
			}
		}
	}

	return &Surface{Root: root, Tracks: tracks}, nil
}

// insert traverses the tree and attaches a command leaf node. It resolves the
// literal path segments of the method and walks the tree, creating missing
// groups if they do not yet exist.
func insert(root *CommandGroup, gb *groupBuilder, method *api.Method) error {
	if provider.IsSingletonResourceMethod(method, gb.model) {
		return nil
	}

	binding := provider.PrimaryBinding(method)
	if binding == nil {
		return nil
	}

	segments := provider.GetLiteralSegments(binding.PathTemplate.Segments)
	if len(segments) == 0 {
		return nil
	}

	curr := root
	for i, seg := range segments {
		if isTerminatedSegment(seg, gb.config) {
			return nil
		}
		isLeaf := i == len(segments)-1
		if flattenedSegments[seg] && !isLeaf {
			continue
		}

		if curr.Groups[seg] == nil {
			curr.Groups[seg] = gb.build(segments[:i+1])
		}
		curr = curr.Groups[seg]
	}

	cmd, err := newCommandBuilder(method, gb.config, gb.model, gb.service).build()
	if err != nil {
		return err
	}

	curr.Commands[cmd.Name] = cmd
	return nil
}

var flattenedSegments = map[string]bool{
	"projects":      true,
	"locations":     true,
	"zones":         true,
	"regions":       true,
	"folders":       true,
	"organizations": true,
}

func isTerminatedSegment(lit string, config *provider.Config) bool {
	return lit == "operations" && !provider.ShouldGenerateOperations(config)
}
