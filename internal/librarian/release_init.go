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

	"github.com/googleapis/librarian/internal/conventionalcommits"
	"github.com/googleapis/librarian/internal/docker"
	"github.com/googleapis/librarian/internal/semver"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/gitrepo"
)

type initRunner struct {
	branch          string
	commit          bool
	containerClient ContainerClient
	ghClient        GitHubClient
	image           string
	librarianConfig *config.LibrarianConfig
	library         string
	libraryVersion  string
	push            bool
	repo            gitrepo.Repository
	sourceRepo      gitrepo.Repository
	state           *config.LibrarianState
	workRoot        string
}

func newInitRunner(cfg *config.Config) (*initRunner, error) {
	languageRepo, err := cloneOrOpenRepo(cfg.WorkRoot, cfg.Repo, cfg.APISourceDepth, cfg.Branch, cfg.CI, cfg.GitHubToken)
	if err != nil {
		return nil, err
	}
	state, err := loadRepoState(languageRepo, "")
	if err != nil {
		return nil, err
	}
	librarianConfig, err := loadLibrarianConfig(languageRepo)
	if err != nil {
		return nil, err
	}

	image := deriveImage(cfg.Image, state)
	container, err := docker.New(cfg.WorkRoot, image, cfg.UserUID, cfg.UserGID)
	if err != nil {
		return nil, err
	}

	ghClient, err := newGitHubClient(cfg.Repo, cfg.GitHubToken, languageRepo)
	if err != nil {
		return nil, fmt.Errorf("failed to create GitHub client: %w", err)
	}

	return &initRunner{
		branch:          cfg.Branch,
		commit:          cfg.Commit,
		containerClient: container,
		ghClient:        ghClient,
		image:           deriveImage(cfg.Image, state),
		librarianConfig: librarianConfig,
		library:         cfg.Library,
		libraryVersion:  cfg.LibraryVersion,
		partialRepo:     filepath.Join(cfg.WorkRoot, "release-init"),
		push:            cfg.Push,
		repo:            languageRepo,
		state:           state,
		workRoot:        cfg.WorkRoot,
	}, nil
}

func (r *initRunner) run(ctx context.Context) error {
	outputDir := filepath.Join(r.workRoot, "output")
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output dir: %s", outputDir)
	}
	slog.Info("Initiating a release", "dir", outputDir)
	if err := r.runInitCommand(ctx, outputDir); err != nil {
		return err
	}

	// No need to update the librarian state if there are no libraries
	// that need to be released
	if !hasLibrariesToRelease(r.state.Libraries) {
		slog.Info("No release created; skipping the commit/PR")
		return nil
	}

	if err := saveLibrarianState(r.repo.GetDir(), r.state); err != nil {
		return err
	}

	commitInfo := &commitInfo{
		branch:         r.branch,
		commit:         r.commit,
		commitMessage:  "chore: create a release",
		ghClient:       r.ghClient,
		library:        r.library,
		libraryVersion: r.libraryVersion,
		prType:         release,
		// Newly created PRs from the `release init` command should have a
		// `release:pending` GitHub tab to be tracked for release.
		pullRequestLabels: []string{"release:pending"},
		push:              r.push,
		repo:              r.repo,
		sourceRepo:        r.sourceRepo,
		state:             r.state,
		workRoot:          r.workRoot,
	}
	if err := commitAndPush(ctx, commitInfo); err != nil {
		return fmt.Errorf("failed to commit and push: %w", err)
	}

	return nil
}

// hasLibrariesToRelease searches through the state of each library and checks
// that there is a single library configured to be triggered.
func hasLibrariesToRelease(libraryStates []*config.LibraryState) bool {
	for _, library := range libraryStates {
		if library.ReleaseTriggered {
			return true
		}
	}
	return false
}

