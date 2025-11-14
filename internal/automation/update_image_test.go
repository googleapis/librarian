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

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
)

func TestNewUpdateImageRunner(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name string
		cfg  *config.Config
		want *updateImageRunner
	}{
		{
			name: "create_a_runner",
			cfg: &config.Config{
				Build:   true,
				Project: "example-project",
				Push:    true,
			},
			want: &updateImageRunner{
				build:     true,
				projectID: "example-project",
				push:      true,
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			got := newUpdateImageRunner(test.cfg)
			if diff := cmp.Diff(test.want, got, cmp.AllowUnexported(updateImageRunner{})); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestUpdateImageRunnerRun(t *testing.T) {
	tests := []struct {
		name          string
		runner        *updateImageRunner
		runCommandErr error
		wantErr       bool
		wantCmd       string
		wantProjectID string
		wantPush      bool
		wantBuild     bool
	}{
		{
			name: "success",
			runner: &updateImageRunner{
				build:     true,
				projectID: "test-project",
				push:      true,
			},
			wantCmd:       updateImageCmdName,
			wantProjectID: "test-project",
			wantPush:      true,
			wantBuild:     true,
		},
		{
			name:          "error from RunCommand",
			runner:        &updateImageRunner{},
			runCommandErr: errors.New("run command failed"),
			wantErr:       true,
			wantCmd:       updateImageCmdName,
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
					if push != test.wantPush {
						t.Errorf("runCommandFn() push = %v, want %v", push, test.wantPush)
					}
					if build != test.wantBuild {
						t.Errorf("runCommandFn() build = %v, want %v", build, test.wantBuild)
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
