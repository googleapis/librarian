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

package legacyimages

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestParseImage(t *testing.T) {
	for _, test := range []struct {
		name    string
		image   string
		want    *containerImage
		wantErr bool
	}{
		{
			name:  "AR unpinned",
			image: "us-central1-docker.pkg.dev/some-project/some-repo/some-image",
			want: &containerImage{
				Name:       "some-image",
				Location:   "us-central1",
				Project:    "some-project",
				Repository: "some-repo",
			},
		},
		{
			name:  "AR pinned SHA",
			image: "us-central1-docker.pkg.dev/some-project/some-repo/some-image@sha256:abcdef",
			want: &containerImage{
				Name:       "some-image",
				Location:   "us-central1",
				Project:    "some-project",
				Repository: "some-repo",
				SHA:        "sha256:abcdef",
			},
		},
		{
			name:  "AR tagged",
			image: "us-central1-docker.pkg.dev/some-project/some-repo/some-image:1.2.3",
			want: &containerImage{
				Name:       "some-image",
				Location:   "us-central1",
				Project:    "some-project",
				Repository: "some-repo",
				Tag:        "1.2.3",
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			got, err := parseImage(test.image)
			if (err != nil) != test.wantErr {
				t.Errorf("parseImage() error = %v, wantErr %v", err, test.wantErr)
				return
			}
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("parseImage() mismatch (-want +got):\n%s", diff)
			}
			str := got.String()
			if diff := cmp.Diff(str, test.image); diff != "" {
				t.Errorf("image.String() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
