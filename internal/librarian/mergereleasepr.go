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
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/go-github/v69/github"
	"github.com/googleapis/librarian/internal/cli"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/githubrepo"
	"github.com/googleapis/librarian/internal/statepb"
)

const (
	// DoNotMergeLabel uses to avoid users merging the PR themselves.
	DoNotMergeLabel = "do-not-merge"

	// DoNotMergeAppId is a GitHub App that skips do-not-merge check.
	DoNotMergeAppId = 91138

	// ConventionalCommitsAppId is a GitHub App that skips conventional commits
	// check.
	ConventionalCommitsAppId = 37172

	// MergeBlockedLabel uses to indicate "I've noticed a problem with this PR;
	// I won't check it again until you've done something".
	MergeBlockedLabel = "merge-blocked-see-comments"
)

var cmdMergeReleasePR = &cli.Command{
	Short:     "merge-release-pr merges a validated release PR",
	UsageLine: "librarian merge-release-pr -release-id=<id> -release-pr-url=<url> -baseline-commit=<commit> [flags]",
	Long: `Specify a GitHub access token as an environment variable, the URL for a release PR, the baseline
commit of the repo when the release PR was being created, and the release ID.
An optional additional URL prefix can be specified in order to wait for a mirror to have synchronized before the
command completes.

This command does not clone any repository locally. Instead, it:
- Polls GitHub until all the conditions required to merge the release PR have been met
- Removes the "do-not-merge" label from the PR
- Merges the PR
- Waits for a mirror repository to be synchronized (if -sync-url-prefix has been specified) to
  include the merged PR

The following conditions are checked before merging:

- The PR must not have the "merge-blocked-see-comments" label
- The PR must not be merged
- The PR must not be closed
- The PR must be mergeable (i.e. not have merge conflicts)
- All required status checks other than the absence of a "do not merge" label must be successful
- The PR must be approved
- All commits in the PR must have the specified Librarian-Release-Id
- No commit in the PR must have a commit message line beginning with "FIXME"
- No source paths contributing to a library that will be released should have been modified
  since the commit at which the release PR was created. (This avoids race conditions between features
  and releases missing out release notes.)

If the PR is merged externally (e.g. by a human), this command fails. It's possible that the step can be skipped
and the release recovered, but this should be validated manually.

If the PR is closed without being merged, and stays closed for at least a minute, this command fails.
If it is closed and reopened within a minute, polling continues: this allows for accidental closures to be reverted,
and for status checks that are triggered by the PR being opened to be re-evaluated.

It's valid for a release PR to be fetched by a human, edited, and then repushed.
Potential reasons for doing this include:
- Modifying release notes (including fixing any "FIXME" lines)
- Rebasing the PR
- Removing a commit for a library that shouldn't be released

If a release PR is observed to be not-ready for a reason that should be addressed by a human (rather than just
waiting for statuses to change, for example) a comment is added to the PR and the "merge-blocked-see-comments" label
is added.
`,
	Run: runMergeReleasePR,
}

func init() {
	cmdMergeReleasePR.InitFlags()
	fs := cmdMergeReleasePR.Flags
	cfg := cmdMergeReleasePR.Config

	addFlagImage(fs, cfg)
	addFlagSecretsProject(fs, cfg)
	addFlagWorkRoot(fs, cfg)
	addFlagBaselineCommit(fs, cfg)
	addFlagReleaseID(fs, cfg)
	addFlagReleasePRUrl(fs, cfg)
	addFlagSyncUrlPrefix(fs, cfg)
	addFlagEnvFile(fs, cfg)
}

func runMergeReleasePR(ctx context.Context, cfg *config.Config) error {
	startTime := time.Now()
	workRoot, err := createWorkRoot(startTime, cfg.WorkRoot)
	if err != nil {
		return err
	}
	return mergeReleasePR(ctx, workRoot, cfg)
}

