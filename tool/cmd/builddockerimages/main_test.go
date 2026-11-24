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

package main

import (
	"os"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

var stagePattern = regexp.MustCompile(`(?m)^FROM\s.*\sAS\s+(\S+)$`)

// TestSupportedLanguages checks that every language image stage in the
// Dockerfile is built. Helper stages, such as php-runtime, have a hyphen in
// their name.
func TestSupportedLanguages(t *testing.T) {
	content, err := os.ReadFile("../../../cmd/librarian/Dockerfile")
	if err != nil {
		t.Fatal(err)
	}
	var want []string
	for _, match := range stagePattern.FindAllStringSubmatch(string(content), -1) {
		stage := match[1]
		if stage == "build" || stage == "base" || strings.Contains(stage, "-") {
			continue
		}
		want = append(want, stage)
	}
	slices.Sort(want)
	got := slices.Sorted(slices.Values(supportedLanguages))
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}
