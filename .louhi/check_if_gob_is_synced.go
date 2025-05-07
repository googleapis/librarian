// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/google/go-github/v69/github"
)

const (
	pollInterval = 60 * time.Second
)

func main() {
	gerritRepoURL := flag.String("gerrit-repo-url", "", "Gerrit repo to check for commit (required)")
	gerritAuthToken := flag.String("auth-token", "", "Authorization token for gerrit repo (required)")
	prNumber := flag.Int("pr-number", 0, "PR number to check if merged in gerrit (required)")
	gitRepo := flag.String("git-repo", "", "Git repo for PR (required)")
	gitOwner := flag.String("git-owner", "", "Git owner for PR (required)")

	flag.Parse()

	if *gerritRepoURL == "" || *gerritAuthToken == "" || *prNumber == 0 || *gitRepo == "" || *gitOwner == "" {
		fmt.Println("Usage: go run check_if_gob_is_synced.go -gerrit-repo-url <Gerrit repo to check for commit> -auth-token <Authorization token for gerrit repo> -pr-number <pr to check if synced to gerrit> -git-repo <GitHub repo pr belongs to> -git-owner <GitHub owner>")
		flag.PrintDefaults()
		os.Exit(1)
	}

	commitHash, err := getMergedPRCommitSHA(*prNumber, *gitOwner, *gitRepo)

	if err != nil {
		fmt.Printf("Error getting commit hash for PR %d: %v\n", *prNumber, err)
		os.Exit(1)
	}
	// Check if the commit exists in the Gerrit repository, if not and no error, sleep for pollInterval
	// and check again.
	for {
		exists, err := checkCommitExistsInGerrit(*gerritRepoURL, *gerritAuthToken, commitHash)
		if err != nil {
			fmt.Printf("Error checking commit in Gerrit: %v\n", err)
			os.Exit(1)
		}

		if exists {
			fmt.Printf("Commit '%s' exists in the Gerrit repository.\n", commitHash)
			os.Exit(0)
		} else {
			fmt.Printf("Commit '%s' does NOT exist in the Gerrit repository. Sleeping for %f seconds\n", commitHash, pollInterval.Seconds())
			time.Sleep(pollInterval)
		}
	}
}

// looks up commit hash for a given PR number in a GitHub repository.
func getMergedPRCommitSHA(prNumber int, repoOwner string, repoName string) (string, error) {
	ctx := context.Background()
	client := github.NewClient(nil)

	pr, resp, err := client.PullRequests.Get(ctx, repoOwner, repoName, prNumber)

	if err != nil {
		slog.Error("Error getting PR", "error", err, "response", resp)
		return "", fmt.Errorf("error getting PR: %v", err)
	}

	if pr.GetMerged() == false {
		slog.Info("PR requested is currently not merged", "merge status", pr.GetMerged())
		return "", fmt.Errorf("PR not merged")
	}
	return pr.GetMergeCommitSHA(), nil
}

// checkCommitExistsInGerrit uses the Gerrit API to check if a commit exists.
func checkCommitExistsInGerrit(repoUrl string, authToken string, commitHash string) (bool, error) {
	url := fmt.Sprintf("%s/+/%s", repoUrl, commitHash)
	fmt.Printf("Checking Gerrit URL: %s\n", url)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return false, fmt.Errorf("error creating HTTP request: %v", err)
	}

	req.Header.Add("Authorization", "Bearer "+authToken)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return false, fmt.Errorf("error making HTTP request to Gerrit: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		return true, nil
	} else if resp.StatusCode == http.StatusNotFound {
		return false, nil
	} else {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return false, fmt.Errorf("Gerrit API returned unexpected status: %d - %s", resp.StatusCode, string(bodyBytes))
	}
}
