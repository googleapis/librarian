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

// Package github provides operations on GitHub repos, abstracting away go-github
// (at least somewhat) to only the operations Librarian needs.
package github

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/go-github/v69/github"
)

const (
	maxRetries = 3
	retryDelay = 2 * time.Second
)

type retryableTransport struct {
	transport http.RoundTripper
}

// RoundTrip implements the http.RoundTripper interface and adds retry logic
// for transient server errors.
func (t *retryableTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	var resp *http.Response
	var err error
	for i := 0; i < maxRetries; i++ {
		resp, err = t.transport.RoundTrip(req)
		if err == nil && resp.StatusCode != http.StatusServiceUnavailable {
			return resp, nil
		}
		if err != nil {
			slog.Warn("retrying due to error", "err", err)
		} else {
			slog.Warn("retrying due to status code", "status_code", resp.StatusCode)
		}

		time.Sleep(retryDelay)
	}
	return resp, err
}

// PullRequest is a type alias for the go-github type.
type PullRequest = github.PullRequest

// NewPullRequest is a type alias for the go-github type.
type NewPullRequest = github.NewPullRequest

// RepositoryCommit is a type alias for the go-github type.
type RepositoryCommit = github.RepositoryCommit

// PullRequestReview is a type alias for the go-github type.
type PullRequestReview = github.PullRequestReview

// RepositoryRelease is a type alias for the go-github type.
type RepositoryRelease = github.RepositoryRelease

// Client represents this package's abstraction of a GitHub client, including
// an access token.
type Client struct {
	*github.Client
	accessToken string
	repo        *Repository
}

// NewClient creates a new Client to interact with GitHub.
func NewClient(accessToken string, repo *Repository) *Client {
	return newClientWithHTTP(accessToken, repo, nil)
}

func newClientWithHTTP(accessToken string, repo *Repository, httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = &http.Client{}
	}
	transport := httpClient.Transport
	if transport == nil {
		transport = http.DefaultTransport
	}
	httpClient.Transport = &retryableTransport{transport: transport}
	client := github.NewClient(httpClient)
	if repo != nil && repo.BaseURL != "" {
		baseURL, _ := url.Parse(repo.BaseURL)
		// Ensure the endpoint URL has a trailing slash.
		if !strings.HasSuffix(baseURL.Path, "/") {
			baseURL.Path += "/"
		}
		client.BaseURL = baseURL
	}
	if accessToken != "" {
		client = client.WithAuthToken(accessToken)
	}
	return &Client{
		Client:      client,
		accessToken: accessToken,
		repo:        repo,
	}
}

// Token returns the access token for Client.
func (c *Client) Token() string {
	return c.accessToken
}

// Repository represents a GitHub repository with an owner (e.g. an organization or a user)
// and a repository name.
type Repository struct {
	// The owner of the repository.
	Owner string
	// The name of the repository.
	Name string
	// Base URL for API requests.
	BaseURL string
}

// PullRequestMetadata identifies a pull request within a repository.
type PullRequestMetadata struct {
	// Repo is the repository containing the pull request.
	Repo *Repository
	// Number is the number of the pull request.
	Number int
}

// ParseRemote parses a GitHub remote (anything to do with a repository) to determine
// the GitHub repo details (owner and name).
func ParseRemote(remote string) (*Repository, error) {
	if strings.HasPrefix(remote, "https://github.com/") {
		return parseHTTPRemote(remote)
	}
	if strings.HasPrefix(remote, "git@") {
		return parseSSHRemote(remote)
	}
	return nil, fmt.Errorf("remote '%s' is not a GitHub remote", remote)
}

func parseHTTPRemote(remote string) (*Repository, error) {
	remotePath := remote[len("https://github.com/"):]
	pathParts := strings.Split(remotePath, "/")
	organization := pathParts[0]
	repoName := pathParts[1]
	repoName = strings.TrimSuffix(repoName, ".git")
	return &Repository{Owner: organization, Name: repoName}, nil
}

