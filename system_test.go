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
	"os"
	"path"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/github"
	"github.com/googleapis/librarian/internal/gitrepo"
)

var testToken = os.Getenv("TEST_GITHUB_TOKEN")

func TestGetRawContentSystem(t *testing.T) {
	if testToken == "" {
		t.Skip("TEST_GITHUB_TOKEN not set, skipping GitHub integration test")
	}
	repoName := "https://github.com/googleapis/librarian"

	for _, credential := range []struct {
		name  string
		token string
	}{
		{
			name:  "with credentials",
			token: testToken,
		},
		{
			name:  "without credentials",
			token: "",
		},
	} {
		t.Run(credential.name, func(t *testing.T) {
			for _, test := range []struct {
				name          string
				path          string
				wantErr       bool
				wantErrSubstr string
			}{
				{
					name:    "existing file",
					path:    ".librarian/state.yaml",
					wantErr: false,
				},
				{
					name:          "missing file",
					path:          "not-a-real-file.txt",
					wantErr:       true,
					wantErrSubstr: "no file named",
				},
			} {
				t.Run(test.name, func(t *testing.T) {
					repo, err := github.ParseRemote(repoName)
					if err != nil {
						t.Fatalf("unexpected error in ParseRemote() %s", err)
					}

					client := github.NewClient(testToken, repo)
					got, err := client.GetRawContent(context.Background(), test.path, "main")

					if test.wantErr {
						if err == nil {
							t.Fatalf("GetRawContent() err = nil, want error containing %q", test.wantErrSubstr)
						} else if !strings.Contains(err.Error(), test.wantErrSubstr) {
							t.Errorf("GetRawContent() err = %v, want error containing %q", err, test.wantErrSubstr)
						}
						return
					}
					if err != nil {
						t.Errorf("GetRawContent() err = %v, want nil", err)
					}
					if len(got) <= 0 {
						t.Fatalf("GetRawContent() expected to fetch contents for %s from %s", test.path, repoName)
					}
				})
			}
		})
	}
}

