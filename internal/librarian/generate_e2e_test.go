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

//go:build e2e
// +build e2e

package librarian

import (
	"context"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/googleapis/librarian/internal/config"
	"testing"
	"time"

	"github.com/go-git/go-git/v5"
)

func TestRunGenerate(t *testing.T) {
	const (
		localRepoDir = "../../testdata/e2e/generate/repo"
	)
	for _, test := range []struct {
		name string
		cfg  *config.Config
	}{
		{
			name: "testRun",
			cfg: &config.Config{
				API:    "google/cloud/pubsub/v1",
				Repo:   localRepoDir,
				Source: "../../testdata/e2e/generate/api_root",
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			if err := initTestRepo(localRepoDir); err != nil {
				t.Errorf("init test repo error = %v", err)
			}
			runner, err := newGenerateRunner(test.cfg)
			if err != nil {
				t.Errorf("newGenerateRunner() error = %v", err)
			}
			if err := runner.run(context.Background()); err != nil {
				t.Errorf("run() error = %v", err)
			}
		})
	}
}

func initTestRepo(dir string) error {
	localRepo, err := git.PlainInit(dir, false)
	if err != nil {
		return err
	}
	workTree, err := localRepo.Worktree()
	if err != nil {
		return err
	}
	if _, err := workTree.Add("."); err != nil {
		return err
	}

	if _, err := workTree.Commit("init test repo", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "test user",
			Email: "test@github.com",
			When:  time.Now(),
		},
	}); err != nil {
		return err
	}

	return nil
}
