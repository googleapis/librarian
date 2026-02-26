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

	// Legacy Discovery
	"zones":         true,
	"regions":       true,
	"networks":      true,
	"instances":     true,
	"machineTypes":  true,
	"subnetworks":   true,
}

// isBaseVocabulary checks if a segment is part of the common resource vocabulary.
func isBaseVocabulary(segment string) bool {
	return baseVocabulary[segment]
}

// isCollectionIdentifier returns true if the segment is likely a collection ID.
// It checks in the following order:
// 1. Base vocabulary (projects, locations, etc.)
// 2. Known resource plurals (from the API model).
func isCollectionIdentifier(segment string, vocabulary map[string]bool) bool {
	if isBaseVocabulary(segment) {
		return true
	}
	if vocabulary != nil && vocabulary[segment] {
		return true
	}
	// // Fallback: if the segment ends in 's' and stripping it matches the variable name exactly.
	// if len(segment) > 1 && strings.HasSuffix(segment, "s") {
	// 	return true
	// }
	return false
}

// BuildHeuristicVocabulary collects known resource target names from the model.
// It learns from both explicit ResourceDefinitions and by analyzing the
// HTTP path templates of standard methods.
func BuildHeuristicVocabulary(model *API) map[string]bool {
	vocab := make(map[string]bool, len(model.ResourceDefinitions))

	// 1. Collect from explicit definitions (if any)
	for _, res := range model.ResourceDefinitions {
		if res.Plural != "" {
			vocab[res.Plural] = true
		}
	}

	// 2. Discover from standard method paths
	for _, service := range model.Services {
		for _, method := range service.Methods {
			if !method.IsAIPStandard || method.PathInfo == nil {
				continue
			}
			for _, binding := range method.PathInfo.Bindings {
				vocab = extractLiteralsFromPath(binding.PathTemplate, vocab)
			}
		}
	}
	return vocab
}

// extractLiteralsFromPath parses a path template and adds any string literals
// (both top-level and nested inside variables) to the provided vocabulary set.
func extractLiteralsFromPath(tmpl *PathTemplate, vocab map[string]bool) map[string]bool {
	if tmpl == nil {
		return vocab
	}
	for _, seg := range tmpl.Segments {
		// Add regular literals (e.g. /projects/...)
		if seg.Literal != nil && *seg.Literal != "" {
			vocab[*seg.Literal] = true
		}
		// Add literals nested inside variables (e.g. /{name=projects/*/instances/*})
		if seg.Variable != nil {
			for _, subSeg := range seg.Variable.Segments {
				if subSeg != "" && subSeg != SingleSegmentWildcard && subSeg != MultiSegmentWildcard {
					vocab[subSeg] = true
				}
			}
		}
	}
	return vocab
}
