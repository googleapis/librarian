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

// Package githubrepo provides operations on GitHub repos, abstracting away go-github
// (at least somewhat) to only the operations Librarian needs.
package githubrepo

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/google/go-github/v69/github"
)

type GitHubRepo struct {
	Owner string
	Name  string
}

type PullRequestMetadata struct {
	Repo   GitHubRepo
	Number int
}

const gitHubTokenEnvironmentVariable string = "LIBRARIAN_GITHUB_TOKEN"

// Creates a pull request in the remote repo. At the moment this requires a single remote to be
// configured, which must have a GitHub HTTPS URL. We assume a base branch of "main".
func CreatePullRequest(ctx context.Context, repo GitHubRepo, remoteBranch string, title string, body string) (*PullRequestMetadata, error) {
	if body == "" {
		body = "Regenerated all changed APIs. See individual commits for details."
	}
	gitHubClient := createClient()
	newPR := &github.NewPullRequest{
		Title:               &title,
		Head:                &remoteBranch,
		Base:                github.Ptr("main"),
		Body:                github.Ptr(body),
		MaintainerCanModify: github.Ptr(true),
	}
	pr, _, err := gitHubClient.PullRequests.Create(ctx, repo.Owner, repo.Name, newPR)
	if err != nil {
		return nil, err
	}

	fmt.Printf("PR created: %s\n", pr.GetHTMLURL())
	pullRequestMetadata := &PullRequestMetadata{Repo: repo, Number: pr.GetNumber()}
	return pullRequestMetadata, nil
}

func CreateRelease(ctx context.Context, repo GitHubRepo, tag, commit, title, description string, prerelease bool) (*github.RepositoryRelease, error) {
	gitHubClient := createClient()

	release := &github.RepositoryRelease{
		TagName:         &tag,
		TargetCommitish: &commit,
		Name:            &title,
		Body:            &description,
		Draft:           github.Ptr(false),
		MakeLatest:      github.Ptr("true"),
		Prerelease:      &prerelease,
		// TODO: Check whether this is what we want
		GenerateReleaseNotes: github.Ptr(false),
	}
	release, _, err := gitHubClient.Repositories.CreateRelease(ctx, repo.Owner, repo.Name, release)
	return release, err
}

func AddLabelToPullRequest(ctx context.Context, prMetadata PullRequestMetadata, label string) error {
	gitHubClient := createClient()

	labels := []string{label}

	_, _, err := gitHubClient.Issues.AddLabelsToIssue(ctx, prMetadata.Repo.Owner, prMetadata.Repo.Name, prMetadata.Number, labels)
	if err != nil {
		return fmt.Errorf("failed to add label: %w", err)
	}
	return nil
}

func RemoveLabelFromPullRequest(ctx context.Context, repo GitHubRepo, prNumber int, label string) error {
	gitHubClient := createClient()

	_, err := gitHubClient.Issues.RemoveLabelForIssue(ctx, repo.Owner, repo.Name, prNumber, label)
	if err != nil {
		return fmt.Errorf("failed to remove label: %w", err)
	}
	return nil
}

func AddCommentToPullRequest(ctx context.Context, repo GitHubRepo, prNumber int, comment string) error {
	gitHubClient := createClient()
	issueComment := &github.IssueComment{
		Body: &comment,
	}
	_, _, err := gitHubClient.Issues.CreateComment(ctx, repo.Owner, repo.Name, prNumber, issueComment)
	if err != nil {
		return fmt.Errorf("failed to add comment: %w", err)
	}
	return nil
}

func MergePullRequest(ctx context.Context, repo GitHubRepo, prNumber int, method github.MergeMethod) (*github.PullRequestMergeResult, error) {
	gitHubClient := createClient()

	options := &github.PullRequestOptions{
		MergeMethod: string(method),
	}
	result, _, err := gitHubClient.PullRequests.Merge(ctx, repo.Owner, repo.Name, prNumber, "", options)
	if err != nil {
		return nil, fmt.Errorf("failed to merge pull request: %w", err)
	}
	return result, nil
}

func GetPullRequest(ctx context.Context, repo GitHubRepo, prNumber int) (*github.PullRequest, error) {
	gitHubClient := createClient()
	pr, _, err := gitHubClient.PullRequests.Get(ctx, repo.Owner, repo.Name, prNumber)
	return pr, err
}

// Parses a GitHub URL (anything to do with a repository) to determine
// the GitHub repo details (owner and name)
func ParseUrl(remoteUrl string) (GitHubRepo, error) {
	if !strings.HasPrefix(remoteUrl, "https://github.com/") {
		return GitHubRepo{}, fmt.Errorf("remote '%s' is not a GitHub remote", remoteUrl)
	}
	remotePath := remoteUrl[len("https://github.com/"):]
	pathParts := strings.Split(remotePath, "/")
	organization := pathParts[0]
	repoName := pathParts[1]
	repoName = strings.TrimSuffix(repoName, ".git")
	return GitHubRepo{Owner: organization, Name: repoName}, nil
}

func CreateGitHubRepoFromRepository(repo *github.Repository) GitHubRepo {
	return GitHubRepo{Owner: *repo.Owner.Login, Name: *repo.Name}
}

func GetRawContent(ctx context.Context, repo GitHubRepo, path, ref string) ([]byte, error) {
	gitHubClient := github.NewClient(nil)
	options := &github.RepositoryContentGetOptions{
		Ref: ref,
	}
	closer, _, err := gitHubClient.Repositories.DownloadContents(ctx, repo.Owner, repo.Name, path, options)
	if err != nil {
		return nil, err
	}
	defer closer.Close()
	return io.ReadAll(closer)
}

func GetDiffCommits(ctx context.Context, repo GitHubRepo, source, target string) ([]*github.RepositoryCommit, error) {
	gitHubClient := createClient()
	// TODO: Implement pagination or use go-github-paginate
	listOptions := &github.ListOptions{PerPage: 100}
	commitsComparison, _, err := gitHubClient.Repositories.CompareCommits(ctx, repo.Owner, repo.Name, source, target, listOptions)
	return commitsComparison.Commits, err
}

func GetCommit(ctx context.Context, repo GitHubRepo, sha string) (*github.RepositoryCommit, error) {
	gitHubClient := createClient()
	// TODO: Implement pagination or use go-github-paginate (if necessary)
	listOptions := &github.ListOptions{PerPage: 100}
	commit, _, err := gitHubClient.Repositories.GetCommit(ctx, repo.Owner, repo.Name, sha, listOptions)
	return commit, err
}

func createClient() *github.Client {
	accessToken := GetAccessToken()
	return github.NewClient(nil).WithAuthToken(accessToken)
}

func GetAccessToken() string {
	return os.Getenv(gitHubTokenEnvironmentVariable)
}
