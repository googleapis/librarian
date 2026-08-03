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

package php

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestNamespace(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name    string
		content string
		want    string
	}{
		{
			name:    "option without parentheses",
			content: `option php_namespace = "Google\\Cloud\\SecretManager\\V1";`,
			want:    `Google\Cloud\SecretManager\V1`,
		},
		{
			name:    "extra whitespace",
			content: `option php_namespace   =   "Google\\Cloud\\Storage";`,
			want:    `Google\Cloud\Storage`,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			tmpDir := t.TempDir()
			file := filepath.Join(tmpDir, test.name+".proto")
			if err := os.WriteFile(file, []byte(test.content), 0o644); err != nil {
				t.Fatal(err)
			}
			got, err := namespace(file)
			if err != nil {
				t.Fatalf("namespace() failed: %v", err)
			}
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
