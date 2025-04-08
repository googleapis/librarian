package main

import (
	"context"
	"flag"
	"fmt"
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
			if pr.GetMergeable() {
				slog.Info("PR is mergable")
				return
			} else {
				slog.Info("PR is not mergable, will try again", "mergeable status", pr.GetMerged())
				time.Sleep(pollInterval)
			}
		}
	}
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

	if (*statusCheck != "merged") && (*statusCheck != "mergeable") {
		slog.Error("Invalid status check type", "type", statusCheck)
		os.Exit(1)
	}

	checkPRStatus(*prNumber, *owner, *repo, *statusCheck)
	//if it gets here it means the PR is merged or mergeable
	os.Exit(0)
}