func (r *initRunner) runInitCommand(ctx context.Context, outputDir string) error {
	src := r.repo.GetDir()
	librariesToRelease := r.state.Libraries
	if r.library != "" {
		library := findLibraryByID(r.state, r.library)
		if library == nil {
			return fmt.Errorf("unable to find library for release: %s", r.library)
		}
		librariesToRelease = []*config.LibraryState{library}
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
		slog.Info("No libraries need to be released")
		return nil
	}

	initRequest := &docker.ReleaseInitRequest{
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

	if err := r.containerClient.ReleaseInit(ctx, initRequest); err != nil {
		return err
	}

	// Read the response file.
	if _, err := readLibraryState(
		filepath.Join(initRequest.RepoDir, config.LibrarianDir, config.ReleaseInitResponse)); err != nil {
		return err
	}

	for _, library := range librariesToRelease {
		// Copy the library files back if a release is needed
		if library.ReleaseTriggered {
			if err := copyLibraryFiles(r.state, r.repo.GetDir(), library.ID, outputDir); err != nil {
				return err
			}
		}
	}

	return copyGlobalAllowlist(r.librarianConfig, r.repo.GetDir(), outputDir, false)
}

// processLibrary wrapper to process the library for release. Helps retrieve latest commits
// since the last release and passing the changes to updateLibrary.
func (r *initRunner) processLibrary(library *config.LibraryState) error {
	commits, err := getConventionalCommitsSinceLastRelease(r.repo, library)
	if err != nil {
		return fmt.Errorf("failed to fetch conventional commits for library, %s: %w", library.ID, err)
	}
	// Filter specifically for commits relevant to a library
	commits = filterCommitsByLibraryID(commits, library.ID)
	return r.updateLibrary(library, commits)
}

// filterCommitsByLibraryID keeps the conventional commits that match a libaryID.
func filterCommitsByLibraryID(commits []*conventionalcommits.ConventionalCommit, libraryID string) []*conventionalcommits.ConventionalCommit {
	var filteredCommits []*conventionalcommits.ConventionalCommit
	for _, commit := range commits {
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
func (r *initRunner) updateLibrary(library *config.LibraryState, commits []*conventionalcommits.ConventionalCommit) error {
	var nextVersion string
	// If library version was explicitly set, attempt to use it. Otherwise, try to determine the version from the commits.
	if r.libraryVersion != "" {
		slog.Info("Library version override inputted", "currentVersion", library.Version, "inputVersion", r.libraryVersion)
		nextVersion = semver.MaxVersion(library.Version, r.libraryVersion)
		slog.Debug("Determined the library's next version from version input", "library", library.ID, "nextVersion", nextVersion)
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
		slog.Debug("Determined the library's next version from commits", "library", library.ID, "nextVersion", nextVersion)
		// Unable to find a releasable unit from the changes
		if nextVersion == library.Version {
			// No library was inputted for release. Skipping this library for release
			if r.library == "" {
				slog.Info("Library does not have any releasable units and will not be released.", "library", library.ID, "version", library.Version)
				return nil
			}
			// Library was inputted for release, but does not contain a releasable unit
			return fmt.Errorf("library does not have a releasable unit and will not be released. Use the version flag to force a release for: %s", library.ID)
		}
		slog.Info("Updating library to the next version", "library", r.library, "currentVersion", library.Version, "nextVersion", nextVersion)
	}

	// Update the previous version, we need this value when creating release note.
	library.PreviousVersion = library.Version
	library.Changes = commits
	library.Version = nextVersion
	library.ReleaseTriggered = true
	return nil
}

// determineNextVersion determines the next valid SemVer version from the commits or from
// the next_version override value in the config.yaml file.
func (r *initRunner) determineNextVersion(commits []*conventionalcommits.ConventionalCommit, currentVersion string, libraryID string) (string, error) {
	nextVersionFromCommits, err := NextVersion(commits, currentVersion)
	if err != nil {
		return "", err
	}

	if r.librarianConfig == nil {
		slog.Info("No librarian config")
		return nextVersionFromCommits, nil
	}

	// Look for next_version override from config.yaml
	libraryConfig := r.librarianConfig.LibraryConfigFor(libraryID)
	slog.Info("Looking up library config", "library", libraryID, slog.Any("config", libraryConfig))
	if libraryConfig == nil || libraryConfig.NextVersion == "" {
		return nextVersionFromCommits, nil
	}

	// Compare versions and pick latest
	return semver.MaxVersion(nextVersionFromCommits, libraryConfig.NextVersion), nil
}
