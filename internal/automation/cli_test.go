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
			runCommandFn = func(ctx context.Context, command string, projectId string, push bool, build bool) error {
				return tt.runCommandErr
			}
			if err := Run(context.Background(), tt.args); (err != nil) != tt.wantErr {
				t.Errorf("Run() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestParseArgs(t *testing.T) {
	for _, test := range []struct {
		name    string
		args    []string
		want    *runOptions
		wantErr bool
	}{
		{
			name:    "parses defaults",
			args:    []string{},
			wantErr: false,
			want: &runOptions{
				Command:   "generate",
				ProjectId: "cloud-sdk-librarian-prod",
				Push:      true,
				Build:     true,
			},
		},
		{
			name:    "sets project",
			args:    []string{"--project=some-project-id"},
			wantErr: false,
			want: &runOptions{
				Command:   "generate",
				ProjectId: "some-project-id",
				Push:      true,
				Build:     true,
			},
		},
		{
			name:    "sets command",
			args:    []string{"--command=stage-release"},
			wantErr: false,
			want: &runOptions{
				Command:   "stage-release",
				ProjectId: "cloud-sdk-librarian-prod",
				Push:      true,
				Build:     true,
			},
		},
		{
			name:    "sets command",
			args:    []string{"--command=stage-release", "--push=false"},
			wantErr: false,
			want: &runOptions{
				Command:   "stage-release",
				ProjectId: "cloud-sdk-librarian-prod",
				Push:      false,
				Build:     true,
			},
		},
		{
			name:    "sets build",
			args:    []string{"--command=generate", "--build=false"},
			wantErr: false,
			want: &runOptions{
				Command:   "generate",
				ProjectId: "cloud-sdk-librarian-prod",
				Push:      true,
				Build:     false,
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := parseFlags(test.args)
			if test.wantErr && err == nil {
				t.Fatal("expected error, but did not return one")
			} else if !test.wantErr && err != nil {
				t.Errorf("did not expect error, but received one: %s", err)
			}
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
