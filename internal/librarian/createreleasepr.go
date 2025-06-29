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
	"path/filepath"
	"strconv"
	"strings"

	"github.com/googleapis/librarian/internal/cli"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/github"

	"github.com/Masterminds/semver/v3"
	"github.com/googleapis/librarian/internal/statepb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const prNumberEnvVarName = "_PR_NUMBER"
const baselineCommitEnvVarName = "_BASELINE_COMMIT"

var cmdCreateReleasePR = &cli.Command{
	Short:     "create-release-pr creates a release PR",
	UsageLine: "librarian create-release-pr -language=<language> [flags]",
	Long: `Specify the language, and optional flags to use non-default repositories, e.g. for testing.
A single library may be specified if desired (with an optional version override);
otherwise all configured libraries will be checked to see if they should be released.
A pull request will only be created if -push is specified, in which case the LIBRARIAN_GITHUB_TOKEN
environment variable must be populated with an access token which has write access to the
language repo in which the pull request will be created.

After acquiring the language repository, the Librarian state for the language
repository is loaded and each configured library is checked to see if it should be released.
(If the -library-id flag is specified, only the specified library is considered.)

Each library is considered separately for release. Libraries which have releases blocked
are skipped. Otherwise, the commits in the repository are considered backwards from the head
to the tagged commit for the current release version of the library (if any). Each commit is checked
to see if it affects any of the source paths specified for the library. Each commit which *does*
affect one of the source paths for the library is then checked for lines representing conventional commit messages.
(See https://www.conventionalcommits.org/ for details on conventional commits.)

Conventional commit messages with types of "doc" (or "docs"), "feat" and "fix" are retained.
Additionally, lines beginning "TriggerRelease:" and "NoTriggerRelease:" are retained as further signals for releasing.

Any individual commit will trigger a release for a library if either:
- It contains a TriggerRelease line specifying the library being considered
- It contains a "feat" or "fix" message

If the commit contains a NoTriggerRelease line specifying the library being considered, that stops
that specific commit from triggering a release, but other commits may still trigger a release (and
messages within the "NoTriggerRelease" commit are still used for release notes).

After examining the commits, if none of them triggers a release for the library, the command proceeds to
the next library - unless the library has been specified with the -library-id flag, in which case the flag
is considered an override, and the library will still be released. If the library *is* to be released:

- The next version is determined based on the current version, the commit messages, any configured "next version"
  in the Librarian state.
- Release notes are formatted based on the commit messages
- The Librarian state is updated with the new version
- The following language container commands are run:
  - "prepare-library-release" (used to update any other occurrences of the version number and version history files)
  - "build-library"
  - "integration-test-library"

A commit is then created, including metadata in the commit message to indicate which library is being released,
at which version, and for which release ID. (A single timestamp-based release ID is used for all commits for a single
run of the create-release-pr command.)

If any container command fails, the error is reported, and the repository is reset as
if the release hadn't been triggered for that library.

After iterating across all libraries, if the -push flag has been specified and a release commit
was successfully created for any library, a pull request is created in the
language repository, containing the release commits. The pull request description
includes an overview list of what's in each commit, along with any failures in other
libraries, and the release ID. (The details of the failures are not included; consult the logs for
the command to see exactly what happened.) The pull request has the "do not merge" label added,
which the "merge-release-pr" command is expected to remove before merging.

If the -push flag has not been specified but a pull request would have been created,
the description of the pull request that would have been created is included in the
output of the command. Even if a pull request isn't created, any successful release
commits will still be present in the language repo.
`,
	Run: runCreateReleasePR,
}

func init() {
	cmdCreateReleasePR.Init()
	fs := cmdCreateReleasePR.Flags
	cfg := cmdCreateReleasePR.Config

	addFlagImage(fs, cfg)
	addFlagSecretsProject(fs, cfg)
	addFlagWorkRoot(fs, cfg)
	addFlagLanguage(fs, cfg)
	addFlagLibraryID(fs, cfg)
	addFlagLibraryVersion(fs, cfg)
	addFlagPush(fs, cfg)
	addFlagGitUserEmail(fs, cfg)
	addFlagGitUserName(fs, cfg)
	addFlagRepoRoot(fs, cfg)
	addFlagSkipIntegrationTests(fs, cfg)
	addFlagEnvFile(fs, cfg)
	addFlagRepoUrl(fs, cfg)
}

