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

// eligibleServices defines the service IDs (Protobuf package names) that are eligible for the gated heuristic.
var eligibleServices = map[string]bool{
	".google.cloud.compute":  true,
	".google.cloud.sql":      true,
	".google.cloud.bigquery": true,
}

// IsHeuristicEligible returns true if the given service ID is in the allowlist for the resource name heuristic.
func IsHeuristicEligible(serviceID string) bool {
	parts := strings.Split(serviceID, ".")
	if len(parts) >= 4 {
		// Extract the service prefix (e.g., ".google.cloud.compute.v1")
		prefix := strings.Join(parts[:4], ".")
		return eligibleServices[prefix]
	}
	return false
}

// baseVocabulary is the set of literal segments that are common
// across Google Cloud APIs and should always be considered valid tokens.
var baseVocabulary = map[string]bool{
	"projects":        true,
	"locations":       true,
	"folders":         true,
	"organizations":   true,
	"billingAccounts": true,
}

// isBaseVocabulary checks if a segment is part of the common resource vocabulary.
func isBaseVocabulary(segment string) bool {
	return baseVocabulary[segment]
}

// isCollectionIdentifier returns true if the segment is likely a collection ID.
// This includes strict base vocabulary matches OR heuristic plural nouns (ending in 's').
// Single letter segments (like 'v1' or param 's') are not considered collections.
func isCollectionIdentifier(segment string) bool {
	if isBaseVocabulary(segment) {
		return true
	}
	return len(segment) > 1 && strings.HasSuffix(segment, "s")
}
