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

package main

import (
	"context"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/fetch"
)

var (
	fetchSource     = fetchGoogleapis
	githubEndpoints = &fetch.Endpoints{
		API:      "https://api.github.com",
		Download: "https://github.com",
	}
)

func fetchGoogleapis(ctx context.Context) (*config.Source, error) {
	return fetchGoogleapisWithCommit(ctx, githubEndpoints, fetch.DefaultBranchMaster)
}

func fetchGoogleapisWithCommit(ctx context.Context, endpoints *fetch.Endpoints, commitish string) (*config.Source, error) {
	repo := &fetch.RepoRef{
		Org:    "googleapis",
		Name:   "googleapis",
		Branch: commitish,
	}
	commit, sha256, err := fetch.LatestCommitAndChecksum(endpoints, repo)
	if err != nil {
		return nil, err
	}

	dir, err := fetch.Repo(ctx, googleapisRepo, commit, sha256)
	if err != nil {
		return nil, err
	}

	return &config.Source{
		Commit: commit,
		SHA256: sha256,
		Dir:    dir,
	}, nil
}