func runCreateReleasePR(ctx context.Context, cfg *config.Config) error {
	state, err := createCommandStateForLanguage(cfg.WorkRoot, cfg.RepoRoot, cfg.RepoURL, cfg.Language,
		cfg.Image, cfg.LibrarianRepository, cfg.SecretsProject, cfg.CI, cfg.UserUID, cfg.UserGID)
	if err != nil {
		return err
	}
	return createReleasePR(ctx, state, cfg)
}

func createReleasePR(ctx context.Context, state *commandState, cfg *config.Config) error {
	if err := validateSkipIntegrationTests(cfg.SkipIntegrationTests); err != nil {
		return err
	}

	if cfg.LibraryVersion != "" && cfg.LibraryID == "" {
		return fmt.Errorf("flag -library-version is not valid without -library-id")
	}

	if cfg.LibraryID != "" && findLibraryByID(state.pipelineState, cfg.LibraryID) == nil {
		return fmt.Errorf("no such library: %s", cfg.LibraryID)
	}

	inputDirectory := filepath.Join(state.workRoot, "inputs")
	if err := os.Mkdir(inputDirectory, 0755); err != nil {
		slog.Error("Failed to create input directory")
		return err
	}

	// Find the head of the language repo before we start creating any release commits.
	// This will be validated later to check that libraries haven't changed since the release PR was created.
	baselineCommit, err := state.languageRepo.HeadHash()
	if err != nil {
		return err
	}
	if err := appendResultEnvironmentVariable(state.workRoot, baselineCommitEnvVarName, baselineCommit, cfg.EnvFile); err != nil {
		return err
	}

	releaseID := fmt.Sprintf("release-%s", formatTimestamp(state.startTime))
	if err := appendResultEnvironmentVariable(state.workRoot, releaseIDEnvVarName, releaseID, cfg.EnvFile); err != nil {
		return err
	}

	prContent, err := generateReleaseCommitForEachLibrary(ctx, state, cfg, inputDirectory, releaseID)
	if err != nil {
		return err
	}

	prMetadata, err := createPullRequest(ctx, state, prContent, "chore: Library release", fmt.Sprintf("Librarian-Release-ID: %s", releaseID), "release", cfg.GitHubToken, cfg.Push)
	if err != nil {
		return err
	}

	if prMetadata == nil {
		// We haven't created a release PR, and there are no errors. This could be because:
		// - There are no changes to release
		// - The -push flag wasn't specified.
		// Either way, complete successfully at this point.
		return nil
	}

	// Final steps if we've actually created a release PR.
	// - We always add the do-not-merge label so that Librarian can merge later.
	// - Add a result environment variable with the PR number, for the next stage of the process.
	ghClient, err := github.NewClient(cfg.GitHubToken)
	if err != nil {
		return err
	}
	err = ghClient.AddLabelToPullRequest(ctx, prMetadata, DoNotMergeLabel)
	if err != nil {
		slog.Warn("Received error trying to add label to PR", "err", err)
		return err
	}
	if err := appendResultEnvironmentVariable(state.workRoot, prNumberEnvVarName, strconv.Itoa(prMetadata.Number), cfg.EnvFile); err != nil {
		return err
	}
	return nil
}

