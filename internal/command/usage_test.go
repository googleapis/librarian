// Copyright 2025 Google LLC
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

package command

import (
	"bytes"
	"os/exec"
	"strings"
	"testing"
)

func TestLibrarianUsage(t *testing.T) {
	const bin = "github.com/googleapis/librarian/cmd/librarian"
	tests := []struct {
		desc string
		args []string
		want string
	}{
		{desc: "root", want: "librarian [command]"},
		{desc: "add", args: []string{"add"}, want: "librarian add <apis...>"},
		{desc: "generate", args: []string{"generate"}, want: "librarian generate <library>"},
		{desc: "bump", args: []string{"bump"}, want: "librarian bump <library>"},
		{desc: "tidy", args: []string{"tidy"}, want: "librarian tidy [path]"},
		{desc: "update", args: []string{"update"}, want: "librarian update <sources...>"},
		{desc: "version", args: []string{"version"}, want: "librarian version"},
		{desc: "publish", args: []string{"publish"}, want: "librarian publish"},
		{desc: "tag", args: []string{"tag"}, want: "librarian tag"},
	}
	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			checkUsage(t, bin, tc.args, tc.want)
		})
	}
}

func TestLibrarianopsUsage(t *testing.T) {
	const bin = "github.com/googleapis/librarian/cmd/librarianops"
	tests := []struct {
		desc string
		args []string
		want string
	}{
		{desc: "root", want: "librarianops [command]"},
		{desc: "generate", args: []string{"generate"}, want: "librarianops generate [<repo> | -C <dir>]"},
	}
	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			checkUsage(t, bin, tc.args, tc.want)
		})
	}
}

func TestSurferUsage(t *testing.T) {
	const bin = "github.com/googleapis/librarian/cmd/surfer"
	tests := []struct {
		desc string
		args []string
		want string
	}{
		{desc: "root", want: "surfer [command]"},
		{desc: "generate", args: []string{"generate"}, want: "surfer generate <path to gcloud.yaml> --googleapis <path>"},
	}
	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			checkUsage(t, bin, tc.args, tc.want)
		})
	}
}

func TestToolUsage(t *testing.T) {
	tests := []struct {
		desc string
		bin  string
		args []string
		want string
	}{
		{desc: "import-configs root", bin: "github.com/googleapis/librarian/tool/cmd/importconfigs", want: "import-configs [command]"},
		{desc: "import-configs update-transports", bin: "github.com/googleapis/librarian/tool/cmd/importconfigs", args: []string{"update-transports"}, want: "import-configs update-transports --googleapis <path>"},
		{desc: "import-metadata", bin: "github.com/googleapis/librarian/tool/cmd/importmetadata", want: "import-metadata --python-repo <path> --librarian-repo <path>"},
		{desc: "migrate", bin: "github.com/googleapis/librarian/tool/cmd/migrate", want: "Usage of migrate:"},
	}
	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			checkUsage(t, tc.bin, tc.args, tc.want)
		})
	}
}

func checkUsage(t *testing.T, bin string, args []string, want string) {
	t.Helper()
	var stdout bytes.Buffer
	helpFlag := "--help"
	if strings.Contains(bin, "migrate") {
		helpFlag = "-help"
	}
	fullArgs := append([]string{"run", bin}, args...)
	fullArgs = append(fullArgs, helpFlag)
	cmd := exec.Command("go", fullArgs...)
	cmd.Stdout = &stdout
	cmd.Stderr = &stdout // Some help might go to stderr
	if err := cmd.Run(); err != nil && !strings.Contains(bin, "migrate") {
		t.Fatalf("go %v failed: %v\nOutput: %s", fullArgs, err, stdout.String())
	}
	got := captureUsage(stdout.String())
	if !strings.Contains(got, want) {
		t.Errorf("Usage mismatch\ngot:  %q\nwant: %q", got, want)
	}
}

func captureUsage(output string) string {
	lines := strings.Split(output, "\n")
	for i, line := range lines {
		if strings.Contains(strings.ToUpper(line), "USAGE:") {
			if i+1 < len(lines) {
				return strings.TrimSpace(lines[i+1])
			}
		}
	}
	// Fallback for commands like migrate that might just print usage immediately
	if len(lines) > 0 {
		return strings.TrimSpace(lines[0])
	}
	return ""
}
