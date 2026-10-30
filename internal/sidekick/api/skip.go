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

package api

import (
	"fmt"
	"slices"
)

// SkipModelElements prunes the model of any elements that are not desired.
//
// The elements to be pruned are determined by the `overrides`.
//
// If `overrides.IncludedIDs` is set, then any element that is not a dependency
// of one of the listed IDs is pruned.
//
// If `overrides.SkippedIDs` is set, then any element with an ID in this list is
// pruned.
//
// It is an error to specify both `IncludedIDs` and `SkippedIDs`.
func SkipModelElements(model *API, overrides ModelOverride) error {
	if len(overrides.IncludedIDs) > 0 && len(overrides.SkippedIDs) > 0 {
		return fmt.Errorf("both `IncludedIDs` and `SkippedIDs` set. Only set one")
	}

	if len(overrides.IncludedIDs) > 0 {
		includedIds, err := FindDependencies(model, overrides.IncludedIDs)
		if err != nil {
			return err
		}
		skip := func(id string) bool { return !includedIds[id] }
		skipField := func(id string) bool { return false }
		if err := skipModelElementsImpl(model, skip, skipField); err != nil {
			return err
		}
	}

	if len(overrides.SkippedIDs) > 0 {
		skippedIDs := map[string]bool{}
		for _, id := range overrides.SkippedIDs {
			skippedIDs[id] = true
		}
		skip := func(id string) bool { return skippedIDs[id] }
		skipField := func(id string) bool { return skippedIDs[id] }
		if err := skipModelElementsImpl(model, skip, skipField); err != nil {
			return err
		}
	}
	return nil
}

func skipModelElementsImpl(model *API, skip func(id string) bool, skipField func(id string) bool) error {
	for _, m := range model.Messages {
		skipMessageElements(m, skip, skipField)
	}
	model.Enums = slices.DeleteFunc(model.Enums, func(x *Enum) bool { return skip(x.id) })
	model.Messages = slices.DeleteFunc(model.Messages, func(x *Message) bool { return skip(x.id) })
	model.Services = slices.DeleteFunc(model.Services, func(x *Service) bool { return skip(x.ID) })

	for service := range model.AllServices() {
		service.Methods = slices.DeleteFunc(service.Methods, func(x *Method) bool { return skip(x.id) })
		for _, method := range service.Methods {
			if method.pagination != nil && skipField(method.pagination.id) {
				return fmt.Errorf("unsupported: skipping field %s which enables pagination", method.pagination.id)
			}
			idx := slices.IndexFunc(method.signatures, func(s *MethodSignature) bool {
				return slices.ContainsFunc(s.Fields(), func(f *Field) bool { return skipField(f.id) })
			})
			if idx != -1 {
				return fmt.Errorf("unsupported: skipping field that is in one of the method signatures for method: %s", method.id)
			}
		}
	}
	if model.QuickstartService != nil && !slices.Contains(model.Services, model.QuickstartService) {
		model.QuickstartService = findQuickstartService(model)
	}
	return nil
}

func skipMessageElements(message *Message, skip func(id string) bool, skipField func(id string) bool) {
	for _, m := range message.messages {
		skipMessageElements(m, skip, skipField)
	}
	message.messages = slices.DeleteFunc(message.messages, func(x *Message) bool { return skip(x.id) })
	message.enums = slices.DeleteFunc(message.enums, func(x *Enum) bool { return skip(x.id) })
	previous := slices.Clone(message.fields)
	skipped := func(x *Field) bool {
		if skipField(x.id) {
			return true
		}
		if x.group != nil && skipField(x.group.id) {
			return true
		}
		return false
	}
	message.fields = slices.DeleteFunc(message.fields, skipped)
	message.skippedFields = slices.DeleteFunc(previous, func(x *Field) bool { return !skipped(x) })
	for _, oneof := range message.oneOfs {
		oneof.fields = slices.DeleteFunc(oneof.fields, func(x *Field) bool {
			return skipField(x.id)
		})
		if len(oneof.fields) > 0 {
			oneof.exampleField = slices.MaxFunc(oneof.fields, sortOneOfFieldForExamples)
		}
	}
	message.oneOfs = slices.DeleteFunc(message.oneOfs, func(x *OneOf) bool {
		return skipField(x.id) || len(x.fields) == 0
	})
}
