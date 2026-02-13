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

package docuploader

import (
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/repometadata"
)

func TestFromRepoMetadata(t *testing.T) {
	for _, test := range []struct {
		name  string
		input *repometadata.RepoMetadata
		want  *DocUploaderMetadata
	}{
		{
			name: "fully populated repo metadata",
			input: &repometadata.RepoMetadata{
				APIDescription:       "api-description-ignored",
				APIID:                "api-id-ignored",
				APIShortname:         "api-shortname-ignored",
				ClientDocumentation:  "client-documentation-ignored",
				DefaultVersion:       "default-version-ignored",
				DistributionName:     "distribution-name",
				IssueTracker:         "issue-tracker",
				Language:             "language",
				LibraryType:          "library-type-ignored",
				Name:                 "name",
				NamePretty:           "name-pretty-ignored",
				ProductDocumentation: "product-documentation",
				ReleaseLevel:         "release-level-ignored",
				Repo:                 "repo",
			},
			want: &DocUploaderMetadata{
				DistributionName: "distribution-name",
				GithubRepository: "repo",
				IssueTracker:     "issue-tracker",
				Language:         "language",
				Name:             "name",
				ProductPage:      "product-documentation",
			},
		},
		{
			name:  "empty repo metadata",
			input: &repometadata.RepoMetadata{},
			want:  &DocUploaderMetadata{},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := FromRepoMetadata(test.input)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestSetUpdateTime(t *testing.T) {
	metadata := &DocUploaderMetadata{}
	location := time.FixedZone("UTC+1", 3600)
	updateTime := time.Date(2026, 2, 13, 16, 49, 50, 123456789, location)
	metadata.SetUpdateTime(updateTime)
	want := "2026-02-13T15:49:50Z"
	if diff := cmp.Diff(want, metadata.UpdateTime); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}
