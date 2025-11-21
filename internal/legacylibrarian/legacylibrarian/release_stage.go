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

package legacylibrarian

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/googleapis/librarian/internal/legacylibrarian/legacyconfig"
	"github.com/googleapis/librarian/internal/legacylibrarian/legacydocker"
	"github.com/googleapis/librarian/internal/legacylibrarian/legacygitrepo"
	"github.com/googleapis/librarian/internal/legacylibrarian/legacysemver"
)

type stageRunner struct {
	branch          string
	commit          bool
	containerClient ContainerClient
	ghClient        GitHubClient
	image           string
	librarianConfig *legacyconfig.LibrarianConfig
	library         string
	libraryVersion  string
	push            bool
	repo            legacygitrepo.Repository
	sourceRepo      legacygitrepo.Repository
	state           *legacyconfig.LibrarianState
	workRoot        string
}

func newStageRunner(cfg *legacyconfig.Config) (*stageRunner, error) {
	runner, err := newCommandRunner(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create stage runner: %w", err)
	}
	return &stageRunner{
		branch:          cfg.Branch,
		commit:          cfg.Commit,
		containerClient: runner.containerClient,
		ghClient:        runner.ghClient,
		image:           runner.image,
		librarianConfig: runner.librarianConfig,
		library:         cfg.Library,
		libraryVersion:  cfg.LibraryVersion,
		push:            cfg.Push,
		repo:            runner.repo,
		sourceRepo:      runner.sourceRepo,
		state:           runner.state,
		workRoot:        runner.workRoot,
	}, nil
}

func (r *stageRunner) run(ctx context.Context) error {
	outputDir := filepath.Join(r.workRoot, "output")
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output dir: %s", outputDir)
	}
	slog.Info("staging a release", "dir", outputDir)
	if err := r.runStageCommand(ctx, outputDir); err != nil {
		return err
	}

	// No need to update the librarian state if there are no libraries
	// that need to be released
	if !hasLibrariesToRelease(r.state.Libraries) {
		slog.Info("no release created; skipping the commit/PR")
		return nil
	}

	if err := saveLibrarianState(r.repo.GetDir(), r.state); err != nil {
		return err
	}

	prBodyBuilder := func() (string, error) {
		gitHubRepo, err := GetGitHubRepositoryFromGitRepo(r.repo)
		if err != nil {
			return "", fmt.Errorf("failed to get GitHub repository: %w", err)
		}
		return formatReleaseNotes(r.state, gitHubRepo)
	}
	commitInfo := &commitInfo{
		branch:        r.branch,
		commit:        r.commit,
		commitMessage: "chore: create a release",
		ghClient:      r.ghClient,
		prType:        pullRequestRelease,
		// Newly created PRs from the `release stage` command should have a
		// `release:pending` GitHub tab to be tracked for release.
		pullRequestLabels: []string{"release:pending"},
		push:              r.push,
		languageRepo:      r.repo,
		sourceRepo:        r.sourceRepo,
		state:             r.state,
		workRoot:          r.workRoot,
		prBodyBuilder:     prBodyBuilder,
	}
	if err := commitAndPush(ctx, commitInfo); err != nil {
		return fmt.Errorf("failed to commit and push: %w", err)
	}

	return nil
}

// hasLibrariesToRelease searches through the state of each library and checks
// that there is a single library configured to be triggered.
func hasLibrariesToRelease(libraryStates []*legacyconfig.LibraryState) bool {
	for _, library := range libraryStates {
		if library.ReleaseTriggered {
			return true
		}
	}
	return false
}

