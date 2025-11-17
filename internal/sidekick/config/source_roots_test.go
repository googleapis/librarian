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

package config

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestSourceRoots(t *testing.T) {
	type TestCase struct {
		input map[string]string
		want  []string
	}
	testCases := []TestCase{
		{map[string]string{}, nil},
		{map[string]string{
			"googleapis-root": "foo",
			"other-root":      "bar",
			"ignored":         "baz",
		}, []string{"googleapis-root", "other-root"}},
		{map[string]string{
			"roots":           "googleapis,more",
			"googleapis-root": "foo",
			"other-root":      "bar",
			"more-root":       "bar",
			"ignored":         "baz",
		}, []string{"googleapis-root", "more-root"}},
	}

	for _, c := range testCases {
		got := SourceRoots(c.input)
		less := func(a, b string) bool { return a < b }
		if diff := cmp.Diff(c.want, got, cmpopts.SortSlices(less)); diff != "" {
			t.Errorf("AllSourceRoots mismatch (-want, +got):\n%s", diff)
		}
	}
}

func TestAllSourceRoots(t *testing.T) {
	type TestCase struct {
		input map[string]string
		want  []string
	}
	testCases := []TestCase{
		{map[string]string{}, nil},
		{map[string]string{
			"googleapis-root": "foo",
			"other-root":      "bar",
			"ignored":         "baz",
		}, []string{"googleapis-root", "other-root"}},
	}

	for _, c := range testCases {
		got := AllSourceRoots(c.input)
		less := func(a, b string) bool { return a < b }
		if diff := cmp.Diff(c.want, got, cmpopts.SortSlices(less)); diff != "" {
			t.Errorf("AllSourceRoots mismatch (-want, +got):\n%s", diff)
		}
	}
}
