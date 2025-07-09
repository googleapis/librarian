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
	"fmt"
	"log/slog"
	"strings"
)

// A PullRequestContent builds up the content of a pull request.
// Each entry in "Successes" represents a commit from a successful
// operation; each entry in "Errors" represents an unsuccessful
// operation which would otherwise have created a commit.
// (It's important that each entry in "Successes" represents *exactly*
// one commit, in the same order in which the commits were created. This
// is assumed when observing pull request commit limits.)
type PullRequestContent struct {
	Successes []string
	Errors    []string
}

// addErrorToPullRequest adds details to a PullRequestContent of a partial error which prevents a
// single API or library from being configured/regenerated/released,
// but without halting the overall process. A warning is logged locally with the error details,
// but we don't include detailed errors in the PR, as this could reveal sensitive information.
// The action should describe what failed, e.g. "configuring", "building", "generating".
func addErrorToPullRequest(pr *PullRequestContent, id string, err error, action string) {
	slog.Warn("Error", "action", action, "id", id, "err", err)
	pr.Errors = append(pr.Errors, fmt.Sprintf("Error while %s %s", action, id))
}

// addSuccessToPullRequest adds a success entry to a PullRequestContent.
func addSuccessToPullRequest(pr *PullRequestContent, text string) {
	pr.Successes = append(pr.Successes, text)
}

// formatListAsMarkdown formats the given list as a single Markdown string, with a title preceding the list,
// a "- " at the start of each value and a line break at the end of each value.
// If the list is empty, an empty string is returned instead.
func formatListAsMarkdown(title string, list []string) string {
	if len(list) == 0 {
		return ""
	}
	var builder strings.Builder
	builder.WriteString("## ")
	builder.WriteString(title)
	builder.WriteString("\n\n")
	for _, value := range list {
		builder.WriteString(fmt.Sprintf("- %s\n", value))
	}
	builder.WriteString("\n\n")
	return builder.String()
}
