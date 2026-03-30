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

type commandGroupBuilder struct {
	model  *api.API
	config *provider.Config
	title  string
}

func newCommandGroupBuilder(model *api.API, config *provider.Config) *commandGroupBuilder {
	// TODO: https://github.com/googleapis/librarian/issues/3033 - [0] is wrong.
	// We are currently just grabbing the first api service.
	// Root command group should be derived from model and nested groups should be derived from
	// the associated service.
	title := model.Name
	if len(model.Services) > 0 {
		shortServiceName, _, _ := strings.Cut(model.Services[0].DefaultHost, ".")
		title = provider.GetServiceTitle(model, shortServiceName)
	}

	return &commandGroupBuilder{
		model:  model,
		config: config,
		title:  title,
	}
}

func (b *commandGroupBuilder) buildRoot() *CommandGroup {
	return &CommandGroup{
		Name:     b.model.Name,
		Tracks:   []string{b.track()},
		HelpText: fmt.Sprintf("Manage %s resources.", b.title),
		Groups:   make(map[string]*CommandGroup),
		Commands: make(map[string]*Command),
	}
}

func (b *commandGroupBuilder) build(segments []string, idx int) *CommandGroup {
	seg := segments[idx]
	singular := seg
	if resName := provider.GetSingularResourceNameForPrefix(b.model, segments[:idx+1]); resName != "" {
		singular = resName
	}

	return &CommandGroup{
		Name:     seg,
		Tracks:   []string{b.track()},
		HelpText: fmt.Sprintf("Manage %s %s resources.", b.title, singular),
		Groups:   make(map[string]*CommandGroup),
		Commands: make(map[string]*Command),
	}
}

func (b *commandGroupBuilder) track() string {
	return strings.ToUpper(provider.InferTrackFromPackage(b.model.PackageName))
}
