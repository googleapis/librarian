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

import "strings"

// BuildHeuristicVocabulary builds the vocabulary of valid resource tokens
// based on the last literal before a variable in Get/List methods.
func BuildHeuristicVocabulary(model *API) map[string]bool {
	tokens := make(map[string]bool)

	// Add standard infrastructure tokens
	tokens["projects"] = true
	tokens["locations"] = true
	tokens["folders"] = true
	tokens["organizations"] = true
	tokens["billingAccounts"] = true

	discoveryExactVerbs := map[string]struct{}{
		"get":            {},
		"list":           {},
		"aggregatedlist": {},
		"create":         {},
		"update":         {},
		"delete":         {},
		"patch":          {},
		"insert":         {},
	}
	discoverySuffixes := []string{
		".get", ".list", ".create", ".update", ".delete", ".patch", ".insert",
	}

	crudPrefixes := []string{
		"get", "list", "create", "update", "delete", "patch", "insert",
	}

	for _, service := range model.Services {
		for _, m := range service.Methods {
			nameLower := strings.ToLower(m.Name)

			var isCRUDPrefix bool
			for _, prefix := range crudPrefixes {
				if strings.HasPrefix(nameLower, prefix) {
					isCRUDPrefix = true
					break
				}
			}

			// Discovery APIs (like Compute) use exact lowercase verbs or suffix verb mapping
			// (e.g., "get", "list", "instances.get", "projects.zones.insert")
			_, isDiscoveryExact := discoveryExactVerbs[nameLower]

			var isDiscoverySuffix bool
			for _, suffix := range discoverySuffixes {
				if strings.HasSuffix(nameLower, suffix) {
					isDiscoverySuffix = true
					break
				}
			}

			if !isCRUDPrefix && !isDiscoveryExact && !isDiscoverySuffix {
				continue
			}

			if m.PathInfo == nil || len(m.PathInfo.Bindings) == 0 {
				continue
			}

			// Parse the path template of the primary binding
			tmpl := m.PathInfo.Bindings[0].PathTemplate
			if tmpl == nil {
				continue
			}

			// Iterate backwards.
			for i := len(tmpl.Segments) - 1; i >= 0; i-- {
				seg := tmpl.Segments[i]
				if seg.Variable != nil {
					if i > 0 && tmpl.Segments[i-1].Literal != nil {
						token := *tmpl.Segments[i-1].Literal
						// Do not add API version strings (e.g., v1, v1beta1) to the vocabulary
						if isVersionString(token) {
							continue
						}
						tokens[token] = true
					}
				}
			}
		}
	}
	return tokens
}
