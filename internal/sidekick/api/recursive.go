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

// LabelRecursiveFields labels fields that recursively reference other messages.
func LabelRecursiveFields(model *API) {
	for message := range model.AllMessages() {
		for _, field := range message.Fields {
			visited := map[string]bool{message.ID: true}
			field.recursive = field.recursivelyReferences(message.ID, model, visited)
		}
	}
}

func (field *Field) recursivelyReferences(messageID string, model *API, visited map[string]bool) bool {
	if field.typez == TypezMap {
		mapType := field.mapType
		if mapType == nil {
			mapType = model.Map(field.typezID)
		}
		if mapType == nil || visited[mapType.ID] {
			return false
		}
		visited[mapType.ID] = true
		defer delete(visited, mapType.ID)
		recursive := false
		if mapType.Key != nil && mapType.Key.recursivelyReferences(messageID, model, visited) {
			mapType.Key.recursive = true
			recursive = true
		}
		if mapType.Value != nil && mapType.Value.recursivelyReferences(messageID, model, visited) {
			mapType.Value.recursive = true
			recursive = true
		}
		if recursive {
			field.recursive = true
		}
		return recursive
	}
	if field.typez != TypezMessage {
		return false
	}
	if field.typezID == messageID {
		return true
	}
	if _, ok := visited[field.typezID]; ok {
		return false
	}
	if fieldMessage := model.Message(field.typezID); fieldMessage != nil {
		return fieldMessage.recursivelyReferences(messageID, model, visited)
	}
	return false
}

func (message *Message) recursivelyReferences(messageID string, model *API, visited map[string]bool) bool {
	visited[message.ID] = true
	for _, field := range message.Fields {
		if field.recursivelyReferences(messageID, model, visited) {
			return true
		}
	}
	return false
}
