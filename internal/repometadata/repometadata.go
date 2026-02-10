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

// Package repometadata represents the data in .repo-metadata.json files,
// and the ability to create those files from other Librarian configuration.
package repometadata

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/serviceconfig"
)

var (
	errNoAPIs          = errors.New("library has no APIs from which to get metadata")
	errNoServiceConfig = errors.New("library has no service config from which to get metadata")
)

type libraryInfo struct {
	descriptionOverride string
	name                string
	releaseLevel        string
}

// RepoMetadata represents the .repo-metadata.json file structure.
type RepoMetadata struct {
	// APIDescription is the description of the API.
	APIDescription string `json:"api_description,omitempty"`

	// APIID is the fully qualified API ID (e.g., "secretmanager.googleapis.com").
	APIID string `json:"api_id,omitempty"`

	// APIShortname is the short name of the API.
	APIShortname string `json:"api_shortname,omitempty"`

	// ClientDocumentation is the URL to the client library documentation.
	ClientDocumentation string `json:"client_documentation,omitempty"`

	// DefaultVersion is the default API version (e.g., "v1", "v1beta1").
	DefaultVersion string `json:"default_version,omitempty"`

	// DistributionName is the name of the library distribution package.
	DistributionName string `json:"distribution_name,omitempty"`

	// IssueTracker is the URL to the issue tracker.
	IssueTracker string `json:"issue_tracker"`

	// Language is the programming language (e.g., "rust", "python", "go").
	Language string `json:"language,omitempty"`

	// LibraryType is the type of library (e.g., "GAPIC_AUTO").
	LibraryType string `json:"library_type,omitempty"`

	// Name is the API short name.
	Name string `json:"name,omitempty"`

	// NamePretty is the human-readable name of the API.
	NamePretty string `json:"name_pretty,omitempty"`

	// ProductDocumentation is the URL to the product documentation.
	ProductDocumentation string `json:"product_documentation,omitempty"`

	// ReleaseLevel is the release level (e.g., "stable", "preview").
	ReleaseLevel string `json:"release_level,omitempty"`

	// Repo is the repository name (e.g., "googleapis/google-cloud-rust").
	Repo string `json:"repo,omitempty"`
}

// FromLibrary generates the .repo-metadata.json file from config.Library.
// It retrieves API information from the provided googleapisDir and constructs the metadata.
func FromLibrary(library *config.Library, language, repo, googleapisDir, defaultVersion, outdir string) error {
	// TODO(https://github.com/googleapis/librarian/issues/3146):
	// Compute the default version, potentially with an override, instead of
	// taking it as a parameter.
	if len(library.APIs) == 0 {
		return fmt.Errorf("failed to generate metadata for %s: %w", library.Name, errNoAPIs)
	}
	firstAPIPath := library.APIs[0].Path
	api, err := serviceconfig.Find(googleapisDir, firstAPIPath, language)
	if err != nil {
		return fmt.Errorf("failed to find API for path %s: %w", firstAPIPath, err)
	}
	if api.ServiceConfig == "" {
		return fmt.Errorf("failed to generate metadata for %s: %w", library.Name, errNoServiceConfig)
	}
	info := &libraryInfo{
		descriptionOverride: library.DescriptionOverride,
		name:                library.Name,
		releaseLevel:        library.ReleaseLevel,
	}
	return FromAPI(api, info, language, repo, defaultVersion, outdir)
}

// FromAPI generates the .repo-metadata.json file from a serviceconfig.API and additional library information.
func FromAPI(api *serviceconfig.API, info *libraryInfo, language, repo, defaultVersion, outputDir string) error {
	clientDocURL := buildClientDocURL(language, extractNameFromAPIID(api.ServiceName))
	metadata := &RepoMetadata{
		APIID:               api.ServiceName,
		NamePretty:          cleanTitle(api.Title),
		DefaultVersion:      defaultVersion,
		ClientDocumentation: clientDocURL,
		ReleaseLevel:        info.releaseLevel,
		Language:            language,
		LibraryType:         "GAPIC_AUTO",
		Repo:                repo,
		DistributionName:    info.name,
	}

	metadata.ProductDocumentation = extractBaseProductURL(api.DocumentationURI)
	metadata.IssueTracker = api.NewIssueURI
	metadata.APIShortname = api.ShortName
	metadata.Name = api.ShortName
	metadata.APIDescription = api.Description
	if info.descriptionOverride != "" {
		metadata.APIDescription = info.descriptionOverride
	}

	data, err := json.MarshalIndent(metadata, "", "    ")
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	metadataPath := filepath.Join(outputDir, ".repo-metadata.json")
	if err := os.WriteFile(metadataPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write metadata file: %w", err)
	}

	return nil
}

// buildClientDocURL builds the client documentation URL based on language.
func buildClientDocURL(language, serviceName string) string {
	switch language {
	case "python":
		return fmt.Sprintf("https://cloud.google.com/python/docs/reference/%s/latest", serviceName)
	case "rust":
		return fmt.Sprintf("https://docs.rs/google-cloud-%s/latest", serviceName)
	default:
		return ""
	}
}

// extractBaseProductURL extracts the base product URL from a documentation URI.
// Example: "https://cloud.google.com/secret-manager/docs/overview" -> "https://cloud.google.com/secret-manager/"
func extractBaseProductURL(docURI string) string {
	// Strip off /docs/* suffix to get base product URL
	if base, _, found := strings.Cut(docURI, "/docs/"); found {
		return base + "/"
	}
	// If no /docs/ found, return as-is
	return docURI
}

// cleanTitle removes "API" suffix from title to get name_pretty.
// Example: "Secret Manager API" -> "Secret Manager".
func cleanTitle(title string) string {
	title = strings.TrimSpace(title)
	title = strings.TrimSuffix(title, " API")
	return strings.TrimSpace(title)
}

// extractNameFromAPIID extracts the service name from the API ID.
// Example: "secretmanager.googleapis.com" -> "secretmanager".
func extractNameFromAPIID(apiID string) string {
	name, _, _ := strings.Cut(apiID, ".")
	return name
}
