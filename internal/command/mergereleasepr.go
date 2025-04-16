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

package command

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"strconv"
	"strings"

	"github.com/google/go-github/v69/github"
	"github.com/googleapis/librarian/internal/githubrepo"
	"github.com/googleapis/librarian/internal/gitrepo"
	"github.com/googleapis/librarian/internal/statepb"
	"google.golang.org/protobuf/encoding/protojson"
)

var CmdMergeReleasePR = &Command{
	Name:  "merge-release-pr",
	Short: "Merge a release PR after validating it",
	flagFunctions: []func(fs *flag.FlagSet){
		addFlagImage,
		addFlagSecretsProject,
		addFlagWorkRoot,
		addFlagBaselineCommit,
		addFlagReleaseID,
		addFlagReleasePRUrl,
	},
	maybeGetLanguageRepo: func(workRoot string) (*gitrepo.Repo, error) {
		return nil, nil
	},
	execute: mergeReleasePRImpl,
}

type SuspectRelease struct {
	LibraryID string
	Reason    string
}

const mergedReleaseCommitEnvVarName = "_MERGED_RELEASE_COMMIT"

func mergeReleasePRImpl(ctx *CommandContext) error {
	if githubrepo.GetAccessToken() == "" {
		return errors.New("no GitHub access token specified")
	}
	// We'll assume the PR URL is in the format https://github.com/{owner}/{name}/pulls/{pull-number}
	prRepo, err := githubrepo.ParseUrl(flagReleasePRUrl)
	if err != nil {
		return err
	}

	prNumber, err := parsePrNumberFromUrl(flagReleasePRUrl)
	if err != nil {
		return err
	}

	pr, err := githubrepo.GetPullRequest(ctx.ctx, prRepo, prNumber)
	if err != nil {
		return err
	}

	baseRepo := githubrepo.CreateGitHubRepoFromRepository(pr.Base.Repo)
	baseHeadState, err := fetchRemotePipelineState(ctx.ctx, baseRepo, *pr.Base.Ref)
	if err != nil {
		return err
	}
	baselineState, err := fetchRemotePipelineState(ctx.ctx, baseRepo, flagBaselineCommit)
	if err != nil {
		return err
	}

	prCommits, err := githubrepo.GetDiffCommits(ctx.ctx, prRepo, flagBaselineCommit, *pr.Head.SHA)
	if err != nil {
		return err
	}
	releases, err := parseRemoteCommitsForReleases(prCommits, flagReleaseID)
	if err != nil {
		return err
	}
	// Fetch the commits in the base repo since the baseline commit, but then fetch each individually
	// so we can tell which files were affected.
	baseCommits, err := githubrepo.GetDiffCommits(ctx.ctx, baseRepo, flagBaselineCommit, *pr.Base.Ref)
	if err != nil {
		return err
	}
	fullBaseCommits := []*github.RepositoryCommit{}
	for _, baseCommit := range baseCommits {
		fullCommit, err := githubrepo.GetCommit(ctx.ctx, baseRepo, *baseCommit.SHA)
		if err != nil {
			return err
		}
		fullBaseCommits = append(fullBaseCommits, fullCommit)
	}

	suspectReleases := []SuspectRelease{}

	for _, release := range releases {
		suspectRelease := checkRelease(release, baseHeadState, baselineState, fullBaseCommits)
		if suspectRelease != nil {
			suspectReleases = append(suspectReleases, *suspectRelease)
		}
	}

	if len(suspectReleases) > 0 {
		var builder strings.Builder
		builder.WriteString("At least one library being released may have changed since release PR creation:\n\n")
		for _, suspectRelease := range suspectReleases {
			builder.WriteString(fmt.Sprintf("%s: %s\n", suspectRelease.LibraryID, suspectRelease.Reason))
		}
		description := builder.String()
		if err := githubrepo.AddCommentToPullRequest(ctx.ctx, prRepo, prNumber, description); err != nil {
			return err
		}
		return errors.New("did not merge release PR due to suspected-changed libraries")
	} else {
		if err := githubrepo.RemoveLabelFromPullRequest(ctx.ctx, prRepo, prNumber, "do-not-merge"); err != nil {
			return err
		}
		mergeResult, err := githubrepo.MergePullRequest(ctx.ctx, prRepo, prNumber, github.MergeMethodRebase)
		if err != nil {
			return err
		}

		if err := appendResultEnvironmentVariable(ctx, mergedReleaseCommitEnvVarName, *mergeResult.SHA); err != nil {
			return err
		}
		return nil
	}
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
	// TODO: Find a better way of comparing these.
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

func fetchRemotePipelineState(ctx context.Context, repo githubrepo.GitHubRepo, ref string) (*statepb.PipelineState, error) {
	bytes, err := githubrepo.GetRawContent(ctx, repo, "generator-input/pipeline-state.json", ref)
	if err != nil {
		return nil, err
	}

	state := &statepb.PipelineState{}
	err = protojson.Unmarshal(bytes, state)
	if err != nil {
		return nil, err
	}
	return state, nil
}