func (r *stageRunner) runStageCommand(ctx context.Context, outputDir string) error {
	src := r.repo.GetDir()
	librariesToRelease := r.state.Libraries
	if r.library != "" {
		library := r.state.LibraryByID(r.library)
		if library == nil {
			return fmt.Errorf("unable to find library for release: %s", r.library)
		}
		librariesToRelease = []*legacyconfig.LibraryState{library}
	}
	// Mark if there are any library that needs to be released
	foundReleasableLibrary := false
	for _, library := range librariesToRelease {
		if r.librarianConfig != nil {
			libraryConfig := r.librarianConfig.LibraryConfigFor(library.ID)
			if libraryConfig != nil && libraryConfig.ReleaseBlocked && r.library != library.ID {
				// Do not skip the `release_blocked` library if library ID is explicitly specified.
				slog.Info("library has release_blocked, skipping", "id", library.ID)
				continue
			}
		}
		if err := r.processLibrary(library); err != nil {
			return err
		}

		// Copy the library files over if a release is needed
		if library.ReleaseTriggered {
			foundReleasableLibrary = true
		}
	}

	if !foundReleasableLibrary {
		slog.Info("no libraries need to be released")
		return nil
	}

	stageRequest := &legacydocker.ReleaseStageRequest{
		Branch:          r.branch,
		Commit:          r.commit,
		LibrarianConfig: r.librarianConfig,
		LibraryID:       r.library,
		LibraryVersion:  r.libraryVersion,
		Output:          outputDir,
		RepoDir:         src,
		Push:            r.push,
		State:           r.state,
	}

	if err := r.containerClient.ReleaseStage(ctx, stageRequest); err != nil {
		return err
	}

	// Read the response file.
	if _, err := readLibraryState(
		filepath.Join(stageRequest.RepoDir, legacyconfig.LibrarianDir, legacyconfig.ReleaseStageResponse)); err != nil {
		return err
	}

	for _, library := range librariesToRelease {
		// Copy the library files back if a release is needed
		if library.ReleaseTriggered {
			if err := copyLibraryFiles(r.state, r.repo.GetDir(), library.ID, outputDir, false); err != nil {
				return err
			}
		}
	}

	return copyGlobalAllowlist(r.librarianConfig, r.repo.GetDir(), outputDir, false)
}

// processLibrary wrapper to process the library for release. Helps retrieve latest commits
// since the last release and passing the changes to updateLibrary.
func (r *stageRunner) processLibrary(library *legacyconfig.LibraryState) error {
	var tagName string
	if library.Version != "0.0.0" {
		tagFormat := legacyconfig.DetermineTagFormat(library.ID, library, r.librarianConfig)
		tagName = legacyconfig.FormatTag(tagFormat, library.ID, library.Version)
	}
	commits, err := getConventionalCommitsSinceLastRelease(r.repo, library, tagName)
	if err != nil {
		return fmt.Errorf("failed to fetch conventional commits for library, %s: %w", library.ID, err)
	}
	// Filter specifically for commits relevant to a library
	commits = filterCommitsByLibraryID(commits, library.ID)
	return r.updateLibrary(library, commits)
}

// filterCommitsByLibraryID keeps the conventional commits if the given libraryID appears in the Footer or matches
// the libraryID in the commit.
func filterCommitsByLibraryID(commits []*legacygitrepo.ConventionalCommit, libraryID string) []*legacygitrepo.ConventionalCommit {
	var filteredCommits []*legacygitrepo.ConventionalCommit
	for _, commit := range commits {
		if commit.Footers != nil {
			ids, ok := commit.Footers["Library-IDs"]
			libraryIDs := strings.Split(ids, ",")
			if ok && slices.Contains(libraryIDs, libraryID) {
				filteredCommits = append(filteredCommits, commit)
				continue
			}
		}
		if commit.LibraryID == libraryID {
			filteredCommits = append(filteredCommits, commit)
		}
	}
	return filteredCommits
}

