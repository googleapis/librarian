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

// IdempotencyOverride describes overrides for idempotency config of a method.
type IdempotencyOverride struct {
	// The method ID.
	ID string
	// Whether the method should be considered idempotent.
	IsIdempotent bool
}

// UpdateMethodIdempotency updates method idempotency override based on configuration.
func UpdateMethodIdempotency(overrides []IdempotencyOverride, a *API) {
	overrideByID := make(map[string]bool, len(overrides))
	for _, o := range overrides {
		overrideByID[o.ID] = o.IsIdempotent
	}
	for m := range a.AllMethods() {
		if isIdempotent, ok := overrideByID[m.ID]; ok {
			isIdempotentCopy := isIdempotent
			m.IdempotencyOverride = &isIdempotentCopy
		}
	}
}
