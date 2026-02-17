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

// TargetList defines the service IDs (Protobuf package names) that are eligible for the gated heuristic.
var TargetList = []string{
	".google.cloud.compute",
	".google.cloud.sql",
	".google.cloud.bigquery",
}

// IsHeuristicEligible returns true if the given service ID is in the allowlist for the resource name heuristic.
func IsHeuristicEligible(serviceID string) bool {
	for _, target := range TargetList {
		if strings.HasPrefix(serviceID, target) {
			return true
		}
	}
	return false
}

// baseVocabulary is the set of literal segments that are universal
// across Google Cloud APIs and should always be considered valid tokens.
var baseVocabulary = map[string]bool{
	"projects":        true,
	"locations":       true,
	"folders":         true,
	"organizations":   true,
	"billingAccounts": true,
}

// BaseVocabulary returns the set of literal segments that are universal
// across Google Cloud APIs and should always be considered valid tokens.
// The returned map is read-only and should not be modified.
func BaseVocabulary() map[string]bool {
	return baseVocabulary
}