func parseSSHRemote(remote string) (*Repository, error) {
	pathParts := strings.Split(remote, ":")
	if len(pathParts) != 2 {
		return nil, fmt.Errorf("remote %q is not a GitHub remote", remote)
	}
	orgRepo := strings.Split(pathParts[1], "/")
	if len(orgRepo) != 2 {
		return nil, fmt.Errorf("remote %q is not a GitHub remote", remote)
	}
	organization := orgRepo[0]
	repoName := strings.TrimSuffix(orgRepo[1], ".git")
	return &Repository{Owner: organization, Name: repoName}, nil
}

// GetRawContent fetches the raw content of a file within a repository repo,
// identifying the file by path, at a specific commit/tag/branch of ref.
func (c *Client) GetRawContent(ctx context.Context, path, ref string) ([]byte, error) {
	options := &github.RepositoryContentGetOptions{
		Ref: ref,
	}
	body, _, err := c.Repositories.DownloadContents(ctx, c.repo.Owner, c.repo.Name, path, options)
	if err != nil {
		return nil, err
	}
	defer body.Close()
	return io.ReadAll(body)
}

// CreatePullRequest creates a pull request in the remote repo.
// At the moment this requires a single remote to be configured,
// which must have a GitHub HTTPS URL. We assume a base branch of "main".
func (c *Client) CreatePullRequest(ctx context.Context, repo *Repository, remoteBranch, baseBranch, title, body string, isDraft bool) (*PullRequestMetadata, error) {
	if body == "" {
		slog.Warn("provided PR body is empty, setting default.")
		body = "Regenerated all changed APIs. See individual commits for details."
	}
	slog.Info("creating PR", "branch", remoteBranch, "base", baseBranch, "title", title)
	// The body may be excessively long, only display in debug mode.
	slog.Debug("with PR body", "body", body)
	newPR := &github.NewPullRequest{
		Title:               &title,
		Head:                &remoteBranch,
		Base:                &baseBranch,
		Body:                github.Ptr(body),
		MaintainerCanModify: github.Ptr(true),
		Draft:               github.Ptr(isDraft),
	}
	pr, _, err := c.PullRequests.Create(ctx, repo.Owner, repo.Name, newPR)
	if err != nil {
		return nil, err
	}

	slog.Info("pr created", "url", pr.GetHTMLURL())
	pullRequestMetadata := &PullRequestMetadata{Repo: repo, Number: pr.GetNumber()}
	return pullRequestMetadata, nil
}

// GetLabels fetches the labels for an issue.
func (c *Client) GetLabels(ctx context.Context, number int) ([]string, error) {
	slog.Info("getting labels", "number", number)
	var allLabels []string
	opts := &github.ListOptions{
		PerPage: 100,
	}
	for {
		labels, resp, err := c.Issues.ListLabelsByIssue(ctx, c.repo.Owner, c.repo.Name, number, opts)
		if err != nil {
			return nil, err
		}
		for _, label := range labels {
			allLabels = append(allLabels, *label.Name)
		}
		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}
	return allLabels, nil
}

// ReplaceLabels replaces all labels for an issue.
func (c *Client) ReplaceLabels(ctx context.Context, number int, labels []string) error {
	slog.Info("replacing labels", "number", number, "labels", labels)
	_, _, err := c.Issues.ReplaceLabelsForIssue(ctx, c.repo.Owner, c.repo.Name, number, labels)
	return err
}

// AddLabelsToIssue adds labels to an existing issue in a GitHub repository.
func (c *Client) AddLabelsToIssue(ctx context.Context, repo *Repository, number int, labels []string) error {
	slog.Info("labels added to issue", "number", number, "labels", labels)
	_, _, err := c.Issues.AddLabelsToIssue(ctx, repo.Owner, repo.Name, number, labels)
	return err
}

// SearchPullRequests searches for pull requests in the repository using the provided raw query.
func (c *Client) SearchPullRequests(ctx context.Context, query string) ([]*PullRequest, error) {
	var prs []*PullRequest
	opts := &github.SearchOptions{
		ListOptions: github.ListOptions{PerPage: 100},
	}
	query = fmt.Sprintf("repo:%s/%s %s", c.repo.Owner, c.repo.Name, query)
	for {
		result, resp, err := c.Search.Issues(ctx, query, opts)
		if err != nil {
			return nil, err
		}
		for _, issue := range result.Issues {
			if issue.IsPullRequest() {
				pr, _, err := c.PullRequests.Get(ctx, c.repo.Owner, c.repo.Name, issue.GetNumber())
				if err != nil {
					return nil, err
				}
				prs = append(prs, pr)
			}
		}
		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}
	return prs, nil
}

