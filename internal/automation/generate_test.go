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

func TestNewGenerateRunner(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name string
		cfg  *config.Config
	}{
		{
			name: "create_a_runner",
			cfg: &config.Config{
				Build:   true,
				Project: "example-project",
				Push:    true,
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			runner := newGenerateRunner(test.cfg)
			if runner.build != test.cfg.Build {
				t.Errorf("newGenerateRunner() build is not set")
			}
			if runner.projectID != test.cfg.Project {
				t.Errorf("newGenerateRunner() projectID is not set")
			}
			if runner.push != test.cfg.Push {
				t.Errorf("newGenerateRunner() push is not set")
			}
		})
	}
}

func TestGenerateRunnerRun(t *testing.T) {
	originalRunCommandFn := runCommandFn
	defer func() { runCommandFn = originalRunCommandFn }()

	tests := []struct {
		name          string
		runner        *generateRunner
		runCommandErr error
		wantErr       bool
		wantCmd       string
		wantProjectID string
		wantPush      bool
		wantBuild     bool
	}{
		{
			name: "success",
			runner: &generateRunner{
				build:     true,
				projectID: "test-project",
				push:      true,
			},
			wantCmd:       generateCmdName,
			wantProjectID: "test-project",
			wantPush:      true,
			wantBuild:     true,
		},
		{
			name:          "error from RunCommand",
			runner:        &generateRunner{},
			runCommandErr: errors.New("run command failed"),
			wantErr:       true,
			wantCmd:       generateCmdName,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runCommandFn = func(ctx context.Context, command string, projectId string, push bool, build bool) error {
				if command != tt.wantCmd {
					t.Errorf("runCommandFn() command = %v, want %v", command, tt.wantCmd)
				}
				// Only check other args on success case to avoid nil pointer with empty runner
				if tt.runCommandErr == nil {
					if projectId != tt.wantProjectID {
						t.Errorf("runCommandFn() projectId = %v, want %v", projectId, tt.wantProjectID)
					}
					if push != tt.wantPush {
						t.Errorf("runCommandFn() push = %v, want %v", push, tt.wantPush)
					}
					if build != tt.wantBuild {
						t.Errorf("runCommandFn() build = %v, want %v", build, tt.wantBuild)
					}
				}
				return tt.runCommandErr
			}

			if err := tt.runner.run(t.Context()); (err != nil) != tt.wantErr {
				t.Errorf("run() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
