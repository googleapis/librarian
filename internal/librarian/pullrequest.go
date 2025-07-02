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
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/googleapis/librarian/internal/github"
	"github.com/googleapis/librarian/internal/gitrepo"
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

// Add details to a PullRequestContent of a partial error which prevents a
// single API or library from being configured/regenerated/released,
// but without halting the overall process. A warning is logged locally with the error details,
// but we don't include detailed errors in the PR, as this could reveal sensitive information.
// The action should describe what failed, e.g. "configuring", "building", "generating".
func addErrorToPullRequest(pr *PullRequestContent, id string, err error, action string) {
	slog.Warn("Error", "action", action, "id", id, "err", err)
	pr.Errors = append(pr.Errors, fmt.Sprintf("Error while %s %s", action, id))
}

// Adds a success entry to a PullRequestContent.
func addSuccessToPullRequest(pr *PullRequestContent, text string) {
	pr.Successes = append(pr.Successes, text)
}

// createPullRequest creates a GitHub pull request based on the given content,
// with a title prefix (e.g. "feat: API regeneration")
// using a branch with a name of the form "librarian-{branchtype}-{timestamp}".
//
// If content is empty, the pull request is not created and no error is
// returned.
//
// If content only contains errors, the pull request is not created and an
// error is returned (to highlight that everything failed)
// If content contains any successes, a pull request is created and no error is
// returned (if the creation is successful) even if the content includes
// errors.
//
// If the pull request would contain an excessive number of commits (as
// configured in pipeline-config.json).
func createPullRequest(
	ctx context.Context,
	state *commandState,
	content *PullRequestContent,
	titlePrefix, descriptionSuffix, branchType string,
	gitHubToken string,
	push bool,
) (*github.PullRequestMetadata, error) {
	ghClient, err := github.NewClient(gitHubToken)
	if err != nil {
		return nil, err
	}
	anySuccesses := len(content.Successes) > 0
	anyErrors := len(content.Errors) > 0
	languageRepo := state.languageRepo

	excessSuccesses := []string{}
	if state.pipelineConfig != nil {
		maxCommits := int(state.pipelineConfig.MaxPullRequestCommits)
		if maxCommits > 0 && len(content.Successes) > maxCommits {
			// We've got too many commits. Roll some back locally, and we'll add them to the description.
			excessSuccesses = content.Successes[maxCommits:]
			content.Successes = content.Successes[:maxCommits]
			slog.Info(
				"Excess commits created; winding back language repo",
				"excess_commits",
				len(excessSuccesses),
			)
			if err := languageRepo.CleanAndRevertCommits(len(excessSuccesses)); err != nil {
				return nil, err
			}
		}
	}

	var description string
	if !anySuccesses && !anyErrors {
		slog.Info("No PR to create, and no errors.")
		return nil, nil
	} else if !anySuccesses && anyErrors {
		slog.Error("No PR to create, but errors were logged (and restated below). Aborting.")
		for _, error := range content.Errors {
			slog.Error(error)
		}
		return nil, errors.New("errors encountered but no PR to create")
	}

	successesText := formatListAsMarkdown("Changes in this PR", content.Successes)
	errorsText := formatListAsMarkdown("Errors", content.Errors)
	excessText := formatListAsMarkdown("Excess changes not included", excessSuccesses)

	description = strings.TrimSpace(
		successesText + errorsText + excessText + "\n" + descriptionSuffix,
	)

	title := fmt.Sprintf("%s: %s", titlePrefix, formatTimestamp(state.startTime))

	if !push {
		slog.Info(
			"Push not specified; would have created PR",
			"title",
			title,
			"description",
			description,
		)
		return nil, nil
	}

	gitHubRepo, err := getGitHubRepoFromRemote(languageRepo)
	if err != nil {
		return nil, err
	}

	branch := fmt.Sprintf("librarian-%s-%s", branchType, formatTimestamp(state.startTime))
	err = languageRepo.PushBranch(branch, ghClient.Token())
	if err != nil {
		slog.Info("Received error pushing branch", "err", err)
		return nil, err
	}
	return ghClient.CreatePullRequest(ctx, gitHubRepo, branch, title, description)
}

// Formats the given list as a single Markdown string, with a title preceding the list,
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

// Parses the GitHub repo name from the remote for this repository.
// There must only be a single remote with a GitHub URL (as the first URL), in order to provide an
// unambiguous result.
// Remotes without any URLs, or where the first URL does not start with https://github.com/ are ignored.
func getGitHubRepoFromRemote(repo *gitrepo.Repository) (*github.Repository, error) {
	remotes, err := repo.Remotes()
	if err != nil {
		return nil, err
	}
	gitHubRemoteNames := []string{}
	gitHubUrl := ""
	for _, remote := range remotes {
		urls := remote.Config().URLs
		if len(urls) > 0 && strings.HasPrefix(urls[0], "https://github.com/") {
			gitHubRemoteNames = append(gitHubRemoteNames, remote.Config().Name)
			gitHubUrl = urls[0]
		}
	}

	if len(gitHubRemoteNames) == 0 {
		return nil, fmt.Errorf("no GitHub remotes found")
	}

	if len(gitHubRemoteNames) > 1 {
		joinedRemoteNames := strings.Join(gitHubRemoteNames, ", ")
		return nil, fmt.Errorf(
			"can only determine the GitHub repo with a single matching remote; GitHub remotes in repo: %s",
			joinedRemoteNames,
		)
	}
	return github.ParseUrl(gitHubUrl)
}