// A SuspectRelease is a library release which is probably invalid due
// to other changes (e.g. to the pipeline state for the library, or the source
// code) which have been committed since the release process was initiated.
type SuspectRelease struct {
	LibraryID string
	Reason    string
}

const mergedReleaseCommitEnvVarName = "_MERGED_RELEASE_COMMIT"

func mergeReleasePR(ctx context.Context, workRoot string, cfg *config.Config) error {
	if cfg.SyncURLPrefix != "" && cfg.SyncAuthToken == "" {
		return errors.New("-sync-url-prefix specified, but no sync auth token present")
	}
	if err := validateRequiredFlag("release-id", cfg.ReleaseID); err != nil {
		return err
	}
	if err := validateRequiredFlag("release-pr-url", cfg.ReleasePRURL); err != nil {
		return err
	}
	if err := validateRequiredFlag("baseline-commit", cfg.BaselineCommit); err != nil {
		return err
	}

	// We'll assume the PR URL is in the format https://github.com/{owner}/{name}/pulls/{pull-number}
	prRepo, err := githubrepo.ParseUrl(cfg.ReleasePRURL)
	if err != nil {
		return err
	}

	prNumber, err := parsePrNumberFromUrl(cfg.ReleasePRURL)
	if err != nil {
		return err
	}

	prMetadata := &githubrepo.PullRequestMetadata{Repo: prRepo, Number: prNumber}

	if err := waitForPullRequestReadiness(ctx, prMetadata, cfg); err != nil {
		return err
	}

	mergeCommit, err := mergePullRequest(ctx, prMetadata, cfg)
	if err != nil {
		return err
	}

	if err := appendResultEnvironmentVariable(workRoot, mergedReleaseCommitEnvVarName, mergeCommit, cfg.EnvFile); err != nil {
		return err
	}

	if err := waitForSync(mergeCommit, cfg); err != nil {
		return err
	}
	return nil
}

// TODO(https://github.com/googleapis/librarian/issues/544): make timing configurable?
const sleepDelay = time.Duration(60) * time.Second

func waitForPullRequestReadiness(ctx context.Context, prMetadata *githubrepo.PullRequestMetadata, cfg *config.Config) error {
	// TODO(https://github.com/googleapis/librarian/issues/543): time out here, or let Kokoro do so?

	for {
		ready, err := waitForPullRequestReadinessSingleIteration(ctx, prMetadata, cfg)
		if ready || err != nil {
			return err
		}
		slog.Info("Sleeping before next iteration")
		// TODO(https://github.com/googleapis/librarian/issues/544): make timing configurable?
		time.Sleep(sleepDelay)
	}
}

