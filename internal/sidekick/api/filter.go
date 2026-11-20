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

package api

import (
	"cmp"
	"slices"
)

// Filter returns a shallow copy of the API with Codec cleared, cloned services,
// and only the top-level messages, enums, and external types whose IDs satisfy
// keep. If keep is nil, all types are retained.
//
// Lookup maps on the returned API retain all messages, enums, maps, and
// resources from the receiver so cross-type references continue to resolve
// without aliasing the receiver's maps.
func (a *API) Filter(keep func(id string) bool) *API {
	if a == nil {
		return nil
	}
	if keep == nil {
		keep = func(string) bool { return true }
	}

	var filteredMsgs []*Message
	for _, m := range a.Messages {
		if keep(m.ID()) {
			filteredMsgs = append(filteredMsgs, m)
		}
	}
	var filteredEnums []*Enum
	for _, e := range a.Enums {
		if keep(e.ID()) {
			filteredEnums = append(filteredEnums, e)
		}
	}

	// Collect external top-level types satisfying keep, sorted by ID because
	// AllMessages/AllEnums iterate over maps in randomized order.
	var externalMessages []*Message
	for m := range a.AllMessages() {
		if m.ID() != "" && m.Parent() == nil && m.Package() != a.PackageName && m.Package() != ReservedPackageName && keep(m.ID()) {
			externalMessages = append(externalMessages, m)
		}
	}
	slices.SortFunc(externalMessages, func(x, y *Message) int { return cmp.Compare(x.ID(), y.ID()) })

	var externalEnums []*Enum
	for e := range a.AllEnums() {
		if e.ID() != "" && e.Parent() == nil && e.Package() != a.PackageName && e.Package() != ReservedPackageName && keep(e.ID()) {
			externalEnums = append(externalEnums, e)
		}
	}
	slices.SortFunc(externalEnums, func(x, y *Enum) int { return cmp.Compare(x.ID(), y.ID()) })

	var services []*Service
	quickstart := a.QuickstartService
	if a.Services != nil {
		services = make([]*Service, len(a.Services))
		for i, s := range a.Services {
			services[i] = s.Clone()
			if s == a.QuickstartService {
				quickstart = services[i]
			}
		}
	}

	filtered := &API{
		Name:                a.Name,
		PackageName:         a.PackageName,
		Title:               a.Title,
		Description:         a.Description,
		Revision:            a.Revision,
		Services:            services,
		Messages:            filteredMsgs,
		ExternalMessages:    externalMessages,
		Enums:               filteredEnums,
		ExternalEnums:       externalEnums,
		ResourceDefinitions: slices.Clone(a.ResourceDefinitions),
		QuickstartService:   quickstart,
		RubyPackage:         a.RubyPackage,
		PhpNamespace:        a.PhpNamespace,
		CsharpNamespace:     a.CsharpNamespace,
	}
	for _, s := range services {
		s.model = filtered
		filtered.AddService(s)
		for _, m := range s.Methods() {
			m.model = filtered
			filtered.AddMethod(m)
		}
	}
	// Populate lookup maps with all package messages, enums, maps, and
	// resources to avoid aliasing a's maps while still resolving field types
	// across excluded top-level elements.
	for m := range a.AllMessages() {
		filtered.AddMessage(m)
	}
	for e := range a.AllEnums() {
		filtered.AddEnum(e)
	}
	for m := range a.AllMaps() {
		filtered.AddMap(m)
	}
	for r := range a.AllResources() {
		filtered.AddResource(r)
	}
	return filtered
}
