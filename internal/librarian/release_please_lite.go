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
	"strings"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/conventionalcommits"
	"github.com/googleapis/librarian/internal/gitrepo"
)

const defaultTagFormat = "{id}-{version}"

// GetConventionalCommitsSinceLastRelease returns all conventional commits for the given library since the
// version specified in the state file.
func GetConventionalCommitsSinceLastRelease(repo gitrepo.Repository, library *config.LibraryState) ([]*conventionalcommits.ConventionalCommit, error) {
	tag := formatTag(library)
	commits, err := repo.GetCommitsForPathsSinceTag(library.SourceRoots, tag)
	if err != nil {
		return nil, fmt.Errorf("failed to get commits for library %s: %w", library.ID, err)
	}
	var result []*conventionalcommits.ConventionalCommit
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
		result = append(result, parsedCommits...)
	}
	return result, nil
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
func formatTag(library *config.LibraryState) string {
	tagFormat := library.TagFormat
	if tagFormat == "" {
		tagFormat = defaultTagFormat
	}
	r := strings.NewReplacer("{id}", library.ID, "{version}", library.Version)
	return r.Replace(tagFormat)
}
