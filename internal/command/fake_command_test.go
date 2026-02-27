// Copyright 2026 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package command

import (
	"fmt"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestFakeCommander(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name        string
		fakeResults map[string]FakeResult
		runCmd      string
		runArgs     []string
		wantCmds    [][]string
		wantOut     string
	}{
		{
			name:     "single command defaults to success",
			runCmd:   "echo",
			runArgs:  []string{"hello world"},
			wantCmds: [][]string{{"echo", "hello world"}},
			wantOut:  "",
		},
		{
			name:     "multiple arguments default to success",
			runCmd:   "ls",
			runArgs:  []string{"-l", "-a"},
			wantCmds: [][]string{{"ls", "-l", "-a"}},
			wantOut:  "",
		},
		{
			name: "successfully returns configured stdout",
			fakeResults: map[string]FakeResult{
				FormatCmd("gh", "pr", "view"): {Stdout: `{"state": "OPEN"}`},
			},
			runCmd:   "gh",
			runArgs:  []string{"pr", "view"},
			wantCmds: [][]string{{"gh", "pr", "view"}},
			wantOut:  `{"state": "OPEN"}`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			faker := &FakeCommander{FakeResults: tc.fakeResults}
			ctx := faker.InjectContext(t.Context())

			// Using Output instead of Run to verify stdout interception
			out, err := Output(ctx, tc.runCmd, tc.runArgs...)
			if err != nil {
				t.Fatal(err)
			}

			if out != tc.wantOut {
				t.Errorf("Output() = %q, want %q", out, tc.wantOut)
			}

			if diff := cmp.Diff(tc.wantCmds, faker.GotCommands); diff != "" {
				t.Errorf("GotCommands mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestFakeCommander_Failure(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name        string
		fakeResults map[string]FakeResult
		defaultRes  FakeResult
		runCmd      string
		runArgs     []string
		wantErr     string
		wantCmds    [][]string
	}{
		{
			name: "specific command error triggers",
			fakeResults: map[string]FakeResult{
				FormatCmd("git", "clone", "repo"): {Error: fmt.Errorf("repository not found")},
			},
			runCmd:   "git",
			runArgs:  []string{"clone", "repo"},
			wantErr:  "repository not found",
			wantCmds: [][]string{{"git", "clone", "repo"}},
		},
		{
			name: "default error triggers when specific is missing",
			fakeResults: map[string]FakeResult{
				FormatCmd("git", "clone", "repo"): {Error: fmt.Errorf("repository not found")},
			},
			defaultRes: FakeResult{Error: fmt.Errorf("network offline")},
			runCmd:     "curl",
			runArgs:    []string{"http://example.com"},
			wantErr:    "network offline",
			wantCmds:   [][]string{{"curl", "http://example.com"}},
		},
		{
			name: "explicit exit code and stderr",
			fakeResults: map[string]FakeResult{
				FormatCmd("make", "build"): {ExitCode: 2, Stderr: "compilation failed"},
			},
			runCmd:   "make",
			runArgs:  []string{"build"},
			wantErr:  "compilation failed",
			wantCmds: [][]string{{"make", "build"}},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			faker := &FakeCommander{
				FakeResults: tc.fakeResults,
				Default:     tc.defaultRes,
			}

			ctx := faker.InjectContext(t.Context())

			err := Run(ctx, tc.runCmd, tc.runArgs...)
			if err == nil {
				t.Fatalf("Run() unexpectedly succeeded, want err containing %q", tc.wantErr)
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("Run() error = %v, want err containing %q", err, tc.wantErr)
			}

			if diff := cmp.Diff(tc.wantCmds, faker.GotCommands); diff != "" {
				t.Errorf("GotCommands mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestFakeCommander_FallbackToReal(t *testing.T) {
	t.Parallel()
	faker := &FakeCommander{}
	_ = faker.InjectContext(t.Context())
	// Use t.Context() directly to bypass the faker's context injection logic
	out, err := Output(t.Context(), "echo", "real fallback")
	if err != nil {
		t.Fatalf("Output() unexpectedly failed during real fallback: %v", err)
	}
	if !strings.Contains(out, "real fallback") {
		t.Errorf("Output() = %q, want it to contain %q", out, "real fallback")
	}
}
