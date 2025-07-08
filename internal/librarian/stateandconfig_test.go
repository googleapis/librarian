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

	"github.com/go-playground/validator/v10"
	"github.com/google/go-cmp/cmp"
)

func TestParseLibrarianState(t *testing.T) {
	for _, test := range []struct {
		name    string
		content string
		want    *LibrarianState
		wantErr bool
	}{
		{
			name:    "valid state",
			content: "image: gcr.io/test/image:v1.2.3\nlibraries:\n  - id: a/b\n    source_paths:\n      - src/a\n      - src/b\n    apis:\n      - path: a/b/v1\n        service_config: a/b/v1/service.yaml\n",
			want: &LibrarianState{
				Image: "gcr.io/test/image:v1.2.3",
				Libraries: []Library{
					{
						Id:          "a/b",
						SourcePaths: []string{"src/a", "src/b"},
						APIs: []API{
							{
								Path:          "a/b/v1",
								ServiceConfig: "a/b/v1/service.yaml",
							},
						},
					},
				},
			},
		},
		{
			name:    "invalid yaml",
			content: "image: gcr.io/test/image:v1.2.3\n  libraries: []\n",
			wantErr: true,
		},
		{
			name:    "validation error",
			content: "image: gcr.io/test/image:v1.2.3\nlibraries: []\n",
			wantErr: true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			contentLoader := func() ([]byte, error) {
				return []byte(test.content), nil
			}
			got, err := parseLibrarianState(contentLoader)
			if (err != nil) != test.wantErr {
				t.Errorf("parseLibrarianState() error = %v, wantErr %v", err, test.wantErr)
				return
			}
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("parseLibrarianState() mismatch (-want +got): %s", diff)
			}
		})
	}
}

func TestValidators(t *testing.T) {
	for _, test := range []struct {
		name       string
		validation string
		value      string
		valid      bool
	}{
		// is-regexp
		{"regexp valid", "is-regexp", `.*`, true},
		{"regexp invalid", "is-regexp", `(`, false},

		// is-dirpath
		{"dirpath valid", "is-dirpath", "a/b/c", true},
		{"dirpath valid with dots", "is-dirpath", "a/./b/../c", true},
		{"dirpath empty", "is-dirpath", "", false},
		{"dirpath absolute", "is-dirpath", "/a/b", false},
		{"dirpath up traversal", "is-dirpath", "../a", false},
		{"dirpath double dot", "is-dirpath", "..", false},
		{"dirpath single dot", "is-dirpath", ".", false},
		{"dirpath invalid chars", "is-dirpath", "a/b<c", false},

		// is-image
		{"image valid", "is-image", "gcr.io/google/go-container", true},
		{"image invalid", "is-image", "gcr.io/google/go-container with spaces", false},

		// is-library-id
		{"library-id valid", "is-library-id", "a/b-c.d_e", true},
		{"library-id empty", "is-library-id", "", false},
		{"library-id dot", "is-library-id", ".", false},
		{"library-id double dot", "is-library-id", "..", false},
		{"library-id invalid chars", "is-library-id", "a/b?c", false},
	} {
		t.Run(test.name, func(t *testing.T) {
			validate := validator.New()
			if err := validate.RegisterValidation("is-regexp", validateRegexp); err != nil {
				t.Fatalf("failed to register validation: %v", err)
			}
			if err := validate.RegisterValidation("is-dirpath", validateDirPath); err != nil {
				t.Fatalf("failed to register validation: %v", err)
			}
			if err := validate.RegisterValidation("is-image", validateImage); err != nil {
				t.Fatalf("failed to register validation: %v", err)
			}
			if err := validate.RegisterValidation("is-library-id", validateLibraryID); err != nil {
				t.Fatalf("failed to register validation: %v", err)
			}
			err := validate.Var(test.value, test.validation)
			if (err == nil) != test.valid {
				t.Errorf("validation %q on value %q valid = %v, want %v", test.validation, test.value, err == nil, test.valid)
			}
		})
	}
}
