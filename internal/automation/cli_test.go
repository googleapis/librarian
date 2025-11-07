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
)

func TestRun(t *testing.T) {

	tests := []struct {
		name          string
		args          []string
		runCommandErr error
		wantErr       bool
	}{
		{
			name:    "success",
			args:    []string{"--command=generate"},
			wantErr: false,
		},
		{
			name:    "error parsing flags",
			args:    []string{"--unknown-flag"},
			wantErr: true,
		},
		{
			name:          "error from RunCommand",
			args:          []string{"--command=generate"},
			runCommandErr: errors.New("run command failed"),
			wantErr:       true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runCommandFn = func(ctx context.Context, command string, projectId string, push bool, build bool, forceRun bool) error {
				return tt.runCommandErr
			}
			if err := Run(context.Background(), tt.args); (err != nil) != tt.wantErr {
				t.Errorf("Run() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
