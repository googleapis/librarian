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
	method *api.Method
	api    *api.API
	config *provider.Config
}

func newCommandGroupBuilder(method *api.Method, api *api.API, config *provider.Config) *commandGroupBuilder {
	return &commandGroupBuilder{
		method: method,
		api:    api,
		config: config,
	}
}

func (b *commandGroupBuilder) build() (*CommandGroup, error) {
	segments := provider.GetFilteredSegments(b.method)
	if len(segments) == 0 {
		return nil, nil
	}

	return b.buildGroup(segments)
}

func (b *commandGroupBuilder) buildGroup(segments []string) (*CommandGroup, error) {
	if len(segments) == 0 {
		return nil, nil
	}

	name := segments[0]
	group := &CommandGroup{
		Name:     name,
		Tracks:   []string{b.track()},
		HelpText: fmt.Sprintf("Manage %s %s resources.", b.serviceName(), name),
		Groups:   make(map[string]*CommandGroup),
		Commands: make(map[string]*Command),
	}

	if len(segments) == 1 {
		cmd, cmdName, err := b.buildCommand()
		if err != nil {
			return nil, err
		}
		group.Commands[cmdName] = cmd
		return group, nil
	}

	subGroup, err := b.buildGroup(segments[1:])
	if err != nil {
		return nil, err
	}
	if subGroup != nil {
		group.Groups[subGroup.Name] = subGroup
	}

	return group, nil
}

func (b *commandGroupBuilder) buildCommand() (*Command, string, error) {
	name, err := provider.GetCommandName(b.method)
	if err != nil {
		return nil, "", err
	}
	cmd, err := newCommandBuilder(b.method, b.config, b.api, b.method.Service).build()
	if err != nil {
		return nil, "", err
	}
	return cmd, name, nil
}

func (b *commandGroupBuilder) serviceName() string {
	parts := strings.Split(b.method.Service.Name, ".")
	return parts[0]
}

func (b *commandGroupBuilder) track() string {
	return strings.ToUpper(provider.InferTrackFromPackage(b.method.ID))
}
