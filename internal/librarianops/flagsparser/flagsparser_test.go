// Copyright 2026 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package flagsparser

import (
	"context"
	"testing"

	"github.com/googleapis/librarian/internal/command"
	"github.com/urfave/cli/v3"
)

func TestParseRepoFlags(t *testing.T) {
	testCases := []struct {
		name         string
		args         []string
		wantRepoName string
		wantWorkDir  string
		wantErr      bool
		wantVerbose  bool
	}{
		{
			name:         "with -C flag",
			args:         []string{"librarianops", "test-command", "-C", "/path/to/repo"},
			wantRepoName: "repo",
			wantWorkDir:  "/path/to/repo",
			wantErr:      false,
			wantVerbose:  false,
		},
		{
			name:         "with positional argument",
			args:         []string{"librarianops", "test-command", "repo-name"},
			wantRepoName: "repo-name",
			wantWorkDir:  "",
			wantErr:      false,
			wantVerbose:  false,
		},
		{
			name:         "with -v flag",
			args:         []string{"librarianops", "test-command", "-v", "repo-name"},
			wantRepoName: "repo-name",
			wantWorkDir:  "",
			wantErr:      false,
			wantVerbose:  true,
		},
		{
			name:    "no arguments",
			args:    []string{"librarianops", "test-command"},
			wantErr: true,
		},
		{
			name:         "with -C and -v",
			args:         []string{"librarianops", "test-command", "-C", "/path/to/repo", "-v"},
			wantRepoName: "repo",
			wantWorkDir:  "/path/to/repo",
			wantErr:      false,
			wantVerbose:  true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Reset global verbose flag for each test.
			originalVerbose := command.Verbose
			command.Verbose = false
			t.Cleanup(func() { command.Verbose = originalVerbose })

			var repoName, workDir string
			var err error
			app := &cli.Command{
				Name: "librarianops",
				Commands: []*cli.Command{
					{
						Name: "test-command",
						Flags: []cli.Flag{
							&cli.StringFlag{Name: "C"},
							&cli.BoolFlag{Name: "v"},
						},
						Action: func(ctx context.Context, cmd *cli.Command) error {
							repoName, workDir, err = ParseRepoFlags(cmd)
							return err
						},
					},
				},
			}

			runErr := app.Run(context.Background(), tc.args)

			if (runErr != nil) != tc.wantErr {
				t.Fatalf("app.Run() error = %v, wantErr %v", runErr, tc.wantErr)
			}
			if tc.wantErr {
				return
			}

			if repoName != tc.wantRepoName {
				t.Errorf("parseRepoFlags() repoName = %q, want %q", repoName, tc.wantRepoName)
			}
			if workDir != tc.wantWorkDir {
				t.Errorf("parseRepoFlags() workDir = %q, want %q", workDir, tc.wantWorkDir)
			}
			if command.Verbose != tc.wantVerbose {
				t.Errorf("command.Verbose = %v, want %v", command.Verbose, tc.wantVerbose)
			}
		})
	}
}
