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

// Filter returns a shallow copy of the API containing cloned services and only
// the top-level messages, enums, and external types whose IDs satisfy keep.
// If keep is nil, all types are retained.
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
	for _, m := range a.Messages() {
		if keep(m.ID()) {
			filteredMsgs = append(filteredMsgs, m)
		}
	}
	var filteredEnums []*Enum
	for _, e := range a.Enums() {
		if keep(e.ID()) {
			filteredEnums = append(filteredEnums, e)
		}
	}

	pkgName := a.PackageName()
	// Collect external top-level types satisfying keep, sorted by ID because
	// AllMessages/AllEnums iterate over maps in randomized order.
	var externalMessages []*Message
	for m := range a.AllMessages() {
		if m.ID() != "" && m.Parent() == nil && m.Package() != pkgName && m.Package() != ReservedPackageName && keep(m.ID()) {
			externalMessages = append(externalMessages, m)
		}
	}
	slices.SortFunc(externalMessages, func(x, y *Message) int { return cmp.Compare(x.ID(), y.ID()) })

	var externalEnums []*Enum
	for e := range a.AllEnums() {
		if e.ID() != "" && e.Parent() == nil && e.Package() != pkgName && e.Package() != ReservedPackageName && keep(e.ID()) {
			externalEnums = append(externalEnums, e)
		}
	}
	slices.SortFunc(externalEnums, func(x, y *Enum) int { return cmp.Compare(x.ID(), y.ID()) })

	origServices := a.Services()
	origQuickstart := a.QuickstartService()
	var services []*Service
	quickstart := origQuickstart
	if origServices != nil {
		services = make([]*Service, len(origServices))
		for i, s := range origServices {
			services[i] = s.Clone()
			if s == origQuickstart {
				quickstart = services[i]
			}
		}
	}

	resourceDefs := a.ResourceDefinitions()
	filtered := &API{
		name:                a.Name(),
		packageName:         pkgName,
		title:               a.Title(),
		description:         a.Description(),
		revision:            a.Revision(),
		services:            services,
		messages:            filteredMsgs,
		externalMessages:    externalMessages,
		enums:               filteredEnums,
		externalEnums:       externalEnums,
		resourceDefinitions: resourceDefs,
		quickstartService:   quickstart,
		rubyPackage:         a.RubyPackage(),
		phpNamespace:        a.PhpNamespace(),
		csharpNamespace:     a.CsharpNamespace(),
	}
	for _, s := range services {
		s.model = filtered
		filtered.addService(s)
		for _, m := range s.Methods() {
			m.model = filtered
			filtered.addMethod(m)
		}
	}
	// Populate lookup maps with all package messages, enums, maps, and
	// resources to avoid aliasing a's maps while still resolving field types
	// across excluded top-level elements.
	for m := range a.AllMessages() {
		filtered.addMessage(m)
	}
	for e := range a.AllEnums() {
		filtered.addEnum(e)
	}
	for m := range a.AllMaps() {
		filtered.addMap(m)
	}
	for _, r := range resourceDefs {
		filtered.addResource(r)
	}
	return filtered
}