// updateLibrary updates the library's state with the new release information:
//
// 1. Determines the library version's next version.
//
// 2. Updates the library's previous version and the new current version.
//
// 3. Set the library's release trigger to true.
func (r *stageRunner) updateLibrary(library *legacyconfig.LibraryState, commits []*legacygitrepo.ConventionalCommit) error {
	var nextVersion string
	// If library version was explicitly set, attempt to use it. Otherwise, try to determine the version from the commits.
	if r.libraryVersion != "" {
		slog.Info("library version override inputted", "currentVersion", library.Version, "inputVersion", r.libraryVersion)
		nextVersion = legacysemver.MaxVersion(library.Version, r.libraryVersion)
		slog.Debug("determined the library's next version from version input", "library", library.ID, "nextVersion", nextVersion)
		// Currently, nextVersion is the max of current version or input version. If nextVersion is equal to the current version,
		// then the input version is either equal or less than current version and cannot be used for release
		if nextVersion == library.Version {
			return fmt.Errorf("inputted version is not SemVer greater than the current version. Set a version SemVer greater than current than: %s", library.Version)
		}
	} else {
		var err error
		nextVersion, err = r.determineNextVersion(commits, library.Version, library.ID)
		if err != nil {
			return err
		}
		slog.Debug("determined the library's next version from commits", "library", library.ID, "nextVersion", nextVersion)
		// Unable to find a releasable unit from the changes
		if nextVersion == library.Version {
			// No library was inputted for release. Skipping this library for release
			if r.library == "" {
				slog.Info("library does not have any releasable units and will not be released.", "library", library.ID, "version", library.Version)
				return nil
			}
			// Library was inputted for release, but does not contain a releasable unit
			return fmt.Errorf("library does not have a releasable unit and will not be released. Use the version flag to force a release for: %s", library.ID)
		}
		slog.Info("updating library to the next version", "library", library.ID, "currentVersion", library.Version, "nextVersion", nextVersion)
	}

	// Update the previous version, we need this value when creating release note.
	library.PreviousVersion = library.Version
	library.Changes = toCommit(commits, library.ID)
	library.Version = nextVersion
	library.ReleaseTriggered = true
	return nil
}

// determineNextVersion determines the next valid SemVer version from the commits or from
// the next_version override value in the legacyconfig.yaml file.
func (r *stageRunner) determineNextVersion(commits []*legacygitrepo.ConventionalCommit, currentVersion string, libraryID string) (string, error) {
	nextVersionFromCommits, err := NextVersion(commits, currentVersion)
	if err != nil {
		return "", err
	}

	if r.librarianConfig == nil {
		slog.Debug("no librarian config")
		return nextVersionFromCommits, nil
	}

	// Look for next_version override from legacyconfig.yaml
	libraryConfig := r.librarianConfig.LibraryConfigFor(libraryID)
	slog.Debug("looking up library config", "library", libraryID, slog.Any("config", libraryConfig))
	if libraryConfig == nil || libraryConfig.NextVersion == "" {
		return nextVersionFromCommits, nil
	}

	// Compare versions and pick latest
	return legacysemver.MaxVersion(nextVersionFromCommits, libraryConfig.NextVersion), nil
}

// toCommit converts a slice of legacygitrepo.ConventionalCommit to a slice of legacyconfig.Commit.
// If the ConventionalCommit has NestedCommits, they are also extracted and
// converted.
// Set LibraryIDs to the given libraryID if the conventional commit doesn't have key `Library-IDs` in the Footers;
// otherwise use the value in the Footers as LibraryIDs.
func toCommit(c []*legacygitrepo.ConventionalCommit, libraryID string) []*legacyconfig.Commit {
	var commits []*legacyconfig.Commit
	for _, cc := range c {
		var libraryIDs string
		libraryIDs, ok := cc.Footers["Library-IDs"]
		if !ok {
			libraryIDs = libraryID
		}

		commits = append(commits, &legacyconfig.Commit{
			Type:          cc.Type,
			Subject:       cc.Subject,
			Body:          cc.Body,
			CommitHash:    cc.CommitHash,
			PiperCLNumber: cc.Footers["PiperOrigin-RevId"],
			LibraryIDs:    libraryIDs,
		})
	}
	return commits
}
