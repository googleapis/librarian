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

// Package docuploader provides functionality for packaging documentation for
// googleapis.dev and cloud.google.com.
package docuploader

// DocUploaderMetadata represents the structure of the docs.metadata.json file
// embedded in documentation tarballs.
// See https://github.com/googleapis/docuploader/blob/main/docuploader/protos/metadata.proto
// for original .proto file. Field comments are loosely-edited versions of
// comments on those fields.
type DocUploaderMetadata struct {
	// UpdateTime is the timestamp when the documentation was built, in
	// an https://www.ietf.org/rfc/rfc3339.txt format (in UTC).
	UpdateTime string `json:"updateTime,omitempty"`

	// Name is the product/API name. This *should* match the DNS name of the API
	// service. For example, Python's Cloud Storage client would list "storage"
	// because it talks to storage.googleapis.com. For non-API libraries, such
	// as auth, consult the docs team for guidance.
	Name string `json:"name,omitempty"`

	// Version is the package version, for example 1.2.3, or 0.1.2-beta1.
	Version string `json:"version,omitempty"`

	// Language is the (programming) language. While this isn't enforced with an
	// enum, you should use one of the following strings:  python, nodejs, ruby,
	// dotnet, java, go, php, cpp. In some cases you might need to use something
	// else, consult the docs team for guidance.
	Language string `json:"language,omitempty"`

	// DistributionName is the language-idiomatic package/distribution name, for
	// example:
	// Python: google-cloud-storage
	// Node.js: @google/cloud-storage
	DistributionName string `json:"distributionName,omitempty"`

	// ProductPage is the URI of the product page. For example
	// https://cloud.google.com/storage/
	ProductPage string `json:"productPage,omitempty"`

	// GithubRepository is the URI of the github repository. For example, Node's
	// Cloud Storage client would be
	// https://github.com/googleapis/nodejs-storage
	GithubRepository string `json:"githubRepository,omitempty"`

	// IssueTracker is the URI of the issue tracker. For example the Node
	// Storage client's issue tracker is
	// https://github.com/googleapis/nodejs-storage/issues
	IssueTracker string `json:"issueTracker,omitempty"`

	// Stem is the published stem for this docset. If unspecified, it will be
	// set to `{language}/{name}`. This is an advanced field, and should only be
	// set after consultation with the docs team.
	Stem string `json:"stem,omitempty"`

	// ServingPath is the serving path for this docset. If unspecified, it will
	// be set to `{stem}/{version}`. This is an advanced field, and should only
	// be set after consultation with the docs team.
	ServingPath string `json:"servingPath,omitempty"`

	// XRefs specifies the DocFX xref URLs required for these docs. This is an
	// advanced field, and should only be set after consultation with the docs
	// team.
	XRefs []string `json:"xrefs,omitempty"`

	// XRefServices specifies the DocFX xref service URLs required for these
	// docs. This is an advanced field, and should only be set after
	// consultation with the docs team.
	XRefServices []string `json:"xrefServices,omitempty"`
}
