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

// Usage text rules:
// - <arg> = required argument
// - [arg] = optional argument
// - <arg...> = one or more required arguments
// - Flags are included in the usage line if they are required
// - Use --all for all libraries
//
// Note: We do not test the 'migrate' tool here because it is not built with
// the urfave/cli library and does not follow these usage patterns.

import (
	"bytes"
	"os/exec"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestLibrarianUsage(t *testing.T) {
	const bin = "github.com/googleapis/librarian/cmd/librarian"
	for _, test := range []struct {
		desc string
		args []string
		want string
	}{
		{"root", nil, "librarian [command]"},
		{"add", []string{"add"}, "librarian add <apis...>"},
		{"generate", []string{"generate"}, "librarian generate <library>"},
		{"bump", []string{"bump"}, "librarian bump <library>"},
		{"tidy", []string{"tidy"}, "librarian tidy"},
		{"update", []string{"update"}, "librarian update <sources...>"},
		{"version", []string{"version"}, "librarian version"},
		{"publish", []string{"publish"}, "librarian publish"},
		{"tag", []string{"tag"}, "librarian tag"},
	} {
		t.Run(test.desc, func(t *testing.T) {
			got := runUsage(t, bin, test.args)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestLibrarianopsUsage(t *testing.T) {
	const bin = "github.com/googleapis/librarian/cmd/librarianops"
	for _, test := range []struct {
		desc string
		args []string
		want string
	}{
		{"root", nil, "librarianops [command]"},
		{"generate", []string{"generate"}, "librarianops generate [<repo> | -C <dir>]"},
	} {
		t.Run(test.desc, func(t *testing.T) {
			got := runUsage(t, bin, test.args)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestSurferUsage(t *testing.T) {
	const bin = "github.com/googleapis/librarian/cmd/surfer"
	for _, test := range []struct {
		desc string
		args []string
		want string
	}{
		{"root", nil, "surfer [command]"},
		{"generate", []string{"generate"}, "surfer generate <path to gcloud.yaml> --googleapis <path>"},
	} {
		t.Run(test.desc, func(t *testing.T) {
			got := runUsage(t, bin, test.args)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestToolUsage(t *testing.T) {
	for _, test := range []struct {
		desc string
		bin  string
		args []string
		want string
	}{
		{"import-configs root", "github.com/googleapis/librarian/tool/cmd/importconfigs", nil, "import-configs [command]"},
		{"import-metadata", "github.com/googleapis/librarian/tool/cmd/importmetadata", nil, "import-metadata --python-repo <path> --librarian-repo <path>"},
	} {
		t.Run(test.desc, func(t *testing.T) {
			got := runUsage(t, test.bin, test.args)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

// runUsage executes the binary with the given args and the appropriate help flag,
// returning the captured usage string.
func runUsage(t *testing.T, bin string, args []string) string {
	t.Helper()
	var stdout bytes.Buffer
	fullArgs := append([]string{"run", bin}, args...)
	fullArgs = append(fullArgs, "--help")
	cmd := exec.Command("go", fullArgs...)
	cmd.Stdout = &stdout
	cmd.Stderr = &stdout // Some help might go to stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("go %v failed: %v\nOutput: %s", fullArgs, err, stdout.String())
	}
	return captureUsage(t, stdout.String())
}

// captureUsage extracts the usage line from the command output. It looks for
// a line containing "USAGE:" and returns the following line.
func captureUsage(t *testing.T, output string) string {
	t.Helper()
	lines := strings.Split(output, "\n")
	for i, line := range lines {
		if strings.Contains(strings.ToUpper(line), "USAGE:") {
			if i+1 < len(lines) {
				return strings.TrimSpace(lines[i+1])
			}
		}
	}
	return ""
}
