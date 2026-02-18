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

// IdentifyTargetResources populates the TargetResource field in PathBinding
// for all methods in the API.
//
// This is done in two passes:
//  1. Explicit Identification: Matches google.api.resource_reference annotations
//     with fields present in the PathTemplate.
//  2. Heuristic Identification: For allow-listed services, uses path segment
//     patterns to identify resources when annotations are missing.
func IdentifyTargetResources(model *API) {
	for _, service := range model.Services {
		for _, method := range service.Methods {
			if method.PathInfo == nil {
				continue
			}
			for _, binding := range method.PathInfo.Bindings {
				if binding.PathTemplate == nil {
					continue
				}

				// Priority 1: Explicit Identification
				// Matches google.api.resource_reference annotations.
				if target := identifyExplicitTarget(method, binding); target != nil {
					binding.TargetResource = target
					continue
				}

				// Priority 2: Heuristic Identification
				// Uses path segment patterns to guess the resource.
				// TODO(#4100): Implement IdentifyTargetResources for allow-listed services using heuristic path segment patterns.
			}
		}
	}
}

func identifyExplicitTarget(_ *Method, _ *PathBinding) *TargetResource {
	// TODO(#4099): Implement IdentifyTargetResources for consistent explicit resource identification in the sidekick parser.
	return nil
}
