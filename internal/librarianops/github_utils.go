// Copyright 2026 Google LLC
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

package librarianops

import (
	"context"
	"fmt"
	"time"

	"github.com/googleapis/librarian/internal/command"
)

// uploadToGithub creates a branch, commits changes, pushes the changes, and creates a PR with the given details.
func uploadToGithub(ctx context.Context, githubDetails GithubDetails) error {
	if err := createBranch(ctx, githubDetails.BranchName); err != nil {
		return err
	}
	if err := commitChanges(ctx, githubDetails.PrTitle); err != nil {
		return err
	}
	if err := pushChanges(ctx); err != nil {
		return err
	}
	if err := createPR(ctx, githubDetails); err != nil {
		return err
	}
	return nil
}

func cloneRepoInDir(ctx context.Context, repoName string, repoDir string) error {
	return command.Run(ctx, "gh", "repo", "clone", fmt.Sprintf("googleapis/%s", repoName), repoDir)
}

func generateBranchName(prefix string, time time.Time) string {
	return fmt.Sprintf("%s%s", prefix, time.Format("2006-01-02"))
}

func createBranch(ctx context.Context, branchName string) error {
	return command.Run(ctx, "git", "checkout", "-b", branchName)
}

func commitChanges(ctx context.Context, commitTitle string) error {
	if err := command.Run(ctx, "git", "add", "."); err != nil {
		return err
	}
	return command.Run(ctx, "git", "commit", "-m", commitTitle)
}

func pushChanges(ctx context.Context) error {
	return command.Run(ctx, "git", "push", "-u", "origin", "HEAD")
}

// GithubDetails contains the details for the github changes to be made.
type GithubDetails struct {
	PrTitle    string
	PrBody     string
	BranchName string
}

func createPR(ctx context.Context, githubDetails GithubDetails) error {
	return command.Run(ctx, "gh", "pr", "create", "--title", githubDetails.PrTitle, "--body", githubDetails.PrBody)
}