// A single iteration of the loop in waitForPullRequestReadiness,
// in a separate function to make it easy to indicate an "early out".
// Returns true for "PR is ready to merge", false for "keep polling".
// If this function returns false (with no error) the reason will have been logged.
// Checks performed:
// - The PR must not be merged
// - The PR must not have the label with the name specified in MergeBlockedLabel
// - The PR must have the label with the name specified in DoNotMergeLabel
// - The PR must be mergeable
// - All status checks must pass, other than conventional commits and the do-not-merge check
// - All commits in the PR must contain Librarian-Release-Id (for this release), Librarian-Release-Library and Librarian-Release-Version metadata
// - No commit in the PR must start its release notes with "FIXME"
// - There must be no commits in the head of the repo which affect libraries released by the PR
// - There must be at least one approving reviews from a member/owner of the repo, and no reviews from members/owners requesting changes
func waitForPullRequestReadinessSingleIteration(ctx context.Context, prMetadata *githubrepo.PullRequestMetadata, cfg *config.Config) (bool, error) {
	slog.Info("Checking pull request for readiness")
	ghClient, err := githubrepo.NewClient(cfg.GitHubToken)
	if err != nil {
		return false, err
	}
	pr, err := ghClient.GetPullRequest(ctx, prMetadata.Repo, prMetadata.Number)
	if err != nil {
		return false, err
	}

	// If the PR has been merged by someone else, abort this command. (We can skip this step in the flow, if we still want to release.)
	if pr.GetMerged() {
		return false, errors.New("pull request already merged")
	}

	// If the PR is closed, wait a minute and check if it's *still* closed (to allow for deliberate "close/reopen" workflows),
	// and if it is, abort the job.
	if pr.ClosedAt != nil {
		slog.Info("PR is closed; sleeping for a minute before checking again.")
		time.Sleep(sleepDelay)
		pr, err = ghClient.GetPullRequest(ctx, prMetadata.Repo, prMetadata.Number)
		if err != nil {
			return false, err
		}
		if pr.ClosedAt != nil {
			slog.Info("PR is still closed; aborting.")
			return false, errors.New("pull request closed")
		}
		slog.Info("PR has been reopened. Continuing.")
	}

	// If we've already blocked this PR, and the user hasn't cleared the label yet, don't check anything else.
	gotDoNotMergeLabel := false
	for _, label := range pr.Labels {
		if label.GetName() == MergeBlockedLabel {
			slog.Info(fmt.Sprintf("PR still has '%s' label; skipping other checks", MergeBlockedLabel))
			return false, nil
		}
		if label.GetName() == DoNotMergeLabel {
			gotDoNotMergeLabel = true
		}
	}

	// We expect to remove the do-not-merge label ourselves (and we'll fail otherwise).
	if !gotDoNotMergeLabel {
		return false, reportBlockingReason(ctx, prMetadata, fmt.Sprintf("Label '%s' has been removed already", DoNotMergeLabel), cfg)
	}

	// If the PR isn't mergeable, that requires user action.
	if !pr.GetMergeable() {
		// This will log the reason.
		return false, reportBlockingReason(ctx, prMetadata, "PR is not mergeable (e.g. there are conflicting commit)", cfg)
	}

	// Check that all the statuses have passed.
	checkRuns, err := ghClient.GetPullRequestCheckRuns(ctx, pr)
	if err != nil {
		return false, err
	}
	for _, checkRun := range checkRuns {
		// Skip the do-not-merge check and conventional commits checks
		// (Once b/416489721 has been fixed, we can remove the conventional commits check)
		if checkRun.GetApp().GetID() == DoNotMergeAppId || checkRun.GetApp().GetID() == ConventionalCommitsAppId {
			continue
		}

		// For now, we assume that every other check must be complete and successful.
		// We can't get at the required status checks with the current access token;
		// we can rethink this if it turns out to be too conservative.
		if checkRun.GetStatus() != "completed" {
			slog.Info(fmt.Sprintf("Check '%s' is not complete", *checkRun.Name))
			return false, nil
		}
		if checkRun.GetConclusion() != "success" {
			return false, reportBlockingReason(ctx, prMetadata, fmt.Sprintf("Check '%s' failed", *checkRun.Name), cfg)
		}
	}

	// Check the commits in the pull request. If this returns false,
	// the reason will already be logged (so we don't need to log it again).
	commitStatus, err := checkPullRequestCommits(ctx, prMetadata, pr, cfg)
	if err != nil {
		return false, err
	}
	if !commitStatus {
		return false, err
	}

	// Check for approval
	approved, err := checkPullRequestApproval(ctx, prMetadata, cfg)
	if err != nil {
		return false, err
	}
	if !approved {
		slog.Info("PR not yet approved")
		return false, nil
	}

	slog.Info("All checks passed, ready to merge.")
	return true, nil
}

func mergePullRequest(ctx context.Context, prMetadata *githubrepo.PullRequestMetadata, cfg *config.Config) (string, error) {
	ghClient, err := githubrepo.NewClient(cfg.GitHubToken)
	if err != nil {
		return "", err
	}
	slog.Info("Merging release PR")
	if err := ghClient.RemoveLabelFromPullRequest(ctx, prMetadata.Repo, prMetadata.Number, "do-not-merge"); err != nil {
		return "", err
	}
	mergeResult, err := ghClient.MergePullRequest(ctx, prMetadata.Repo, prMetadata.Number, github.MergeMethodRebase)
	if err != nil {
		return "", err
	}

	slog.Info("Release PR merged")
	return *mergeResult.SHA, nil
}