// Iterate over all configured libraries, and check for new commits since the previous release tag for that library.
// The error handling here takes one of two forms:
//   - Library-level errors do not halt the process, but are reported in the resulting PR (if any).
//     This can include tags being missing, release preparation failing, or the build failing.
//   - More fundamental errors (e.g. a failure to commit, or to save pipeline state) abort the whole process immediately.
func generateReleaseCommitForEachLibrary(ctx context.Context, state *commandState, cfg *config.Config, inputDirectory, releaseID string) (*PullRequestContent, error) {
	cc := state.containerConfig
	libraries := state.pipelineState.Libraries
	languageRepo := state.languageRepo

	pr := new(PullRequestContent)

	for _, library := range libraries {
		// If we've specified a single library to release, skip all the others.
		if cfg.LibraryID != "" && library.Id != cfg.LibraryID {
			continue
		}
		if library.ReleaseAutomationLevel == statepb.AutomationLevel_AUTOMATION_LEVEL_BLOCKED {
			slog.Info("Skipping release-blocked library", "id", library.Id)
			continue
		}
		var commitMessages []*CommitMessage
		var previousReleaseTag string
		if library.CurrentVersion == "" {
			previousReleaseTag = ""
		} else {
			previousReleaseTag = formatReleaseTag(library.Id, library.CurrentVersion)
		}
		allSourcePaths := append(state.pipelineState.CommonLibrarySourcePaths, library.SourcePaths...)
		commits, err := languageRepo.GetCommitsForPathsSinceTag(allSourcePaths, previousReleaseTag)
		if err != nil {
			addErrorToPullRequest(pr, library.Id, err, "retrieving commits since last release")
			continue
		}

		for _, commit := range commits {
			commitMessages = append(commitMessages, ParseCommit(commit))
		}

		// If nothing release-worthy has happened, just continue to the next library.
		// (But if we've been asked to release a specific library, we force-release it anyway.)
		if cfg.LibraryID == "" && (len(commitMessages) == 0 || !isReleaseWorthy(commitMessages, library.Id)) {
			continue
		}

		releaseVersion, err := calculateNextVersion(library, cfg.LibraryVersion)
		if err != nil {
			return nil, err
		}

		releaseNotes := formatReleaseNotes(commitMessages)
		if err = createReleaseNotesFile(inputDirectory, library.Id, releaseVersion, releaseNotes); err != nil {
			return nil, err
		}

		// Update the pipeline state to record what we're releasing and when, and to clear the next version field.
		// Performing this before anything else means that container code can use the pipeline state for the steps
		// below, if it doesn't want/need to store the version separately.
		library.CurrentVersion = releaseVersion
		library.NextVersion = ""
		library.LastReleasedCommit = library.LastGeneratedCommit
		library.ReleaseTimestamp = timestamppb.Now()
		if err = savePipelineState(state); err != nil {
			return nil, err
		}

		if err := cc.PrepareLibraryRelease(ctx, cfg, languageRepo.Dir, inputDirectory, library.Id, releaseVersion); err != nil {
			addErrorToPullRequest(pr, library.Id, err, "preparing library release")
			// Clean up any changes before starting the next iteration.
			if err := languageRepo.CleanWorkingTree(); err != nil {
				return nil, err
			}
			continue
		}
		if err := cc.BuildLibrary(ctx, cfg, languageRepo.Dir, library.Id); err != nil {
			addErrorToPullRequest(pr, library.Id, err, "building/testing library")
			// Clean up any changes before starting the next iteration.
			if err := languageRepo.CleanWorkingTree(); err != nil {
				return nil, err
			}
			continue
		}
		if cfg.SkipIntegrationTests != "" {
			slog.Info("Skipping integration tests", "bug", cfg.SkipIntegrationTests)
		} else if err := cc.IntegrationTestLibrary(ctx, cfg, languageRepo.Dir, library.Id); err != nil {
			addErrorToPullRequest(pr, library.Id, err, "integration testing library")
			if err := languageRepo.CleanWorkingTree(); err != nil {
				return nil, err
			}
			continue
		}

		releaseDescription := fmt.Sprintf("chore: Release library %s version %s", library.Id, releaseVersion)
		addSuccessToPullRequest(pr, releaseDescription)
		// Metadata for easy extraction later.
		metadata := fmt.Sprintf("Librarian-Release-Library: %s\nLibrarian-Release-Version: %s\nLibrarian-Release-ID: %s", library.Id, releaseVersion, releaseID)
		// Note that releaseDescription will already end with two line breaks, so we don't need any more before the metadata.
		err = commitAll(languageRepo, fmt.Sprintf("%s\n\n%s%s", releaseDescription, releaseNotes, metadata),
			cfg.GitUserName, cfg.GitUserEmail)
		if err != nil {
			return nil, err
		}
	}
	return pr, nil
}

