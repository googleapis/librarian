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

// ResolveAutoPopulatedFields resolves and attaches auto-populated fields to
// methods in the API model. `methodFields` maps a method ID (e.g.
// ".google.example.v1.Service.Method") to the configured auto-populated field
// names. Fields that are not marked as `AutoPopulated` on the input message are
// ignored.
func (a *API) ResolveAutoPopulatedFields(methodFields map[string][]string) {
	for id, fieldNames := range methodFields {
		method := a.Method(id)
		if method == nil {
			continue
		}
		message := a.Message(method.inputTypeID)
		if message == nil {
			continue
		}
		for _, name := range fieldNames {
			for _, field := range message.Fields() {
				if field.Name() != name {
					continue
				}
				if field.AutoPopulated() {
					method.autoPopulated = append(method.autoPopulated, field)
				}
				break
			}
		}
	}
}