// The maximum amount of time that waitForSync will poll to see if
// the merge commit has syncrhonized
// TODO(https://github.com/googleapis/librarian/issues/544): make timing configurable?
const waitForSyncMaxDuration = time.Duration(10) * time.Minute

// If flagSyncUrlPrefix is empty, this returns immediately.
// Otherwise, polls for up to 10 minutes (once every 30 seconds) for the
// given merge commit to be available at the repo specified via flagSyncUrlPrefix.
func waitForSync(mergeCommit string, cfg *config.Config) error {
	if cfg.SyncURLPrefix == "" {
		return nil
	}
	req, err := http.NewRequest("GET", cfg.SyncURLPrefix+mergeCommit, nil)
	if err != nil {
		return fmt.Errorf("error creating HTTP request: %v", err)
	}
	authToken := cfg.SyncAuthToken
	req.Header.Add("Authorization", "Bearer "+authToken)
	client := &http.Client{}

	// TODO(https://github.com/googleapis/librarian/issues/544): make timing configurable?
	end := time.Now().Add(waitForSyncMaxDuration)

	for time.Now().Before(end) {
		slog.Info("Checking if merge commit has synchronized")
		resp, err := client.Do(req)
		if err != nil {
			return err
		}
		// A status of OK means the commit has synced; we're done.
		// A status of NotFound means the commit hasn't *yet* synced; sleep and keep trying.
		// Any other status is unexpected, and we abort.
		if resp.StatusCode == http.StatusOK {
			slog.Info("Merge commit has synchronized")
			return nil
		} else if resp.StatusCode == http.StatusNotFound {
			slog.Info("Merge commit has not yet synchronized; sleeping before next attempt")
			// TODO(https://github.com/googleapis/librarian/issues/544): make timing configurable?
			time.Sleep(sleepDelay)
			continue
		} else {
			bodyBytes, _ := io.ReadAll(resp.Body)
			return fmt.Errorf("unexpected status fetching commit: %d - %s", resp.StatusCode, string(bodyBytes))
		}
	}
	return fmt.Errorf("timed out waiting for commit to sync")
}

