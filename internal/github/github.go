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
	"strings"

	"github.com/google/go-github/v69/github"
	"github.com/googleapis/librarian/internal/gitrepo"
)

// PullRequest is a type alias for the go-github type.
type PullRequest = github.PullRequest

// RepositoryCommit is a type alias for the go-github type.
type RepositoryCommit = github.RepositoryCommit

// PullRequestReview is a type alias for the go-github type.
type PullRequestReview = github.PullRequestReview

// MergeMethodRebase is a constant alias for the go-github constant.
const MergeMethodRebase = github.MergeMethodRebase

// Client represents this package's abstraction of a GitHub client, including
// an access token.
type Client struct {
	*github.Client
	accessToken string
	repo        *Repository
}

// NewClient creates a new Client to interact with GitHub.
func NewClient(accessToken string, repo *Repository) (*Client, error) {
	return newClientWithHTTP(accessToken, repo, nil)
}

func newClientWithHTTP(accessToken string, repo *Repository, httpClient *http.Client) (*Client, error) {
	return &Client{
		Client:      github.NewClient(httpClient).WithAuthToken(accessToken),
		accessToken: accessToken,
		repo:        repo,
	}, nil
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
}

// PullRequestMetadata identifies a pull request within a repository.
type PullRequestMetadata struct {
	// Repo is the repository containing the pull request.
	Repo *Repository
	// Number is the number of the pull request.
	Number int
}

// ParseURL parses a GitHub URL (anything to do with a repository) to determine
// the GitHub repo details (owner and name).
func ParseURL(remoteURL string) (*Repository, error) {
	if !strings.HasPrefix(remoteURL, "https://github.com/") {
		return nil, fmt.Errorf("remote '%s' is not a GitHub remote", remoteURL)
	}
	remotePath := remoteURL[len("https://github.com/"):]
	pathParts := strings.Split(remotePath, "/")
	organization := pathParts[0]
	repoName := pathParts[1]
	repoName = strings.TrimSuffix(repoName, ".git")
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

	slog.Info("PR created", "url", pr.GetHTMLURL())
	pullRequestMetadata := &PullRequestMetadata{Repo: repo, Number: pr.GetNumber()}
	return pullRequestMetadata, nil
}

// FetchGitHubRepoFromRemote parses the GitHub repo name from the remote for this repository.
// There must be a remote named 'origin' with a Github URL (as the first URL), in order to
// provide an unambiguous result.
// Remotes without any URLs, or where the first URL does not start with https://github.com/ are ignored.
func FetchGitHubRepoFromRemote(repo gitrepo.Repository) (*Repository, error) {
	remotes, err := repo.Remotes()
	if err != nil {
		return nil, err
	}

	for _, remote := range remotes {
		if remote.Config().Name == "origin" {
			urls := remote.Config().URLs
			if len(urls) > 0 && strings.HasPrefix(urls[0], "https://github.com/") {
				return ParseURL(urls[0])
			}
			// If 'origin' exists but is not a GitHub remote, we stop.
			break
		}
	}

	return nil, fmt.Errorf("could not find an 'origin' remote pointing to a GitHub https URL")
}
