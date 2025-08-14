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
	"errors"
	"fmt"

	"github.com/googleapis/librarian/internal/cli"
	"github.com/googleapis/librarian/internal/config"
)

// cmdInit is the command for the `release init` subcommand.
var cmdInit = &cli.Command{
	Short:     "release init initiates a release by creating a release pull request.",
	UsageLine: "librarian release init [arguments]",
	Long: `The release init command is the primary entry point for initiating a release.
It orchestrates the process of parsing commits, determining new versions, generating
a changelog, and creating a release pull request.`,
	Run: func(ctx context.Context, cfg *config.Config) error {
		runner, err := newInitRunner(cfg)
		if err != nil {
			return err
		}
		return runner.run(ctx)
	},
}

func init() {
	cmdInit.Init()
	fs := cmdInit.Flags
	cfg := cmdInit.Config

	addFlagRepo(fs, cfg)
	addFlagImage(fs, cfg)
	addFlagLibrary(fs, cfg)
}

type initRunner struct {
	cfg *config.Config
}

func newInitRunner(cfg *config.Config) (*initRunner, error) {
	if ok, err := cfg.IsValid(); !ok || err != nil {
		return nil, fmt.Errorf("invalid config: %+v", cfg)
	}

	return &initRunner{
		cfg: cfg,
	}, nil
}

func (r *initRunner) run(ctx context.Context) error {
	return errors.New("not implemented")
}
