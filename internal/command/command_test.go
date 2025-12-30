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
	"fmt"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
)

func TestRun(t *testing.T) {
	if err := Run(t.Context(), "go", "version"); err != nil {
		t.Fatal(err)
	}
}

func TestRunError(t *testing.T) {
	err := Run(t.Context(), "go", "invalid-subcommand-bad-bad-bad")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "invalid-subcommand-bad-bad-bad") {
		t.Errorf("error should mention the invalid subcommand, got: %v", err)
	}
}

func TestRunWithEnv_SetsAndVerifiesVariable(t *testing.T) {
	ctx := t.Context()
	const (
		name  = "LIBRARIAN_TEST_VAR"
		value = "value"
	)
	err := RunWithEnv(ctx, map[string]string{name: value},
		"sh", "-c", fmt.Sprintf("test \"$%s\" = \"%s\"", name, value))
	if err != nil {
		t.Fatalf("RunWithEnv() = %v, want %v", err, nil)
	}
}

func TestRunWithEnv_VariableNotSetFailsValidation(t *testing.T) {
	ctx := t.Context()
	const (
		name  = "LIBRARIAN_TEST_VAR"
		value = "value"
	)
	err := RunWithEnv(ctx, map[string]string{}, "sh", "-c", fmt.Sprintf("test \"$%s\" = \"%s\"", name, value))
	if err == nil {
		t.Fatalf("RunWithEnv() = %v, want non-nil", err)
	}
}

func TestGetExecutablePath(t *testing.T) {
	tests := []struct {
		name           string
		releaseConfig  *config.Release
		executableName string
		want           string
	}{
		{
			name: "Preinstalled tool found",
			releaseConfig: &config.Release{
				Preinstalled: map[string]string{
					"cargo": "/usr/bin/cargo",
					"git":   "/usr/bin/git",
				},
			},
			executableName: "cargo",
			want:           "/usr/bin/cargo",
		},
		{
			name: "Preinstalled tool not found",
			releaseConfig: &config.Release{
				Preinstalled: map[string]string{
					"git": "/usr/bin/git",
				},
			},
			executableName: "cargo",
			want:           "cargo",
		},
		{
			name:           "No preinstalled section",
			releaseConfig:  &config.Release{},
			executableName: "cargo",
			want:           "cargo",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := GetExecutablePath(test.releaseConfig.Preinstalled, test.executableName)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

