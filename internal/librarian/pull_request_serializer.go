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
	"fmt"
	"log/slog"
	"regexp"
)

// NewGistOverflowHandler handles storing overflow content in a gist.
type GistOverflowHandler struct {
	github         GitHubClient
	maxContentSize int
}

const maxPullRequestBodySize = 65536

var overflowPullRequestRegex = regexp.MustCompile(`See full release notes at: https://gist.github.com/[^\/]+/([0-9a-f]+)`)

// NewGistOverflowHandler returns a handler for storing overflow content in a gist.
func NewGistOverflowHandler(github GitHubClient, maxContentSize int) (*GistOverflowHandler, error) {
	if maxContentSize == 0 {
		maxContentSize = maxPullRequestBodySize
	}
	return &GistOverflowHandler{
		github:         github,
		maxContentSize: maxContentSize,
	}, nil
}

// SavePullRequestBody stores content in a gist if it's too big and returns a minimized version.
func (g *GistOverflowHandler) SavePullRequestBody(ctx context.Context, body string) (string, error) {
	if len(body) > g.maxContentSize {
		slog.Info("content is too big, saving to gist", slog.Int("len", len(body)))
		contents := map[string]string{
			"release-notes.md": body,
		}
		gist, err := g.github.CreateGist(ctx, contents, true)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("See full release notes at: %s", gist.Url), nil
	}
	return body, nil
}

// FetchPullRequestBody restores content from a stroed gist.
func (g *GistOverflowHandler) FetchPullRequestBody(ctx context.Context, body string) (string, error) {
	matches := overflowPullRequestRegex.FindStringSubmatch(body)
	if len(matches) == 2 {
		slog.Info("found gist in pull request body", "gistID", matches[1])
		contents, err := g.github.GetGistContent(ctx, matches[1])
		if err != nil {
			return "", err
		}
		content, found := contents["release-notes.md"]
		if found {
			return content, nil
		}
		return "", fmt.Errorf("unable to find release-notes.md from gist %s", matches[1])
	}
	return body, nil
}
