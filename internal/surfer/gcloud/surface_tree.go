// Copyright 2025 Google LLC
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
	"path/filepath"
	"strings"

	"github.com/googleapis/librarian/internal/sidekick/api"
	"github.com/googleapis/librarian/internal/surfer/provider"
)

// SurfaceNode represents a directory level in the generated gcloud command surface.
type SurfaceNode struct {
	Name             string
	Path             string
	Methods          []*provider.MethodAdapter
	Children         map[string]*SurfaceNode
	ServiceTitle     string
	ResourceSingular string
	Tracks           []string
}

// buildSurfaceTree constructs the hierarchical routing tree for a specific service's commands.
func buildSurfaceTree(service *api.Service, methods []*provider.MethodAdapter, model *api.API, outputDir string) *SurfaceNode {
	shortServiceName, _, _ := strings.Cut(service.DefaultHost, ".")
	track := strings.ToUpper(provider.InferTrackFromPackage(service.Package))
	basePath := filepath.Join(outputDir, shortServiceName)

	root := &SurfaceNode{
		Name:         shortServiceName,
		Path:         basePath,
		Children:     make(map[string]*SurfaceNode),
		ServiceTitle: provider.GetServiceTitle(model, shortServiceName),
		Tracks:       []string{track},
	}

	for _, method := range methods {
		collectionID := method.GetPluralResourceName(model)
		if collectionID == "" {
			continue
		}

		// Node traversal logic.
		// For a standard 2-level surface, this maps directly to `root -> collectionID`.
		// As the collection pattern parsing evolves (e.g., AIP-122 deep nesting),
		// this loop will split deeper pattern segments into recursive child nodes.
		child, exists := root.Children[collectionID]
		if !exists {
			childPath := filepath.Join(basePath, collectionID)
			child = &SurfaceNode{
				Name:             collectionID,
				Path:             childPath,
				Children:         make(map[string]*SurfaceNode),
				ServiceTitle:     root.ServiceTitle, // Inherit parent title context
				ResourceSingular: method.GetSingularResourceName(model),
				Tracks:           []string{track},
			}
			root.Children[collectionID] = child
		}

		child.Methods = append(child.Methods, method)
	}

	return root
}
