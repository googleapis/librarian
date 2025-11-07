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

package automation

import (
	"context"
	"log/slog"

	"github.com/googleapis/librarian/internal/config"
)

// runCommandFn is a function type that matches RunCommand, for mocking in tests.
var runCommandFn = RunCommand

type automationRunner struct {
	build    bool
	command  string
	forceRun bool
	project  string
	push     bool
}

func newAutomationRunner(cfg *config.Config) *automationRunner {
	return &automationRunner{
		build:    cfg.Build,
		command:  cfg.CommandName,
		forceRun: cfg.ForceRun,
		project:  cfg.Project,
		push:     cfg.Push,
	}
}

func (r *automationRunner) run(ctx context.Context) error {
	err := runCommandFn(ctx, r.command, r.project, r.push, r.build, r.forceRun)
	if err != nil {
		slog.Error("error running command", slog.Any("err", err))
		return err
	}
	return nil
}
