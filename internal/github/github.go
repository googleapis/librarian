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
	"net/http"
	"strings"

	"github.com/google/go-github/v69/github"
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

// ParseUrl parses a GitHub URL (anything to do with a repository) to determine
// the GitHub repo details (owner and name).
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