// TODO(https://github.com/googleapis/librarian/issues/564): decide on release notes ordering
func formatReleaseNotes(commitMessages []*CommitMessage) string {
	// Group release notes by type, preserving ordering (FIFO)
	var features, docs, fixes []string
	featuresSeen := make(map[string]bool)
	docsSeen := make(map[string]bool)
	fixesSeen := make(map[string]bool)

	// TODO(https://github.com/googleapis/librarian/issues/547): perhaps record breaking changes in a separate section
	// TODO(https://github.com/googleapis/librarian/issues/550): include backlinks, googleapis commits etc
	for _, commitMessage := range commitMessages {
		for _, feature := range commitMessage.Features {
			if !featuresSeen[feature] {
				featuresSeen[feature] = true
				features = append(features, feature)
			}
		}
		for _, doc := range commitMessage.Docs {
			if !docsSeen[doc] {
				docsSeen[doc] = true
				docs = append(docs, doc)
			}
		}
		for _, fix := range commitMessage.Fixes {
			if !fixesSeen[fix] {
				fixesSeen[fix] = true
				fixes = append(fixes, fix)
			}
		}
	}

	var builder strings.Builder

	maybeAppendReleaseNotesSection(&builder, "New features", features)
	maybeAppendReleaseNotesSection(&builder, "Bug fixes", fixes)
	maybeAppendReleaseNotesSection(&builder, "Documentation improvements", docs)

	if builder.Len() == 0 {
		builder.WriteString("FIXME: Forced release with no commit messages; please write release notes.\n\n")
	}
	return builder.String()
}

func createReleaseNotesFile(inputDirectory, libraryId, releaseVersion, releaseNotes string) error {
	path := filepath.Join(inputDirectory, fmt.Sprintf("%s-%s-release-notes.txt", libraryId, releaseVersion))
	return createAndWriteToFile(path, releaseNotes)
}

func maybeAppendReleaseNotesSection(builder *strings.Builder, description string, lines []string) {
	if len(lines) == 0 {
		return
	}
	fmt.Fprintf(builder, "### %s\n\n", description)
	for _, line := range lines {
		if len(line) > 1 {
			// This assumes the first character is ASCII, but that's reasonable for all our
			// actual use cases.
			line = strings.ToUpper(line[0:1]) + line[1:]
		}
		fmt.Fprintf(builder, "- %s\n", line)
	}
	builder.WriteString("\n")
}

func calculateNextVersion(library *statepb.LibraryState, libraryVersion string) (string, error) {
	if libraryVersion != "" {
		return libraryVersion, nil
	}
	if library.NextVersion != "" {
		return library.NextVersion, nil
	}
	if library.CurrentVersion == "" {
		return "", fmt.Errorf("cannot determine new version for %s; no current version", library.Id)
	}
	current, err := semver.StrictNewVersion(library.CurrentVersion)
	if err != nil {
		return "", err
	}
	var next *semver.Version
	prerelease := current.Prerelease()
	if prerelease != "" {
		nextPrerelease, err := calculateNextPrerelease(prerelease)
		if err != nil {
			return "", err
		}
		next = semver.New(current.Major(), current.Minor(), current.Patch(), nextPrerelease, "")
	} else {
		next = semver.New(current.Major(), current.Minor()+1, current.Patch(), "", "")
	}
	return next.String(), nil
}

// Match trailing digits in the prerelease part, then parse those digits as an integer.
// Increment the integer, then format it again - keeping as much of the existing prerelease as is
// required to end up with a string longer-than-or-equal to the original.
// If there are no trailing digits, fail.
// Note: this assumes the prerelease is purely ASCII.
func calculateNextPrerelease(prerelease string) (string, error) {
	digits := 0
	for i := len(prerelease) - 1; i >= 0; i-- {
		c := prerelease[i]
		if c < '0' || c > '9' {
			break
		} else {
			digits++
		}
	}
	if digits == 0 {
		return "", fmt.Errorf("unable to create next prerelease from '%s'", prerelease)
	}
	currentSuffix := prerelease[len(prerelease)-digits:]
	currentNumber, err := strconv.Atoi(currentSuffix)
	if err != nil {
		return "", err
	}
	nextSuffix := strconv.Itoa(currentNumber + 1)
	if len(nextSuffix) < len(currentSuffix) {
		nextSuffix = strings.Repeat("0", len(currentSuffix)-len(nextSuffix)) + nextSuffix
	}
	return prerelease[:(len(prerelease)-digits)] + nextSuffix, nil
}

func isReleaseWorthy(messages []*CommitMessage, libraryId string) bool {
	for _, message := range messages {
		if IsReleaseWorthy(message, libraryId) {
			return true
		}
	}
	return false
}
