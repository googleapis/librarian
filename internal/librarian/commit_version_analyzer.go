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
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/conventionalcommits"
	"github.com/googleapis/librarian/internal/gitrepo"
	"github.com/googleapis/librarian/internal/semver"
)

// getConventionalCommitsSinceLastRelease returns all conventional commits for the given library since the
// version specified in the state file. The repo should be the language repo.
func getConventionalCommitsSinceLastRelease(repo gitrepo.Repository, library *config.LibraryState, tag string) ([]*conventionalcommits.ConventionalCommit, error) {
	commits, err := repo.GetCommitsForPathsSinceTag(library.SourceRoots, tag)

	if err != nil {
		return nil, fmt.Errorf("failed to get commits for library %s: %w", library.ID, err)
	}

	// checks that if the files in the commit are in the sources root. The release
	// changes are in the language repo and NOT in the source repo.
	shouldIncludeFiles := func(files []string) bool {
		return shouldIncludeForRelease(files, library.SourceRoots, library.ReleaseExcludePaths)
	}

	return convertToConventionalCommits(repo, library, commits, shouldIncludeFiles)
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
func getConventionalCommitsSinceLastGeneration(sourceRepo, languageRepo gitrepo.Repository, library *config.LibraryState, lastGenCommit string) ([]*conventionalcommits.ConventionalCommit, error) {
	if lastGenCommit == "" {
		slog.Info("the last generation commit is empty, skip fetching conventional commits", "library", library.ID)
		return make([]*conventionalcommits.ConventionalCommit, 0), nil
	}

	apiPaths := make([]string, 0)
	for _, oneAPI := range library.APIs {
		apiPaths = append(apiPaths, oneAPI.Path)
	}

	sourceCommits, err := sourceRepo.GetCommitsForPathsSinceCommit(apiPaths, lastGenCommit)
	if err != nil {
		return nil, fmt.Errorf("failed to get commits for library %s at commit %s: %w", library.ID, lastGenCommit, err)
	}

	var languageRepoFiles []string
	if ok, _ := languageRepo.IsClean(); ok {
		headHash, err := languageRepo.HeadHash()
		if err != nil {
			return nil, err
		}
		languageRepoFiles, err = languageRepo.ChangedFilesInCommit(headHash)
		if err != nil {
			return nil, err
		}
	} else {
		// The commit or push flag is not set, get all locally changed files.
		languageRepoFiles, err = languageRepo.ChangedFiles()
		if err != nil {
			return nil, err
		}
	}

	// checks that the files in the commit are in the api paths for the source repo.
	// The generation change is for changes in the source repo and NOT the language repo.
	shouldIncludeFiles := func(sourceFiles []string) bool {
		return shouldIncludeForGeneration(sourceFiles, languageRepoFiles, library)
	}

	return convertToConventionalCommits(sourceRepo, library, sourceCommits, shouldIncludeFiles)
}

// shouldIncludeForGeneration determines if a commit should be included in generation.
// It returns true if there is at least one file in the commit that is under the
// library's API(s) path (a library could have multiple APIs) and has local
// changes associated with it.
func shouldIncludeForGeneration(sourceFiles, languageRepoFiles []string, library *config.LibraryState) bool {
	var apiPaths []string
	for _, api := range library.APIs {
		apiPaths = append(apiPaths, api.Path)
	}

	var sourceFilesInPath bool
	for _, file := range sourceFiles {
		if isUnderAnyPath(file, apiPaths) {
			sourceFilesInPath = true
			break
		}
	}

	var languageRepoFilesInPath bool
	for _, file := range languageRepoFiles {
		if isUnderAnyPath(file, library.SourceRoots) && !isUnderAnyPath(file, library.ReleaseExcludePaths) {
			languageRepoFilesInPath = true
			break
		}
	}
	return sourceFilesInPath && languageRepoFilesInPath
}

// libraryFilter filters a list of conventional commits by library ID.
func libraryFilter(commits []*conventionalcommits.ConventionalCommit, libraryID string) []*conventionalcommits.ConventionalCommit {
	var filteredCommits []*conventionalcommits.ConventionalCommit
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
func convertToConventionalCommits(sourceRepo gitrepo.Repository, library *config.LibraryState, commits []*gitrepo.Commit, filesFilter func(files []string) bool) ([]*conventionalcommits.ConventionalCommit, error) {
	var conventionalCommits []*conventionalcommits.ConventionalCommit
	for _, commit := range commits {
		files, err := sourceRepo.ChangedFilesInCommit(commit.Hash.String())
		if err != nil {
			return nil, fmt.Errorf("failed to get changed files for commit %s: %w", commit.Hash.String(), err)
		}
		if !filesFilter(files) {
			continue
		}
		parsedCommits, err := conventionalcommits.ParseCommits(commit, library.ID)
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
func NextVersion(commits []*conventionalcommits.ConventionalCommit, currentVersion string) (string, error) {
	highestChange := getHighestChange(commits)
	return semver.DeriveNext(highestChange, currentVersion)
}

// getHighestChange determines the highest-ranking change type from a slice of commits.
func getHighestChange(commits []*conventionalcommits.ConventionalCommit) semver.ChangeLevel {
	highestChange := semver.None
	for _, commit := range commits {
		var currentChange semver.ChangeLevel
		switch {
		case commit.IsNested:
			// ignore nested commit type for version bump
			// this allows for always increase minor version for generation PR
			currentChange = semver.Minor
		case commit.IsBreaking:
			currentChange = semver.Major
		case commit.Type == "feat":
			currentChange = semver.Minor
		case commit.Type == "fix":
			currentChange = semver.Patch
		}
		if currentChange > highestChange {
			highestChange = currentChange
		}
	}
	return highestChange
}
