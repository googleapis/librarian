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
	"strings"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/conventionalcommits"
	"github.com/googleapis/librarian/internal/gitrepo"
	"github.com/googleapis/librarian/internal/semver"
)

const defaultTagFormat = "{id}-{version}"

// GetConventionalCommitsSinceLastRelease returns all conventional commits for the given library since the
// version specified in the state file.
func GetConventionalCommitsSinceLastRelease(repo gitrepo.Repository, library *config.LibraryState) ([]*conventionalcommits.ConventionalCommit, error) {
	tag := formatTag(library, "")
	commits, err := repo.GetCommitsForPathsSinceTag(library.SourceRoots, tag)
	if err != nil {
		return nil, fmt.Errorf("failed to get commits for library %s: %w", library.ID, err)
	}

	return convertToConventionalCommits(repo, library, commits)
}

// GetConventionalCommitsSinceLastGeneration returns all conventional commits for
// all API paths in given library since the last generation.
func GetConventionalCommitsSinceLastGeneration(repo gitrepo.Repository, library *config.LibraryState) ([]*conventionalcommits.ConventionalCommit, error) {
	if library.LastGeneratedCommit == "" {
		slog.Info("the last generation commit is empty, skip fetching conventional commits", "library", library.ID)
		return make([]*conventionalcommits.ConventionalCommit, 0), nil
	}
	apiPaths := make([]string, 0)
	for _, oneAPI := range library.APIs {
		apiPaths = append(apiPaths, oneAPI.Path)
	}

	commits, err := repo.GetCommitsForPathsSinceCommit(apiPaths, library.LastGeneratedCommit)
	if err != nil {
		return nil, fmt.Errorf("failed to get commits for library %s: %w", library.ID, err)
	}

	return convertToConventionalCommits(repo, library, commits)
}

func convertToConventionalCommits(repo gitrepo.Repository, library *config.LibraryState, commits []*gitrepo.Commit) ([]*conventionalcommits.ConventionalCommit, error) {
	var conventionalCommits []*conventionalcommits.ConventionalCommit
	for _, commit := range commits {
		files, err := repo.ChangedFilesInCommit(commit.Hash.String())
		if err != nil {
			return nil, fmt.Errorf("failed to get changed files for commit %s: %w", commit.Hash.String(), err)
		}
		if shouldExclude(files, library.ReleaseExcludePaths) {
			continue
		}
		parsedCommits, err := conventionalcommits.ParseCommits(commit.Message, commit.Hash.String())
		if err != nil {
			return nil, fmt.Errorf("failed to parse commit %s: %w", commit.Hash.String(), err)
		}
		if parsedCommits == nil {
			continue
		}
		conventionalCommits = append(conventionalCommits, parsedCommits...)
	}
	return conventionalCommits, nil
}

// shouldExclude determines if a commit should be excluded from a release.
// It returns true if all files in the commit match one of the exclude paths.
func shouldExclude(files, excludePaths []string) bool {
	for _, file := range files {
		excluded := false
		for _, excludePath := range excludePaths {
			if strings.HasPrefix(file, excludePath) {
				excluded = true
				break
			}
		}
		if !excluded {
			return false
		}
	}
	return true
}

// formatTag returns the git tag for a given library version.
func formatTag(library *config.LibraryState, versionOverride string) string {
	version := library.Version
	if versionOverride != "" {
		version = versionOverride
	}
	tagFormat := library.TagFormat
	if tagFormat == "" {
		tagFormat = defaultTagFormat
	}
	r := strings.NewReplacer("{id}", library.ID, "{version}", version)
	return r.Replace(tagFormat)
}

// NextVersion calculates the next semantic version based on a slice of conventional commits.
// If overrideNextVersion is not empty, it is returned as the next version.
func NextVersion(commits []*conventionalcommits.ConventionalCommit, currentVersion, overrideNextVersion string) (string, error) {
	if overrideNextVersion != "" {
		return overrideNextVersion, nil
	}
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
