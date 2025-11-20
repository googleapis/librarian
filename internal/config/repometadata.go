// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package config

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
