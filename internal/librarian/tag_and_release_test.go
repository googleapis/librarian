// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package librarian

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
)

func TestNewTagAndReleaseRunner(t *testing.T) {
	testcases := []struct {
		name    string
		cfg     *config.Config
		wantErr bool
	}{
		{
			name: "valid config",
			cfg: &config.Config{
				GitHubToken: "some-token",
			},
			wantErr: false,
		},
		{
			name:    "missing github token",
			cfg:     &config.Config{},
			wantErr: true,
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			r, err := newTagAndReleaseRunner(tc.cfg)
			if (err != nil) != tc.wantErr {
				t.Errorf("newTagAndReleaseRunner() error = %v, wantErr %v", err, tc.wantErr)
				return
			}
			if !tc.wantErr && r == nil {
				t.Errorf("newTagAndReleaseRunner() got nil runner, want non-nil")
			}
		})
	}
}

func TestParsePullRequestBody(t *testing.T) {
	tests := []struct {
		name string
		body string
		want []libraryRelease
	}{
		{
			name: "single library",
			body: `
Librarian Version: v0.2.0
Language Image: image

<details><summary>google-cloud-storage: 1.2.3</summary>

[1.2.3](https://github.com/googleapis/google-cloud-go/compare/google-cloud-storage-v1.2.2...google-cloud-storage-v1.2.3) (2025-08-15)

### Features

* Add new feature ([abcdef1](https://github.com/googleapis/google-cloud-go/commit/abcdef1))

</details>`,
			want: []libraryRelease{
				{
					Version: "1.2.3",
					Library: "google-cloud-storage",
					Body: `[1.2.3](https://github.com/googleapis/google-cloud-go/compare/google-cloud-storage-v1.2.2...google-cloud-storage-v1.2.3) (2025-08-15)

### Features

* Add new feature ([abcdef1](https://github.com/googleapis/google-cloud-go/commit/abcdef1))`,
				},
			},
		},
		{
			name: "multiple libraries",
			body: `
Librarian Version: 1.2.3
Language Image: gcr.io/test/image:latest

<details><summary>library-one: 1.0.0</summary>

[1.0.0](https://github.com/googleapis/repo/compare/library-one-v0.9.0...library-one-v1.0.0) (2025-08-15)

### Features

* some feature ([1234567](https://github.com/googleapis/repo/commit/1234567))

</details>

<details><summary>another-library-name: 2.3.4</summary>

[2.3.4](https://github.com/googleapis/repo/compare/another-library-name-v2.3.3...another-library-name-v2.3.4) (2025-08-15)

### Bug Fixes

* some bug fix ([abcdefg](https://github.com/googleapis/repo/commit/abcdefg))

</details>`,
			want: []libraryRelease{
				{
					Version: "1.0.0",
					Library: "library-one",
					Body: `[1.0.0](https://github.com/googleapis/repo/compare/library-one-v0.9.0...library-one-v1.0.0) (2025-08-15)

### Features

* some feature ([1234567](https://github.com/googleapis/repo/commit/1234567))`,
				},
				{
					Version: "2.3.4",
					Library: "another-library-name",
					Body: `[2.3.4](https://github.com/googleapis/repo/compare/another-library-name-v2.3.3...another-library-name-v2.3.4) (2025-08-15)

### Bug Fixes

* some bug fix ([abcdefg](https://github.com/googleapis/repo/commit/abcdefg))`,
				},
			},
		},
		{
			name: "empty body",
			body: "",
			want: nil,
		},
		{
			name: "malformed summary",
			body: `
Librarian Version: 1.2.3
Language Image: gcr.io/test/image:latest

<details><summary>no-version-here</summary>

some content

</details>`,
			want: nil,
		},
		{
			name: "v prefix in version",
			body: `
<details><summary>google-cloud-storage: v1.2.3</summary>

[v1.2.3](https://github.com/googleapis/google-cloud-go/compare/google-cloud-storage-v1.2.2...google-cloud-storage-v1.2.3) (2025-08-15)

</details>`,
			want: []libraryRelease{
				{
					Version: "v1.2.3",
					Library: "google-cloud-storage",
					Body:    "[v1.2.3](https://github.com/googleapis/google-cloud-go/compare/google-cloud-storage-v1.2.2...google-cloud-storage-v1.2.3) (2025-08-15)",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parsePullRequestBody(tt.body)
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("ParsePullRequestBody() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
