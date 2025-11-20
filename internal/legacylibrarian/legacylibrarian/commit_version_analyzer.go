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
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"

	"github.com/googleapis/librarian/internal/legacylibrarian/legacyconfig"
	"github.com/googleapis/librarian/internal/legacylibrarian/legacygitrepo"
	"github.com/googleapis/librarian/internal/legacylibrarian/legacysemver"
)

// getConventionalCommitsSinceLastRelease returns all conventional commits for the given library since the
// version specified in the state file. The repo should be the language repo.
func getConventionalCommitsSinceLastRelease(repo legacygitrepo.Repository, library *legacyconfig.LibraryState, tag string) ([]*legacygitrepo.ConventionalCommit, error) {
	commits, err := repo.GetCommitsForPathsSinceTag(library.SourceRoots, tag)

	if err != nil {
		return nil, fmt.Errorf("failed to get commits for library %q with source roots %q at tag %q: %w", library.ID, library.SourceRoots, tag, err)
	}

	// checks that if the files in the commit are in the sources root. The release
	// changes are in the language repo and NOT in the source repo.
	shouldIncludeFiles := func(files []string) bool {
		return shouldIncludeForRelease(files, library.SourceRoots, library.ReleaseExcludePaths)
	}

	conventionalCommits, err := convertToConventionalCommits(repo, library, commits, shouldIncludeFiles)
	if err != nil {
		return nil, fmt.Errorf("failed to convert commits to conventional commits for library %q: %w", library.ID, err)
	}
	return conventionalCommits, nil
}

// shouldIncludeForRelease determines if a commit should be included in a release.
// It returns true if there is at least one file in the commit that is under a source_root
// and not under a release_exclude_path.
func shouldIncludeForRelease(files, sourceRoots, excludePaths []string) bool {
	for _, file := range files {
		if isUnderAnyPath(file, sourceRoots) && !isUnderAnyPath(file, excludePaths) {
			return true
		}
	}
	return false
}

// getConventionalCommitsSinceLastGeneration returns all conventional commits for
// all API paths in given library since the last generation. The repo input should
// be the googleapis source repo.
func getConventionalCommitsSinceLastGeneration(sourceRepo legacygitrepo.Repository, library *legacyconfig.LibraryState, lastGenCommit string) ([]*legacygitrepo.ConventionalCommit, error) {
	if lastGenCommit == "" {
		slog.Info("the last generation commit is empty, skip fetching conventional commits", "library", library.ID)
		return make([]*legacygitrepo.ConventionalCommit, 0), nil
	}

	apiPaths := make([]string, 0)
	for _, oneAPI := range library.APIs {
		apiPaths = append(apiPaths, oneAPI.Path)
	}

	sourceCommits, err := sourceRepo.GetCommitsForPathsSinceCommit(apiPaths, lastGenCommit)
	if err != nil {
		return nil, fmt.Errorf("failed to get commits for library %s at commit %s: %w", library.ID, lastGenCommit, err)
	}

	// checks that the files in the commit are in the api paths for the source repo.
	// The generation change is for changes in the source repo and NOT the language repo.
	shouldIncludeFiles := func(sourceFiles []string) bool {
		return shouldIncludeForGeneration(sourceFiles, library)
	}

	return convertToConventionalCommits(sourceRepo, library, sourceCommits, shouldIncludeFiles)
}

// shouldIncludeForGeneration determines if a commit should be included in generation.
// It returns true if there is at least one file in the commit that is under the
// library's API(s) path (a library could have multiple APIs).
func shouldIncludeForGeneration(sourceFiles []string, library *legacyconfig.LibraryState) bool {
	var apiPaths []string
	for _, api := range library.APIs {
		apiPaths = append(apiPaths, api.Path)
	}

	for _, file := range sourceFiles {
		if isUnderAnyPath(file, apiPaths) {
			return true
		}
	}
	return false
}

// libraryFilter filters a list of conventional commits by library ID.
func libraryFilter(commits []*legacygitrepo.ConventionalCommit, libraryID string) []*legacygitrepo.ConventionalCommit {
	var filteredCommits []*legacygitrepo.ConventionalCommit
	for _, commit := range commits {
		if libraryIDs, ok := commit.Footers["Library-IDs"]; ok {
			ids := strings.Split(libraryIDs, ",")
			for _, id := range ids {
				if strings.TrimSpace(id) == libraryID {
					filteredCommits = append(filteredCommits, commit)
					break
				}
			}
		} else if commit.LibraryID == libraryID {
			filteredCommits = append(filteredCommits, commit)
		}
	}
	return filteredCommits
}

// convertToConventionalCommits converts a list of commits in a git repo into a list
// of conventional commits. The filesFilter parameter is custom filter out non-matching
// files depending on a generation or a release change.
func convertToConventionalCommits(sourceRepo legacygitrepo.Repository, library *legacyconfig.LibraryState, commits []*legacygitrepo.Commit, filesFilter func(files []string) bool) ([]*legacygitrepo.ConventionalCommit, error) {
	var conventionalCommits []*legacygitrepo.ConventionalCommit
	for _, commit := range commits {
		files, err := sourceRepo.ChangedFilesInCommit(commit.Hash.String())
		if err != nil {
			return nil, fmt.Errorf("failed to get changed files for commit %s: %w", commit.Hash.String(), err)
		}
		if !filesFilter(files) {
			continue
		}
		parsedCommits, err := legacygitrepo.ParseCommits(commit, library.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to parse commit %s: %w", commit.Hash.String(), err)
		}
		if parsedCommits == nil {
			continue
		}

		parsedCommits = libraryFilter(parsedCommits, library.ID)

		for _, pc := range parsedCommits {
			pc.CommitHash = commit.Hash.String()
		}
		conventionalCommits = append(conventionalCommits, parsedCommits...)
	}
	return conventionalCommits, nil
}

// isUnderAnyPath returns true if the file is under any of the given paths.
func isUnderAnyPath(file string, paths []string) bool {
	for _, p := range paths {
		if p == "." {
			return true
		}
		rel, err := filepath.Rel(p, file)
		if err == nil && !strings.HasPrefix(rel, "..") {
			return true
		}
	}
	return false
}

// NextVersion calculates the next semantic version based on a slice of conventional commits.
func NextVersion(commits []*legacygitrepo.ConventionalCommit, currentVersion string) (string, error) {
	highestChange := getHighestChange(commits)
	return legacysemver.DeriveNext(highestChange, currentVersion)
}

// getHighestChange determines the highest-ranking change type from a slice of commits.
func getHighestChange(commits []*legacygitrepo.ConventionalCommit) legacysemver.ChangeLevel {
	highestChange := legacysemver.None
	for _, commit := range commits {
		var currentChange legacysemver.ChangeLevel
		switch {
		case commit.IsNested:
			// ignore nested commit type for version bump
			// this allows for always increase minor version for generation PR
			currentChange = legacysemver.Minor
		case commit.IsBreaking:
			currentChange = legacysemver.Major
		case commit.Type == "feat":
			currentChange = legacysemver.Minor
		case commit.Type == "fix":
			currentChange = legacysemver.Patch
		}
		if currentChange > highestChange {
			highestChange = currentChange
		}
	}
	return highestChange
}
