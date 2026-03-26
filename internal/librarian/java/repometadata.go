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
	"fmt"
	"strings"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/repometadata"
	"github.com/googleapis/librarian/internal/serviceconfig"
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
	return repometadata.WriteJSON(metadata, "  ", libraryOutputDir, ".repo-metadata.json")
}

// deriveRepoMetadata constructs the repoMetadata for a Java library using
// information from the primary service configuration and library-level overrides.
func deriveRepoMetadata(cfg *config.Config, library *config.Library, googleapisDir string) (*repoMetadata, error) {
	serviceconfig.SortAPIs(library.APIs)
	sharedMetadata, err := repometadata.FromLibrary(cfg, library, googleapisDir)
	if err != nil {
		return nil, err
	}

	metadata := &repoMetadata{
		APIShortname:         sharedMetadata.APIShortname,
		NamePretty:           sharedMetadata.NamePretty,
		ProductDocumentation: sharedMetadata.ProductDocumentation,
		APIDescription:       sharedMetadata.APIDescription,
		ReleaseLevel:         sharedMetadata.ReleaseLevel,
		Language:             config.LanguageJava,
		Repo:                 sharedMetadata.Repo,
		RepoShort:            fmt.Sprintf("%s-%s", config.LanguageJava, library.Name),
		DistributionName:     sharedMetadata.DistributionName,
		APIID:                sharedMetadata.APIID,
		LibraryType:          repometadata.GAPICAutoLibraryType,
		RequiresBilling:      true,
	}

	// Java-specific overrides and optional fields
	if library.Java != nil {
		if library.Java.APIIDOverride != "" {
			metadata.APIID = library.Java.APIIDOverride
		}
		if library.Java.APIDescriptionOverride != "" {
			metadata.APIDescription = library.Java.APIDescriptionOverride
		}
		if library.Java.DistributionNameOverride != "" {
			metadata.DistributionName = library.Java.DistributionNameOverride
		}
		if library.Java.IssueTrackerOverride != "" {
			metadata.IssueTracker = library.Java.IssueTrackerOverride
		}
		if library.Java.LibraryTypeOverride != "" {
			metadata.LibraryType = library.Java.LibraryTypeOverride
		}
		if library.Java.NamePrettyOverride != "" {
			metadata.NamePretty = library.Java.NamePrettyOverride
		}
		if library.Java.ProductDocumentationOverride != "" {
			metadata.ProductDocumentation = library.Java.ProductDocumentationOverride
		}
		if library.Java.ClientDocumentationOverride != "" {
			metadata.ClientDocumentation = library.Java.ClientDocumentationOverride
		}
		metadata.RequiresBilling = !library.Java.BillingNotRequired
		// Java only fields
		metadata.CodeownerTeam = library.Java.CodeownerTeam
		metadata.ExtraVersionedModules = library.Java.ExtraVersionedModules
		metadata.ExcludedDependencies = library.Java.ExcludedDependencies
		metadata.ExcludedPoms = library.Java.ExcludedPoms
		metadata.MinJavaVersion = library.Java.MinJavaVersion
		metadata.RecommendedPackage = library.Java.RecommendedPackage
		metadata.RestDocumentation = library.Java.RestDocumentation
		metadata.RpcDocumentation = library.Java.RpcDocumentation
	}

	// distribution_name default for Java is groupId:artifactId
	if !strings.Contains(metadata.DistributionName, ":") {
		metadata.DistributionName = deriveDistributionName(library)
	}
	// Default ClientDocumentation uses artifact ID
	if metadata.ClientDocumentation == "" {
		parts := strings.Split(metadata.DistributionName, ":")
		artifactID := parts[len(parts)-1]
		metadata.ClientDocumentation = fmt.Sprintf("https://cloud.google.com/java/docs/reference/%s/latest/overview", artifactID)
	}
	// transport
	apiCfg, err := serviceconfig.Find(googleapisDir, library.APIs[0].Path, config.LanguageJava)
	if err != nil {
		return nil, fmt.Errorf("failed to find api config: %w", err)
	}
	metadata.Transport = "both"
	if apiCfg != nil {
		metadata.Transport = apiCfg.RepoMetadataTransport(config.LanguageJava)
	}
	return metadata, nil
}
