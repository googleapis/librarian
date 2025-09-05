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

package language

import (
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestWalkDir(t *testing.T) {
	// It should get the `*.md.mustache` files and skip `partial.mustache`
	got := WalkTemplatesDir(templates, "testTemplates")
	want := []GeneratedFile{
		{
			TemplatePath: "testTemplates/README.md.mustache",
			OutputPath:   filepath.FromSlash("/README.md"),
		},
		{
			TemplatePath: "testTemplates/test001.txt.mustache",
			OutputPath:   filepath.FromSlash("/test001.txt"),
		},
	}
	if diff := cmp.Diff(want, got); len(diff) != 0 {
		t.Errorf("mismatched config from LoadConfig (-want, +got):\n%s", diff)
	}
}
