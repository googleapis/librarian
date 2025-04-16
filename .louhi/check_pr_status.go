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
	"log"
	"log/slog"
	"os"
	"time"

	"github.com/google/go-github/v69/github"
)

const (
	pollInterval = 60 * time.Second
)

// checkPRStatus checks the status of a PR until it is merged or mergeable.
// Sleeping for [pollInterval] seconds between checks.
func checkPRStatus(prNumber int, repoOwner string, repoName string, statusCheck string) {

	ctx := context.Background()
	client := github.NewClient(nil)
	for {
		slog.Info("Checking", "status", statusCheck, "pr", prNumber, "owner", repoOwner, "repo", repoName)
		pr, resp, err := client.PullRequests.Get(ctx, repoOwner, repoName, prNumber)
		if err != nil {
			slog.Error("Error getting PR", "error", err, "response", resp)
			time.Sleep(pollInterval)
			continue
		}

		if statusCheck == "merged" {
			if pr.GetMerged() {
				slog.Info("PR is merged")
				return
			} else {
				slog.Info("PR not merged, will try again", "merge status", pr.GetMerged())
				time.Sleep(pollInterval)
			}
		} else if statusCheck == "mergeable" {
			if pr.GetMergeable() || pr.GetMerged() {
				slog.Info("PR is mergable or already merged", "mergeable status", pr.GetMergeable(), "merged status", pr.GetMerged())
				return
			} else {
				slog.Info("PR is not mergable, will try again", "mergeable status", pr.GetMergeable(), "merged status", pr.GetMerged())
				time.Sleep(pollInterval)
			}
		} else if statusCheck == "approved" {
			if checkIfPrIsApproved(client, ctx, repoOwner, repoName, prNumber) {
				slog.Info("PR is approved")
				return
			} else {
				slog.Info("PR is not approved, will try again")
				time.Sleep(pollInterval)
			}
		}

	}
}

func checkIfPrIsApproved(client *github.Client, ctx context.Context, owner string, repo string, prNumber int) bool {
	opt := &github.ListOptions{PerPage: 100}
	var allReviews []*github.PullRequestReview
	for {
		reviews, resp, err := client.PullRequests.ListReviews(ctx, owner, repo, prNumber, opt)
		if err != nil {
			log.Fatalf("Error listing reviews: %v", err)
			os.Exit(1)
		}
		allReviews = append(allReviews, reviews...)
		if resp.NextPage == 0 {
			break
		}
		opt.Page = resp.NextPage
	}

	// Check if any review is in the "APPROVED" state
	isApproved := false
	latestReviews := make(map[int64]*github.PullRequestReview) // Store latest review per user

	for _, review := range allReviews {
		// Ignore PENDING reviews as they haven't been submitted
		if review.GetState() == "PENDING" {
			continue
		}

		// Track the latest review submitted by each user
		userID := review.GetUser().GetID()
		if current, exists := latestReviews[userID]; !exists || review.GetSubmittedAt().After(current.GetSubmittedAt().Time) {
			latestReviews[userID] = review
		}
	}

	// Now check the state of the latest review for each user
	for _, review := range latestReviews {
		// Consider it approved if the latest state from *any* relevant reviewer is APPROVED.
		// Note: You might need more complex logic depending on branch protection rules
		// (e.g., requiring specific number of approvals, handling DISMISSED states more carefully,
		// checking if reviewers are code owners, etc.)
		if review.GetState() == "APPROVED" {
			isApproved = true
			break // Found at least one approval
		}
	}
	return isApproved
}

func main() {
	// Define command-line flags
	prNumber := flag.Int("pr-number", 0, "PR number to check if mergable (required)")
	repo := flag.String("repo", "", "GitHub repository name(required)")
	owner := flag.String("owner", "", "GitHub owner name (required)")
	statusCheck := flag.String("status-check", "", "Type of status check: 'merged' or 'mergeable' (required)")

	flag.Parse()

	if *prNumber == 0 || *repo == "" || *owner == "" || *statusCheck == "" {
		fmt.Println("Usage: go run main.go -pr-number <pr number to check> -repo <repo> -owner <owner> -status-check <merged|mergeable>")
		flag.PrintDefaults()
		os.Exit(1)
	}

	if (*statusCheck != "merged") && (*statusCheck != "mergeable") && (*statusCheck != "approved") {
		slog.Error("Invalid status check type", "type", statusCheck)
		os.Exit(1)
	}

	checkPRStatus(*prNumber, *owner, *repo, *statusCheck)
	//if it gets here it means the PR is merged or mergeable
	os.Exit(0)
}