func TestPullRequestSystem(t *testing.T) {
	// Clone a repo
	// Create a commit and push
	// Create a pull request
	// Add a label to the pull request
	// Fetch labels for the issue and verify
	// Replace the issue labels
	// Fetch labels for the issue and verify
	// Add a comment
	// Search for the pull request
	// Fetch the pull request
	// Close the pull request
	if testToken == "" {
		t.Skip("TEST_GITHUB_TOKEN not set, skipping GitHub integration test")
	}
	testRepoName := os.Getenv("TEST_GITHUB_REPO")
	if testRepoName == "" {
		t.Skip("TEST_GITHUB_REPO not set, skipping GitHub integration test")
	}

	// Clone a repo
	workdir := path.Join(t.TempDir(), "test-repo")
	localRepository, err := gitrepo.NewRepository(&gitrepo.RepositoryOptions{
		Dir:          workdir,
		MaybeClone:   true,
		RemoteURL:    testRepoName,
		RemoteBranch: "main",
		GitPassword:  testToken,
		Depth:        1,
	})
	if err != nil {
		t.Fatalf("unexpected error in NewRepository() %s", err)
	}
	repo, err := github.ParseRemote(testRepoName)
	if err != nil {
		t.Fatalf("unexpected error in ParseRemote() %s", err)
	}

	now := time.Now()
	branchName := fmt.Sprintf("integration-test-%s", now.Format("20060102150405"))
	err = localRepository.CreateBranchAndCheckout(branchName)
	if err != nil {
		t.Fatalf("unexpected error in CreateBranchAndCheckout() %s", err)
	}

	// Create a commit and push
	err = os.WriteFile(path.Join(workdir, "some-file.txt"), []byte("some-content"), 0644)
	if err != nil {
		t.Fatalf("unexpected error writing a file to git repo %s", err)
	}
	err = localRepository.AddAll()
	if err != nil {
		t.Fatalf("unexepected error in AddAll() %s", err)
	}
	err = localRepository.Commit("build: add test file")
	if err != nil {
		t.Fatalf("unexpected error in Commit() %s", err)
	}
	err = localRepository.Push(branchName)
	if err != nil {
		t.Fatalf("unexpected error in Push() %s", err)
	}

	cleanupBranch := func() {
		slog.Info("cleaning up created branch", "branch", branchName)
		err := localRepository.DeleteBranch(branchName)
		if err != nil {
			t.Fatalf("unexpected error in DeleteBranch() %s", err)
		}
	}
	defer cleanupBranch()

	// Create a pull request
	client := github.NewClient(testToken, repo)
	createdPullRequest, err := client.CreatePullRequest(t.Context(), repo, branchName, "main", "test: integration test", "do not merge")
	if err != nil {
		t.Fatalf("unexpected error in CreatePullRequest() %s", err)
	}
	t.Logf("created pull request: %d", createdPullRequest.Number)

	// Ensure we clean up the created PR. The pull request is closed later in the
	// test, but this should make sure unless ClosePullRequest() is not working.
	cleanupPR := func() {
		slog.Info("cleaning up opened pull request")
		client.ClosePullRequest(t.Context(), createdPullRequest.Number)
	}
	defer cleanupPR()

	// Add a label to the pull request
	labels := []string{"do not merge", "type: cleanup"}
	err = client.AddLabelsToIssue(t.Context(), repo, createdPullRequest.Number, labels)
	if err != nil {
		t.Fatalf("unexpected error in AddLabelsToIssue() %s", err)
	}

	// Get labels and verify
	foundLabels, err := client.GetLabels(t.Context(), createdPullRequest.Number)
	if err != nil {
		t.Fatalf("unexpected error in GetLabels() %s", err)
	}
	if diff := cmp.Diff(foundLabels, labels); diff != "" {
		t.Fatalf("GetLabels() mismatch (-want + got):\n%s", diff)
	}

	// Replace labels
	labels = []string{"foo", "bar"}
	err = client.ReplaceLabels(t.Context(), createdPullRequest.Number, labels)
	if err != nil {
		t.Fatalf("unexpected error in ReplaceLabels() %s", err)
	}

	// Get labels and verify
	foundLabels, err = client.GetLabels(t.Context(), createdPullRequest.Number)
	if err != nil {
		t.Fatalf("unexpected error in GetLabels() %s", err)
	}
	if diff := cmp.Diff(foundLabels, labels); diff != "" {
		t.Fatalf("GetLabels() mismatch (-want + got):\n%s", diff)
	}

	// Add label
	err = client.AddLabelsToIssue(t.Context(), repo, createdPullRequest.Number, []string{"librarian-test", "asdf"})
	if err != nil {
		t.Fatalf("unexpected error in AddLabelsToIssue() %s", err)
	}

	// Get labels and verify
	wantLabels := []string{"foo", "bar", "librarian-test", "asdf"}
	foundLabels, err = client.GetLabels(t.Context(), createdPullRequest.Number)
	if err != nil {
		t.Fatalf("unexpected error in GetLabels() %s", err)
	}
	if diff := cmp.Diff(foundLabels, wantLabels); diff != "" {
		t.Fatalf("GetLabels() mismatch (-want + got):\n%s", diff)
	}

	// Add a comment
	err = client.CreateIssueComment(t.Context(), createdPullRequest.Number, "some comment body")
	if err != nil {
		t.Fatalf("unexpected error in CreateIssueComment() %s", err)
	}

	// Search for pull requests (this may take a bit of time, so try 5 times)
	found := false
	for i := 0; i < 5; i++ {
		foundPullRequests, err := client.SearchPullRequests(t.Context(), "label:librarian-test is:open")
		if err != nil {
			t.Fatalf("unexpected error in SearchPullRequests() %s", err)
		}
		for _, pullRequest := range foundPullRequests {
			// Look for the PR we created
			if pullRequest.GetNumber() == createdPullRequest.Number {
				found = true

				// Expect that we found a comment
				if pullRequest.GetComments() == 0 {
					t.Fatalf("Expected to have created a comment on the pull request.")
				}
				break
			}
		}
		if found {
			break
		}
		delay := time.Duration(2 * time.Second)
		t.Logf("Retrying in %v...\n", delay)
		time.Sleep(delay)
	}
	if !found {
		t.Fatalf("failed to find pull request after 5 attempts")
	}

	// Close pull request
	err = client.ClosePullRequest(t.Context(), createdPullRequest.Number)
	if err != nil {
		t.Fatalf("unexpected error in ClosePullRequest() %s", err)
	}

	// Get single pull request
	foundPullRequest, err := client.GetPullRequest(t.Context(), createdPullRequest.Number)
	if err != nil {
		t.Fatalf("unexpected error in GetPullRequest() %s", err)
	}
	if diff := cmp.Diff(foundPullRequest.GetNumber(), createdPullRequest.Number); diff != "" {
		t.Fatalf("pull request number mismatch (-want + got):\n%s", diff)
	}
	if diff := cmp.Diff(foundPullRequest.GetState(), "closed"); diff != "" {
		t.Fatalf("pull request state mismatch (-want + got):\n%s", diff)
	}
}

