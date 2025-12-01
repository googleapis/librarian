// Copyright 2024 Google LLC
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

package sidekick

import (
	"context"
	"testing"
)

func TestRefreshAll(t *testing.T) {
	if err := Run(context.Background(), []string{"refresh-all", "-dry-run", "true"}); err != nil {
		t.Fatal(err)
	}
}

func TestIsInPath(t *testing.T) {
	type TestCase struct {
		Dir  string
		Path string
		Want bool
	}
	testCases := []TestCase{
		{"dart", "dart", true},
		{"dart", "dartboard", false},
		{"dart", "dart/v2", true},
		{"dart", "dart/v2/d2/d4", true},
		{"dart", "a/b/c/d/dart/v2/d2/d4", true},
		{"generator", "dart/v2", false},
		{"generator", "generator/", true},
	}
	for _, test := range testCases {
		got := isInPath(test.Dir, test.Path)
		if got != test.Want {
			t.Errorf("got (%v) != want (%v) for %q in %q", got, test.Want, test.Dir, test.Path)
		}
	}
}
