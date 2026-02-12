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
		skipModelElementsImpl(model, skip)
	}

	if len(overrides.SkippedIDs) > 0 {
		skippedIDs := map[string]bool{}
		for _, id := range overrides.SkippedIDs {
			skippedIDs[id] = true
		}
		skip := func(id string) bool { return skippedIDs[id] }
		skipModelElementsImpl(model, skip)
	}
	return nil
}

func skipModelElementsImpl(model *API, skip func(id string) bool) {
	for _, m := range model.Messages {
		skipMessageElements(m, skip)
	}
	model.Enums = slices.DeleteFunc(model.Enums, func(x *Enum) bool { return skip(x.ID) })
	model.Messages = slices.DeleteFunc(model.Messages, func(x *Message) bool { return skip(x.ID) })
	model.Services = slices.DeleteFunc(model.Services, func(x *Service) bool { return skip(x.ID) })
	for _, service := range model.State.ServiceByID {
		service.Methods = slices.DeleteFunc(service.Methods, func(x *Method) bool { return skip(x.ID) })
	}
}

func skipMessageElements(message *Message, skip func(id string) bool) {
	for _, m := range message.Messages {
		skipMessageElements(m, skip)
	}
	message.Messages = slices.DeleteFunc(message.Messages, func(x *Message) bool { return skip(x.ID) })
	message.Enums = slices.DeleteFunc(message.Enums, func(x *Enum) bool { return skip(x.ID) })
}
