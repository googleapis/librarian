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

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/gitrepo"
)

type testGenerateRunner struct {
	branch          string
	image           string
	library         string
	repo            gitrepo.Repository
	sourceRepo      gitrepo.Repository
	state           *config.LibrarianState
	librarianConfig *config.LibrarianConfig
	workRoot        string
	containerClient ContainerClient
	ghClient        GitHubClient
}

func newTestGenerateRunner(cfg *config.Config) (*testGenerateRunner, error) {
	runner, err := newCommandRunner(cfg)
	if err != nil {
		return nil, err
	}
	return &testGenerateRunner{
		branch:          cfg.Branch,
		image:           runner.image,
		library:         cfg.Library,
		repo:            runner.repo,
		sourceRepo:      runner.sourceRepo,
		state:           runner.state,
		librarianConfig: runner.librarianConfig,
		workRoot:        runner.workRoot,
		containerClient: runner.containerClient,
		ghClient:        runner.ghClient,
	}, nil
}

func (r *testGenerateRunner) run(ctx context.Context) error {
	slog.Info("test generate command is not implemented yet")
	fmt.Println("test generate command is not implemented yet")
	// TODO(zhumin): implement the test generate logic
	return nil
}
