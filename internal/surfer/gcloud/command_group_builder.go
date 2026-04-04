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
	"github.com/iancoleman/strcase"
)

type commandGroupBuilder struct {
	model   *api.API
	config  *provider.Config
	service *api.Service
}

func newCommandGroupBuilder(model *api.API, service *api.Service, config *provider.Config) (*commandGroupBuilder, error) {
	return &commandGroupBuilder{
		model:   model,
		config:  config,
		service: service,
	}, nil
}

func (b *commandGroupBuilder) buildRoot() *CommandGroup {
	// TODO (https://github.com/googleapis/librarian/issues/3033): Use service selector
	// to look up the help text from the gcloud config.
	rootName := provider.ResolveRootPackage(b.model)
	return &CommandGroup{
		Name:     rootName,
		Path:     []string{rootName},
		HelpText: fmt.Sprintf("Manage %s resources.", toTitleCase(rootName)),
		Groups:   make(map[string]*CommandGroup),
		Commands: make(map[string]*Command),
	}
}

func (b *commandGroupBuilder) build(segments []string, idx int, parentPath []string) *CommandGroup {
	seg := segments[idx]
	singular := seg
	if resName := provider.GetSingularResourceNameForPrefix(b.model, segments[:idx+1]); resName != "" {
		singular = resName
	}

	path := make([]string, 0, len(parentPath)+1)
	path = append(path, parentPath...)
	path = append(path, seg)

	return &CommandGroup{
		Name:     seg,
		Path:     path,
		HelpText: fmt.Sprintf("Manage %s resources.", toTitleCase(singular)),
		Groups:   make(map[string]*CommandGroup),
		Commands: make(map[string]*Command),
	}
}

// TODO (https://github.com/googleapis/librarian/issues/3414): Move all of the magic
// string manipulation into one location.
// - put all of these helpers in one place
// - make it clear when and where not to use them. Ideally, we shouldn't use
//   them till the presentation layer but help text breaks that pattern.
func toTitleCase(s string) string {
	// Convert to CamelCase first to handle snake_case
	camel := strcase.ToCamel(s)
	var sb strings.Builder
	for i, r := range camel {
		if i > 0 && r >= 'A' && r <= 'Z' {
			sb.WriteByte(' ')
		}
		sb.WriteRune(r)
	}
	return sb.String()
}