func TestFindMergedPullRequest(t *testing.T) {
	if testToken == "" {
		t.Skip("TEST_GITHUB_TOKEN not set, skipping GitHub integration test")
	}
	repoName := "https://github.com/googleapis/librarian"
	repo, err := github.ParseRemote(repoName)
	if err != nil {
		t.Fatalf("unexpected error in ParseRemote() %s", err)
	}

	for _, test := range []struct {
		name          string
		label         string
		want          int
		wantErr       bool
		wantErrSubstr string
	}{
		{
			name:    "existing label",
			label:   "cla: yes",
			want:    2210,
			wantErr: false,
		},
		{
			name:    "non-existing label",
			label:   "non-existing-label",
			want:    0,
			wantErr: false,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			client := github.NewClient(testToken, repo)
			prs, err := client.FindMergedPullRequestsWithLabel(t.Context(), repo.Owner, repo.Name, test.label)
			if test.wantErr {
				if err == nil {
					t.Fatalf("FindMergedPullRequestsWithLabel() err = nil, want error containing %q", test.wantErrSubstr)
				} else if !strings.Contains(err.Error(), test.wantErrSubstr) {
					t.Errorf("FindMergedPullRequestsWithLabel() err = %v, want error containing %q", err, test.wantErrSubstr)
				}
				return
			}
			if err != nil {
				t.Errorf("FindMergedPullRequestsWithLabel() err = %v, want nil", err)
			}
			if test.want == 0 {
				// expect to not find any
				if len(prs) > 0 {
					t.Fatalf("FindMergedPullRequestWithLabel() expected to not find any PRs, found %d", len(prs))
				}
			} else {
				found := false
				for _, pr := range prs {
					pullNumber := pr.GetNumber()
					t.Logf("Found PR %d", pullNumber)
					if pr.Number != nil && pullNumber == test.want {
						found = true
					}
				}
				if !found {
					t.Fatalf("FindMergedPullRequestsWithLabel() expected to find PR #%d", test.want)
				}
			}
		})
	}
}

func TestCreateTag(t *testing.T) {
	if testToken == "" {
		t.Skip("TEST_GITHUB_TOKEN not set, skipping GitHub integration test")
	}
	testRepoName := os.Getenv("TEST_GITHUB_REPO")
	if testRepoName == "" {
		t.Skip("TEST_GITHUB_REPO not set, skipping GitHub integration test")
	}
	repo, err := github.ParseRemote(testRepoName)
	if err != nil {
		t.Fatalf("unexpected error in ParseRemote() %s", err)
	}
	// Clone a repo
	workdir := path.Join(t.TempDir(), "test-repo")
	localRepository, err := gitrepo.NewRepository(&gitrepo.RepositoryOptions{
		Dir:          workdir,
		MaybeClone:   true,
		RemoteURL:    testRepoName,
		RemoteBranch: "main",
		GitPassword:  testToken,
		Depth:        1,
	})
	if err != nil {
		t.Fatalf("unexpected error in NewRepository() %s", err)
	}

	now := time.Now()
	tagName := fmt.Sprintf("create-tag-test-%s", now.Format("20060102150405"))
	commitSHA, err := localRepository.HeadHash()
	if err != nil {
		t.Fatalf("unexpected error fetching head commit %s", err)
	}

	client := github.NewClient(testToken, repo)
	err = client.CreateTag(t.Context(), tagName, commitSHA)
	if err != nil {
		t.Fatalf("unexpected error in CreateTag() %s", err)
	}
}

func TestCreateRelease(t *testing.T) {
	if testToken == "" {
		t.Skip("TEST_GITHUB_TOKEN not set, skipping GitHub integration test")
	}
	testRepoName := os.Getenv("TEST_GITHUB_REPO")
	if testRepoName == "" {
		t.Skip("TEST_GITHUB_REPO not set, skipping GitHub integration test")
	}
	repo, err := github.ParseRemote(testRepoName)
	if err != nil {
		t.Fatalf("unexpected error in ParseRemote() %s", err)
	}
	// Clone a repo
	workdir := path.Join(t.TempDir(), "test-repo")
	localRepository, err := gitrepo.NewRepository(&gitrepo.RepositoryOptions{
		Dir:          workdir,
		MaybeClone:   true,
		RemoteURL:    testRepoName,
		RemoteBranch: "main",
		GitPassword:  testToken,
		Depth:        1,
	})
	if err != nil {
		t.Fatalf("unexpected error in NewRepository() %s", err)
	}

	now := time.Now()
	tagName := fmt.Sprintf("create-release-test-%s", now.Format("20060102150405"))
	commitSHA, err := localRepository.HeadHash()
	if err != nil {
		t.Fatalf("unexpected error fetching head commit %s", err)
	}

	client := github.NewClient(testToken, repo)
	body := "some release body"
	release, err := client.CreateRelease(t.Context(), tagName, tagName, body, commitSHA)
	if err != nil {
		t.Fatalf("unexpected error in CreateTag() %s", err)
	}
	if diff := cmp.Diff(release.GetBody(), body); diff != "" {
		t.Fatalf("release body mismatch (-want + got):\n%s", diff)
	}
}
