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
	"errors"
	"slices"
	"testing"
)

func TestAddErrorToPullRequest(t *testing.T) {
	pr := PullRequestContent{
		Successes: []string{"s1", "s2"},
		Errors:    []string{"e1", "e2"},
	}
	addErrorToPullRequest(&pr, "library-id", errors.New("bang"), "generating")
	if !slices.Equal([]string{"s1", "s2"}, pr.Successes) {
		t.Errorf("addErrorToPullRequest modified successes: %q", pr.Successes)
	}
	// Note the error isn't included here; it's logged, but not added to the PullRequestContent
	want := "Error while generating library-id"
	if !slices.Equal([]string{"e1", "e2", want}, pr.Errors) {
		t.Errorf("unexpected errors after addErrorToPullRequest: %q", pr.Errors)
	}
}

func TestAddSuccessToPullRequest(t *testing.T) {
	pr := PullRequestContent{
		Successes: []string{"s1", "s2"},
		Errors:    []string{"e1", "e2"},
	}
	addSuccessToPullRequest(&pr, "s3")
	if !slices.Equal([]string{"s1", "s2", "s3"}, pr.Successes) {
		t.Errorf("unexpected successes after addSuccessToPullRequest: %q", pr.Successes)
	}
	if !slices.Equal([]string{"e1", "e2"}, pr.Errors) {
		t.Errorf("addSuccessToPullRequest modified errors: %q", pr.Errors)
	}
}

func TestFormatListAsMarkdownEmptyList(t *testing.T) {
	got := formatListAsMarkdown("Title", []string{})
	if got != "" {
		t.Errorf("formatting empty list produced non-empty output %q", got)
	}
}

func TestFormatListAsMarkdownNonEmptyList(t *testing.T) {
	got := formatListAsMarkdown("Title", []string{"x1", "x2"})
	want := "## Title\n\n- x1\n- x2\n\n\n"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}
