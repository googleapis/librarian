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
	"log"
	"testing"

	"github.com/google/go-cmp/cmp"
)

// TODO(https://github.com/googleapis/librarian/issues/202): add better tests
// for librarian.Run.
func TestRun(t *testing.T) {
	if err := Run(t.Context(), []string{"version"}...); err != nil {
		log.Fatal(err)
	}
}

func TestIsURL(t *testing.T) {
	for _, test := range []struct {
		name  string
		input string
		want  bool
	}{
		{
			name:  "Valid HTTPS URL",
			input: "https://github.com/googleapis/librarian-go",
			want:  true,
		},
		{
			name:  "Valid HTTP URL",
			input: "http://example.com/path?query=value",
			want:  true,
		},
		{
			name:  "Valid FTP URL",
			input: "ftp://user:password@host/path",
			want:  true,
		},
		{
			name:  "URL without scheme",
			input: "google.com",
			want:  false,
		},
		{
			name:  "URL with scheme only",
			input: "https://",
			want:  false,
		},
		{
			name:  "Absolute Unix file path",
			input: "/home/user/file",
			want:  false,
		},
		{
			name:  "Relative file path",
			input: "home/user/file",
			want:  false,
		},
		{
			name:  "Empty string",
			input: "",
			want:  false,
		},
		{
			name:  "Plain string",
			input: "just-a-string",
			want:  false,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := isURL(test.input)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("isURL() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