// GetPullRequest gets a pull request by its number.
func (c *Client) GetPullRequest(ctx context.Context, number int) (*PullRequest, error) {
	pr, _, err := c.PullRequests.Get(ctx, c.repo.Owner, c.repo.Name, number)
	return pr, err
}

// CreateRelease creates a tag and release in the repository at the given
// commit-ish. See
// https://git-scm.com/docs/gitglossary#Documentation/gitglossary.txt-commit-ishalsocommittish
// for definition of commit-ish.
func (c *Client) CreateRelease(ctx context.Context, tagName, name, body, commitish string) (*github.RepositoryRelease, error) {
	r, _, err := c.Repositories.CreateRelease(ctx, c.repo.Owner, c.repo.Name, &github.RepositoryRelease{
		TagName:         &tagName,
		Name:            &name,
		Body:            &body,
		TargetCommitish: &commitish,
	})
	return r, err
}

// CreateIssueComment adds a comment to the issue number provided.
func (c *Client) CreateIssueComment(ctx context.Context, number int, comment string) error {
	_, _, err := c.Issues.CreateComment(ctx, c.repo.Owner, c.repo.Name, number, &github.IssueComment{
		Body: &comment,
	})
	return err
}

// hasLabel checks if a pull request has a given label.
func hasLabel(pr *PullRequest, labelName string) bool {
	for _, l := range pr.Labels {
		if l.GetName() == labelName {
			return true
		}
	}
	return false
}

// FindMergedPullRequestsWithPendingReleaseLabel finds all merged pull requests with the "release:pending" label.
func (c *Client) FindMergedPullRequestsWithPendingReleaseLabel(ctx context.Context, owner, repo string) ([]*PullRequest, error) {
	return c.FindMergedPullRequestsWithLabel(ctx, owner, repo, "release:pending")
}

// FindMergedPullRequestsWithLabel finds all merged pull requests with the "release:pending" label.
func (c *Client) FindMergedPullRequestsWithLabel(ctx context.Context, owner, repo, label string) ([]*PullRequest, error) {
	var allPRs []*PullRequest
	opt := &github.PullRequestListOptions{
		State: "closed",
		ListOptions: github.ListOptions{
			PerPage: 100,
		},
	}
	for {
		prs, resp, err := c.PullRequests.List(ctx, owner, repo, opt)
		if err != nil {
			return nil, err
		}
		for _, pr := range prs {
			if !pr.GetMergedAt().IsZero() && hasLabel(pr, label) {
				allPRs = append(allPRs, pr)
			}
		}
		if resp.NextPage == 0 || len(allPRs) >= 10 {
			break
		}
		opt.Page = resp.NextPage
	}
	return allPRs, nil
}

// CreateTag creates a lightweight tag in the repository at the given commit SHA.
// This does NOT create a release, just the tag.
func (c *Client) CreateTag(ctx context.Context, tagName, commitSHA string) error {
	slog.Info("creating tag", "tag", tagName, "commit", commitSHA)
	ref := "refs/tags/" + tagName
	tagRef := &github.Reference{
		Ref:    github.Ptr(ref),
		Object: &github.GitObject{SHA: github.Ptr(commitSHA), Type: github.Ptr("commit")},
	}
	_, _, err := c.Git.CreateRef(ctx, c.repo.Owner, c.repo.Name, tagRef)
	return err
}

// ClosePullRequest closes the pull request specified by pull request number.
func (c *Client) ClosePullRequest(ctx context.Context, number int) error {
	slog.Info("closing pull request", slog.Int("number", number))
	state := "closed"
	_, _, err := c.PullRequests.Edit(ctx, c.repo.Owner, c.repo.Name, number, &github.PullRequest{
		State: &state,
	})
	return err
}
