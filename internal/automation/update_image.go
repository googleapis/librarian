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
	updateImageCmdName = "update-image"
)

type updateImageRunner struct {
	build     bool
	projectID string
	push      bool
}

func newUpdateImageRunner(cfg *config.Config) *updateImageRunner {
	return &updateImageRunner{
		build:     cfg.Build,
		projectID: cfg.Project,
		push:      cfg.Push,
	}
}

func (r *updateImageRunner) run(ctx context.Context) error {
	// TODO(https://github.com/googleapis/librarian/issues/2890): refactor this function after all commands are migrated.
	return runCommandFn(ctx, updateImageCmdName, r.projectID, r.push, r.build)
}