// For each commit in the pull request, check:
// - We still have the Librarian metadata (release ID, library, version)
// - None of the paths which affect the library have been modified since the base of the PR
//
// Returns true if all the commits are fine, or false if a problem was detected, in which
// case it will have been reported on the PR, and the merge-blocking label applied.
func checkPullRequestCommits(ctx context.Context, prMetadata *githubrepo.PullRequestMetadata, pr *github.PullRequest, cfg *config.Config) (bool, error) {
	baseRepo := githubrepo.CreateGitHubRepoFromRepository(pr.Base.Repo)
	baseHeadState, err := fetchRemotePipelineState(ctx, baseRepo, *pr.Base.Ref, cfg.GitHubToken)
	if err != nil {
		return false, err
	}
	baselineState, err := fetchRemotePipelineState(ctx, baseRepo, cfg.BaselineCommit, cfg.GitHubToken)
	if err != nil {
		return false, err
	}

	// Fetch the commits which are in the PR, compared with the base (the target of the merge).
	// In most cases pr.Base.SHA will be the same as cfg.BaselineCommit, but the PR may have been rebased -
	// and we always only want the commits in the PR, not any that it's been rebased on top of.
	ghClient, err := githubrepo.NewClient(cfg.GitHubToken)
	if err != nil {
		return false, err
	}
	prCommits, err := ghClient.GetDiffCommits(ctx, prMetadata.Repo, *pr.Base.SHA, *pr.Head.SHA)
	if err != nil {
		return false, err
	}

	releases, err := parseRemoteCommitsForReleases(prCommits, cfg.ReleaseID)
	if err != nil {
		// This indicates that at least one commit is invalid - either it has missing
		// metadata, or it's for the wrong release. Report that reason, then return
		// a non-error from this function (we don't want to abort the process here).
		if err := reportBlockingReason(ctx, prMetadata, err.Error(), cfg); err != nil {
			return false, err
		}

		return false, nil
	}

	for _, release := range releases {
		if strings.HasPrefix(release.ReleaseNotes, "FIXME") {
			return false, reportBlockingReason(ctx, prMetadata, fmt.Sprintf("Release notes for '%s' need fixing", release.LibraryID), cfg)
		}
	}

	// Fetch the commits in the base repo since the baseline commit, but then fetch each individually
	// so we can tell which files were affected.
	baseCommits, err := ghClient.GetDiffCommits(ctx, baseRepo, cfg.BaselineCommit, *pr.Base.Ref)
	if err != nil {
		return false, err
	}
	fullBaseCommits := []*github.RepositoryCommit{}
	for _, baseCommit := range baseCommits {
		fullCommit, err := ghClient.GetCommit(ctx, baseRepo, *baseCommit.SHA)
		if err != nil {
			return false, err
		}
		fullBaseCommits = append(fullBaseCommits, fullCommit)
	}

	suspectReleases := []SuspectRelease{}

	slog.Info(fmt.Sprintf("Checking %d commits against %d libraries for intervening changes", len(fullBaseCommits), len(releases)))
	for _, release := range releases {
		suspectRelease := checkRelease(release, baseHeadState, baselineState, fullBaseCommits)
		if suspectRelease != nil {
			suspectReleases = append(suspectReleases, *suspectRelease)
		}
	}

	if len(suspectReleases) == 0 {
		return true, nil
	}

	var builder strings.Builder
	builder.WriteString("At least one library being released may have changed since release PR creation:\n\n")
	for _, suspectRelease := range suspectReleases {
		builder.WriteString(fmt.Sprintf("%s: %s\n", suspectRelease.LibraryID, suspectRelease.Reason))
	}
	return false, reportBlockingReason(ctx, prMetadata, builder.String(), cfg)
}

// Checks that the pull request has at least one approved review, and no "changes requested" reviews.
func checkPullRequestApproval(ctx context.Context, prMetadata *githubrepo.PullRequestMetadata, cfg *config.Config) (bool, error) {
	ghClient, err := githubrepo.NewClient(cfg.GitHubToken)
	if err != nil {
		return false, err
	}
	reviews, err := ghClient.GetPullRequestReviews(ctx, prMetadata)
	if err != nil {
		return false, err
	}

	slog.Info(fmt.Sprintf("Considering %d reviews (including history)", len(reviews)))
	// Collect all latest non-pending reviews from members/owners of the repository.
	latestReviews := make(map[int64]*github.PullRequestReview)
	for _, review := range reviews {
		association := review.GetAuthorAssociation()
		// TODO(https://github.com/googleapis/librarian/issues/545): check the required approvals
		if association != "MEMBER" && association != "OWNER" && association != "COLLABORATOR" && association != "CONTRIBUTOR" {
			slog.Info(fmt.Sprintf("Ignoring review with author association '%s'", association))
			continue
		}

		if review.GetState() == "PENDING" {
			slog.Info("Ignoring pending review")
			continue
		}

		userID := review.GetUser().GetID()
		// Need to ensure review is the latest for the user
		if current, exists := latestReviews[userID]; !exists || review.GetSubmittedAt().After(current.GetSubmittedAt().Time) {
			latestReviews[userID] = review
		}
	}

	approved := false
	for _, review := range latestReviews {
		slog.Info(fmt.Sprintf("Review at %s: %s", review.GetSubmittedAt().Format(time.RFC3339), review.GetState()))
		if review.GetState() == "APPROVED" {
			approved = true
		} else if review.GetState() == "CHANGES_REQUESTED" {
			slog.Info("Changes requested by at least one member/owner review; treating as unapproved.")
			return false, nil
		}
	}
	return approved, nil
}

