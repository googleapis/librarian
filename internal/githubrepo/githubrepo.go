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

const GitHubTokenEnvironmentVariable string = "LIBRARIAN_GITHUB_TOKEN"

type Client struct {
	*github.Client
	accessToken string
}

func NewClient() (*Client, error) {
	accessToken := os.Getenv(GitHubTokenEnvironmentVariable)
	return &Client{
		Client:      github.NewClient(nil).WithAuthToken(accessToken),
		accessToken: accessToken,
	}, nil
}

func (c *Client) Token() string {
	return c.accessToken
}

type Repository struct {
	Owner string
	Name  string
}

type PullRequestMetadata struct {
	Repo   *Repository
	Number int
}

// Creates a pull request in the remote repo. At the moment this requires a single remote to be
// configured, which must have a GitHub HTTPS URL. We assume a base branch of "main".
func (c *Client) CreatePullRequest(ctx context.Context, repo *Repository, remoteBranch, title, body string) (*PullRequestMetadata, error) {
	if body == "" {
		body = "Regenerated all changed APIs. See individual commits for details."
	}
	newPR := &github.NewPullRequest{
		Title:               &title,
		Head:                &remoteBranch,
		Base:                github.Ptr("main"),
		Body:                github.Ptr(body),
		MaintainerCanModify: github.Ptr(true),
	}
	pr, _, err := c.PullRequests.Create(ctx, repo.Owner, repo.Name, newPR)
	if err != nil {
		return nil, err
	}

	fmt.Printf("PR created: %s\n", pr.GetHTMLURL())
	pullRequestMetadata := &PullRequestMetadata{Repo: repo, Number: pr.GetNumber()}
	return pullRequestMetadata, nil
}

func (c *Client) CreateRelease(ctx context.Context, repo *Repository, tag, commit, title, description string, prerelease bool) (*github.RepositoryRelease, error) {
	release := &github.RepositoryRelease{
		TagName:         &tag,
		TargetCommitish: &commit,
		Name:            &title,
		Body:            &description,
		Draft:           github.Ptr(false),
		MakeLatest:      github.Ptr("true"),
		Prerelease:      &prerelease,
		// TODO(https://github.com/googleapis/librarian/issues/541): check GenerateReleaseNotes value
		GenerateReleaseNotes: github.Ptr(false),
	}
	release, _, err := c.Repositories.CreateRelease(ctx, repo.Owner, repo.Name, release)
	return release, err
}

func (c *Client) AddLabelToPullRequest(ctx context.Context, prMetadata *PullRequestMetadata, label string) error {
	labels := []string{label}

	_, _, err := c.Issues.AddLabelsToIssue(ctx, prMetadata.Repo.Owner, prMetadata.Repo.Name, prMetadata.Number, labels)
	if err != nil {
		return fmt.Errorf("failed to add label: %w", err)
	}
	return nil
}

func (c *Client) RemoveLabelFromPullRequest(ctx context.Context, repo *Repository, prNumber int, label string) error {
	_, err := c.Issues.RemoveLabelForIssue(ctx, repo.Owner, repo.Name, prNumber, label)
	if err != nil {
		return fmt.Errorf("failed to remove label: %w", err)
	}
	return nil
}

func (c *Client) AddCommentToPullRequest(ctx context.Context, repo *Repository, prNumber int, comment string) error {
	issueComment := &github.IssueComment{
		Body: &comment,
	}
	_, _, err := c.Issues.CreateComment(ctx, repo.Owner, repo.Name, prNumber, issueComment)
	if err != nil {
		return fmt.Errorf("failed to add comment: %w", err)
	}
	return nil
}

func (c *Client) MergePullRequest(ctx context.Context, repo *Repository, prNumber int, method github.MergeMethod) (*github.PullRequestMergeResult, error) {
	options := &github.PullRequestOptions{
		MergeMethod: string(method),
	}
	result, _, err := c.PullRequests.Merge(ctx, repo.Owner, repo.Name, prNumber, "", options)
	if err != nil {
		return nil, fmt.Errorf("failed to merge pull request: %w", err)
	}
	return result, nil
}

func (c *Client) GetPullRequest(ctx context.Context, repo *Repository, prNumber int) (*github.PullRequest, error) {
	pr, _, err := c.PullRequests.Get(ctx, repo.Owner, repo.Name, prNumber)
	return pr, err
}

func (c *Client) GetPullRequestCheckRuns(ctx context.Context, pullRequest *github.PullRequest) ([]*github.CheckRun, error) {
	prHead := pullRequest.Head
	options := &github.ListCheckRunsOptions{}
	checkRuns, _, err := c.Checks.ListCheckRunsForRef(ctx, *prHead.User.Login, *prHead.Repo.Name, *prHead.Ref, options)
	if checkRuns == nil {
		return nil, err
	}
	return checkRuns.CheckRuns, err
}

func (c *Client) GetPullRequestReviews(ctx context.Context, prMetadata *PullRequestMetadata) ([]*github.PullRequestReview, error) {
	// TODO(https://github.com/googleapis/librarian/issues/540): implement pagination or use go-github-paginate
	listOptions := &github.ListOptions{PerPage: 100}
	reviews, _, err := c.PullRequests.ListReviews(ctx, prMetadata.Repo.Owner, prMetadata.Repo.Name, prMetadata.Number, listOptions)
	return reviews, err
}

// Parses a GitHub URL (anything to do with a repository) to determine
// the GitHub repo details (owner and name)
func ParseUrl(remoteUrl string) (*Repository, error) {
	if !strings.HasPrefix(remoteUrl, "https://github.com/") {
		return nil, fmt.Errorf("remote '%s' is not a GitHub remote", remoteUrl)
	}
	remotePath := remoteUrl[len("https://github.com/"):]
	pathParts := strings.Split(remotePath, "/")
	organization := pathParts[0]
	repoName := pathParts[1]
	repoName = strings.TrimSuffix(repoName, ".git")
	return &Repository{Owner: organization, Name: repoName}, nil
}

func CreateGitHubRepoFromRepository(repo *github.Repository) *Repository {
	return &Repository{Owner: *repo.Owner.Login, Name: *repo.Name}
}

func (c *Client) GetRawContent(ctx context.Context, repo *Repository, path, ref string) ([]byte, error) {
	options := &github.RepositoryContentGetOptions{
		Ref: ref,
	}
	closer, _, err := c.Repositories.DownloadContents(ctx, repo.Owner, repo.Name, path, options)
	if err != nil {
		return nil, err
	}
	defer closer.Close()
	return io.ReadAll(closer)
}

func (c *Client) GetDiffCommits(ctx context.Context, repo *Repository, source, target string) ([]*github.RepositoryCommit, error) {
	// TODO(https://github.com/googleapis/librarian/issues/540): implement pagination or use go-github-paginate
	listOptions := &github.ListOptions{PerPage: 100}
	commitsComparison, _, err := c.Repositories.CompareCommits(ctx, repo.Owner, repo.Name, source, target, listOptions)
	return commitsComparison.Commits, err
}

func (c *Client) GetCommit(ctx context.Context, repo *Repository, sha string) (*github.RepositoryCommit, error) {
	// TODO(https://github.com/googleapis/librarian/issues/540): implement pagination or use go-github-paginate
	listOptions := &github.ListOptions{PerPage: 100}
	commit, _, err := c.Repositories.GetCommit(ctx, repo.Owner, repo.Name, sha, listOptions)
	return commit, err
}
