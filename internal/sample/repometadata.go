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

package sample

import (
	"github.com/googleapis/librarian/internal/repometadata"
)

// RepoMetadata returns a sample [repometadata.RepoMetadata]. The Language
// field is left empty so callers can set it for their specific language.
func RepoMetadata() *repometadata.RepoMetadata {
	return &repometadata.RepoMetadata{
		APIDescription:       APIDescription,
		APIID:                "secretmanager.googleapis.com",
		APIShortname:         APIName,
		DefaultVersion:       "v1",
		IssueTracker:         "https://issuetracker.google.com/issues/new?component=784854&template=1380926",
		LibraryType:          repometadata.GAPICAutoLibraryType,
		Name:                 APIName,
		NamePretty:           "Secret Manager",
		ProductDocumentation: "https://cloud.google.com/secret-manager/",
		ReleaseLevel:         "stable",
		Transport:            "grpc+rest",
	}
}