func reportBlockingReason(ctx context.Context, prMetadata *githubrepo.PullRequestMetadata, description string, cfg *config.Config) error {
	slog.Warn(fmt.Sprintf("Adding '%s' label to PR and a comment with a description of '%s'", MergeBlockedLabel, description))
	comment := fmt.Sprintf("%s\n\nAfter resolving the issue, please remove the '%s' label.", description, MergeBlockedLabel)
	ghClient, err := githubrepo.NewClient(cfg.GitHubToken)
	if err != nil {
		return err
	}
	if err := ghClient.AddCommentToPullRequest(ctx, prMetadata.Repo, prMetadata.Number, comment); err != nil {
		return err
	}
	if err := ghClient.AddLabelToPullRequest(ctx, prMetadata, MergeBlockedLabel); err != nil {
		return err
	}
	return nil
}

func checkRelease(release LibraryRelease, baseHeadState, baselineState *statepb.PipelineState, baseCommits []*github.RepositoryCommit) *SuspectRelease {
	baseHeadLibrary := findLibraryByID(baseHeadState, release.LibraryID)
	if baseHeadLibrary == nil {
		return &SuspectRelease{LibraryID: release.LibraryID, Reason: "Library does not exist in head pipeline state"}
	}
	baselineLibrary := findLibraryByID(baselineState, release.LibraryID)
	if baselineLibrary == nil {
		return &SuspectRelease{LibraryID: release.LibraryID, Reason: "Library does not exist in baseline commit pipeline state"}
	}
	// TODO(https://github.com/googleapis/librarian/issues/546): find a better way of comparing these.
	if baseHeadLibrary.String() != baselineLibrary.String() {
		return &SuspectRelease{LibraryID: release.LibraryID, Reason: "Pipeline state has changed between baseline and head"}
	}
	sourcePaths := append(baseHeadState.CommonLibrarySourcePaths, baseHeadLibrary.SourcePaths...)
	changeCommits := []string{}
	for _, commit := range baseCommits {
		if checkIfCommitAffectsAnySourcePath(commit, sourcePaths) {
			changeCommits = append(changeCommits, *commit.SHA)
		}
	}
	if len(changeCommits) > 0 {
		reason := fmt.Sprintf("Library source changed in intervening commits: %s", strings.Join(changeCommits, ", "))
		return &SuspectRelease{LibraryID: release.LibraryID, Reason: reason}
	}
	return nil
}

func checkIfCommitAffectsAnySourcePath(commit *github.RepositoryCommit, sourcePaths []string) bool {
	for _, commitFile := range commit.Files {
		changedPath := *commitFile.Filename
		for _, sourcePath := range sourcePaths {
			if changedPath == sourcePath || (strings.HasPrefix(changedPath, sourcePath) && strings.HasPrefix(changedPath, sourcePath+"/")) {
				return true
			}
		}
	}
	return false
}

func parseRemoteCommitsForReleases(commits []*github.RepositoryCommit, releaseID string) ([]LibraryRelease, error) {
	releases := []LibraryRelease{}
	for _, commit := range commits {
		release, err := parseCommitMessageForRelease(*commit.Commit.Message, *commit.SHA)
		if err != nil {
			return nil, err
		}
		if release.ReleaseID != releaseID {
			return nil, fmt.Errorf("while finding releases for release ID %s, found commit with release ID %s", releaseID, release.ReleaseID)
		}
		releases = append(releases, *release)
	}
	return releases, nil
}

func parsePrNumberFromUrl(url string) (int, error) {
	parts := strings.Split(url, "/")
	return strconv.Atoi(parts[len(parts)-1])
}
