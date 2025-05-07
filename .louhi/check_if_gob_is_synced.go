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
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

const (
	pollInterval = 60 * time.Second
)

func main() {
	gerritRepoURL := flag.String("repo-url", "", "Gerrit repo to check for commit (required)")
	gerritAuthToken := flag.String("auth-token", "", "Authorization token for gerrit repo (required)")
	commitHash := flag.String("commit-hash", "", "Commit hash to check for (required)")

	flag.Parse()

	if *gerritRepoURL == "" || *gerritAuthToken == "" || *commitHash == "" {
		fmt.Println("Usage: go run check_if_gob_is_synced.go -repo-url <Gerrit repo to check for commit> -auth-token <Authorization token for gerrit repo> -commit-hash <Commit hash to check for>")
		flag.PrintDefaults()
		os.Exit(1)
	}

	// Check if the commit exists in the Gerrit repository, if not and no error, sleep for pollInterval
	// and check again.
	for {
		exists, err := checkCommitExistsInGerrit(*gerritRepoURL, *gerritAuthToken, *commitHash)
		if err != nil {
			fmt.Printf("Error checking commit in Gerrit: %v\n", err)
			os.Exit(1)
		}

		if exists {
			fmt.Printf("Commit '%s' exists in the Gerrit repository.\n", *commitHash)
			os.Exit(0)
		} else {
			fmt.Printf("Commit '%s' does NOT exist in the Gerrit repository. Sleeping for %f seconds\n", *commitHash, pollInterval.Seconds())
			time.Sleep(pollInterval)
		}
	}
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
