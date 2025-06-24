// Copyright 2024 Google LLC
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

package gitrepo

import (
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"os"
	"strings"
	"testing"
)

func TestGetCommitsForPathsSinceCommit(t *testing.T) {
	tests := []struct {
		filePaths    []string
		messages     []string
		wantPaths    []string
		numOfCommits int
	}{
		{
			[]string{"local/first", "local/second", "local/third"},
			[]string{"first commit", "2nd commit", "3rd commit"},
			[]string{"local/first", "local/third"},
			2,
		},
		{
			[]string{"local/first", "local/second", "local/third"},
			[]string{"first commit", "2nd commit", "3rd commit"},
			[]string{"local/zero"},
			0,
		},
		{
			[]string{"local/first", "local/second", "local/third"},
			[]string{"first commit", "2nd commit", "3rd commit"},
			[]string{},
			0,
		},
	}
	dir, err := os.MkdirTemp("", "test-*")
	if err != nil {
		t.Errorf("Create a temp directory in %s got error %v.", dir, err)
	}
	defer func(path string) {
		err := os.RemoveAll(path)
		if err != nil {
			t.Errorf("Remove a temp directory in %s got error %v.", dir, err)
		}
	}(dir)

	for _, test := range tests {
		repoDir, _ := os.MkdirTemp(dir, "test-repo-*")
		localRepo, _ := git.PlainInit(repoDir, false)
		worktree, _ := localRepo.Worktree()
		_, err = os.Create(strings.Join([]string{repoDir, "test.txt"}, "/"))
		if err != nil {
			t.Logf("create file has error: %s", err)
		}
		_, err = worktree.Add("test.txt")
		if err != nil {
			t.Logf("git-add file has error: %s", err)
		}
		firstCommit, err := worktree.Commit("empty commit", &git.CommitOptions{})
		if err != nil {
			t.Logf("git-commit file has error: %s", err)
		}
		t.Logf("first commit is %s", firstCommit)
		parent := firstCommit
		for i := 0; i < len(test.filePaths); i++ {
			absFilePath := strings.Join([]string{repoDir, test.filePaths[i]}, "/")
			absFileName := strings.Join([]string{absFilePath, "file.txt"}, "/")
			_ = os.MkdirAll(absFilePath, 0755)
			_, _ = os.Create(absFileName)
			_, _ = worktree.Add(test.filePaths[i] + "/file.txt")
			current, _ := worktree.Commit(test.messages[i], &git.CommitOptions{
				Parents: []plumbing.Hash{parent},
			})
			parent = current
		}

		repo, err := NewRepository(&RepositoryOptions{
			Dir:        repoDir,
			MaybeClone: false,
		})
		if err != nil {
			t.Errorf("NewRepository(%s); got error %v", repoDir, err)
			continue
		}
		commits, err := repo.GetCommitsForPathsSinceCommit(test.wantPaths, firstCommit.String())
		if len(test.wantPaths) == 0 && err == nil {
			t.Errorf("GetCommitsForPathsSinceCommit(%s) should fail", test.filePaths)
		}
		if len(test.wantPaths) != 0 && err != nil {
			t.Errorf("GetCommitsForPathsSinceCommit(%s) expected %d commit(s), got error %v",
				test.wantPaths,
				test.numOfCommits,
				err)
		}
		if len(test.wantPaths) != 0 && len(commits) != test.numOfCommits {
			t.Errorf("GetCommitsForPathsSinceCommit(%s) expected %d commit(s), got %d",
				test.wantPaths,
				test.numOfCommits,
				len(commits))
		}
	}
}
