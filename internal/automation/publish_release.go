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

	"github.com/googleapis/librarian/internal/config"
)

const (
	publishCmdName = "publish-release"
)

type publishRunner struct {
	forceRun  bool
	projectID string
	push      bool
}

func newPublishRunner(cfg *config.Config) *publishRunner {
	return &publishRunner{
		forceRun:  cfg.ForceRun,
		projectID: cfg.Project,
		push:      cfg.Push,
	}
}

func (r *publishRunner) run(ctx context.Context) error {
	return RunCommand(ctx, publishCmdName, r.projectID, r.push, false, r.forceRun)
}
