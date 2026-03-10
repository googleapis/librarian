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

package java

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// repoMetadata represents the .repo-metadata.json file structure for Java.
//
// IMPORTANT: The order of fields in this struct matters. It is ordered
// to match the insertion order in hermetic_build (Python dictionary order).
type repoMetadata struct {
	APIShortname         string `json:"api_shortname"`
	NamePretty           string `json:"name_pretty"`
	ProductDocumentation string `json:"product_documentation"`
	APIDescription       string `json:"api_description"`
	ClientDocumentation  string `json:"client_documentation"`
	ReleaseLevel         string `json:"release_level"`
	// Java-specific field.
	Transport string `json:"transport"`
	Language  string `json:"language"`
	Repo      string `json:"repo"`
	// Java-specific field.
	RepoShort        string `json:"repo_short"`
	DistributionName string `json:"distribution_name"`
	APIID            string `json:"api_id,omitempty"`
	LibraryType      string `json:"library_type"`
	// Java-specific field.
	RequiresBilling bool `json:"requires_billing"`

	// Optional fields (appended in this order in Python)
	// Java-specific field.
	CodeownerTeam string `json:"codeowner_team,omitempty"`
	// Java-specific field.
	ExcludedDependencies string `json:"excluded_dependencies,omitempty"`
	// Java-specific field.
	ExcludedPoms string `json:"excluded_poms,omitempty"`
	IssueTracker string `json:"issue_tracker,omitempty"`
	// Java-specific field.
	RestDocumentation string `json:"rest_documentation,omitempty"`
	// Java-specific field.
	RpcDocumentation string `json:"rpc_documentation,omitempty"`
	// Java-specific field.
	ExtraVersionedModules string `json:"extra_versioned_modules,omitempty"`
	// Java-specific field.
	RecommendedPackage string `json:"recommended_package,omitempty"`
	// Java-specific field.
	MinJavaVersion int `json:"min_java_version,omitempty"`
}

// write writes the given repoMetadata into libraryOutputDir/.repo-metadata.json.
func (metadata *repoMetadata) write(libraryOutputDir string) error {
	data, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	metadataPath := filepath.Join(libraryOutputDir, ".repo-metadata.json")
	if err := os.WriteFile(metadataPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write metadata file: %w", err)
	}

	return nil
}
