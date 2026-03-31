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
	model   *api.API
	config  *provider.Config
	service *api.Service
	title   string
}

func newCommandGroupBuilder(model *api.API, service *api.Service, config *provider.Config) (*commandGroupBuilder, error) {
	if service != nil && service.DefaultHost == "" {
		return nil, fmt.Errorf("service %q has empty default host", service.Name)
	}

	title := model.Name
	if service != nil {
		shortServiceName, _, _ := strings.Cut(service.DefaultHost, ".")
		title = provider.GetServiceTitle(model, shortServiceName)
	}

	return &commandGroupBuilder{
		model:   model,
		config:  config,
		service: service,
		title:   title,
	}, nil
}

func (b *commandGroupBuilder) buildRoot() *CommandGroup {
	return &CommandGroup{
		Name:     b.model.Name,
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
		HelpText: fmt.Sprintf("Manage %s %s resources.", b.title, singular),
		Groups:   make(map[string]*CommandGroup),
		Commands: make(map[string]*Command),
	}
}
