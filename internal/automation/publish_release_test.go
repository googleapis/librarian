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
	"errors"
	"testing"

	"github.com/googleapis/librarian/internal/config"
)

func TestNewPublishRunner(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name string
		cfg  *config.Config
	}{
		{
			name: "create_a_runner",
			cfg: &config.Config{
				Project: "example-project",
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			runner := newPublishRunner(test.cfg)
			if runner.projectID != test.cfg.Project {
				t.Errorf("newPublishRunner() projectID is not set")
			}
		})
	}
}

func TestPublishRunnerRun(t *testing.T) {
	originalRunCommandFn := runCommandFn
	defer func() { runCommandFn = originalRunCommandFn }()

	tests := []struct {
		name          string
		runner        *publishRunner
		runCommandErr error
		wantErr       bool
		wantCmd       string
		wantProjectID string
		wantPush      bool
		wantBuild     bool
	}{
		{
			name: "success",
			runner: &publishRunner{
				projectID: "test-project",
			},
			wantCmd:       publishCmdName,
			wantProjectID: "test-project",
		},
		{
			name:          "error from RunCommand",
			runner:        &publishRunner{},
			runCommandErr: errors.New("run command failed"),
			wantErr:       true,
			wantCmd:       publishCmdName,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			runCommandFn = func(ctx context.Context, command string, projectId string, push bool, build bool) error {
				if command != test.wantCmd {
					t.Errorf("runCommandFn() command = %v, want %v", command, test.wantCmd)
				}
				// Only check other args on success case to avoid nil pointer with empty runner
				if test.runCommandErr == nil {
					if projectId != test.wantProjectID {
						t.Errorf("runCommandFn() projectId = %v, want %v", projectId, test.wantProjectID)
					}
				}
				return test.runCommandErr
			}

			if err := test.runner.run(t.Context()); (err != nil) != test.wantErr {
				t.Errorf("run() error = %v, wantErr %v", err, test.wantErr)
			}
		})
	}
}
