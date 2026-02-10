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

package librarianops

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/urfave/cli/v3"
)

func TestParseRepoFlags(t *testing.T) {
	app := &cli.Command{
		Name: "librarianops",
		Commands: []*cli.Command{
			{
				Name: "test-command",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "C"},
					&cli.BoolFlag{Name: "v"},
				},
			},
		},
	}
	for _, testCase := range []struct {
		name         string
		args         []string
		wantRepoName string
		wantWorkDir  string
		wantVerbose  bool
	}{
		{
			name:         "with -C flag",
			args:         []string{"librarianops", "test-command", "-C", "/path/to/repo"},
			wantRepoName: "repo",
			wantWorkDir:  "/path/to/repo",
			wantVerbose:  false,
		},
		{
			name:         "with positional argument",
			args:         []string{"librarianops", "test-command", "repo-name"},
			wantRepoName: "repo-name",
			wantWorkDir:  "",
			wantVerbose:  false,
		},
		{
			name:         "with -v flag",
			args:         []string{"librarianops", "test-command", "-v", "repo-name"},
			wantRepoName: "repo-name",
			wantWorkDir:  "",
			wantVerbose:  true,
		},
		{
			name:         "with -C and -v",
			args:         []string{"librarianops", "test-command", "-C", "/path/to/repo", "-v"},
			wantRepoName: "repo",
			wantWorkDir:  "/path/to/repo",
			wantVerbose:  true,
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			var (
				repoName, workDir string
				verbose           bool
				err               error
			)
			app.Commands[0].Action = func(ctx context.Context, cmd *cli.Command) error {
				repoName, workDir, verbose, err = parseRepoFlags(cmd)
				return err
			}

			if err := app.Run(t.Context(), testCase.args); err != nil {
				t.Fatalf("app.Run() error = %v, wantErr %v", err, false)
			}

			if diff := cmp.Diff(testCase.wantRepoName, repoName); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(testCase.wantWorkDir, workDir); diff != "" {
				t.Errorf("parseRepoFlags() workDir mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(testCase.wantVerbose, verbose); diff != "" {
				t.Errorf("parseRepoFlags() verbose mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestParseRepoFlags_Error(t *testing.T) {
	app := &cli.Command{
		Name: "librarianops",
		Commands: []*cli.Command{
			{
				Name: "test-command",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "C"},
					&cli.BoolFlag{Name: "v"},
				},
			},
		},
	}
	for _, testCase := range []struct {
		name string
		args []string
	}{
		{
			name: "no arguments",
			args: []string{"librarianops", "test-command"},
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			var (
				err error
			)
			app.Commands[0].Action = func(ctx context.Context, cmd *cli.Command) error {
				_, _, _, err := parseRepoFlags(cmd)
				return err
			}
			if err := app.Run(t.Context(), testCase.args); err == nil {
				t.Fatalf("app.Run() error = %v, wantErr %v", err, true)
			}
		})
	}
}
