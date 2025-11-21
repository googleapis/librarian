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

package legacylibrarian

import (
	"strings"
	"testing"

	"github.com/googleapis/librarian/internal/legacylibrarian/legacycli"
	"github.com/googleapis/librarian/internal/legacylibrarian/legacyconfig"
)

func TestLibrarianAction(t *testing.T) {
	for _, test := range []struct {
		name string
		fn   func() *legacycli.Command
	}{
		{
			name: "generate",
			fn:   newCmdGenerate,
		},
		{
			name: "init",
			fn:   newCmdStage,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			testActionConfig(t, test.fn())
		})
	}
}

// testActionConfig tests the execution flow for each Command.Action. The
// functionality of the Config methods called inside these Actions are tested
// separately in internal/legacyconfig.
func testActionConfig(t *testing.T, cmd *legacycli.Command) {
	t.Helper()
	for _, test := range []struct {
		cfg     *legacyconfig.Config
		wantErr string
	}{
		{
			cfg: &legacyconfig.Config{
				WorkRoot: t.TempDir(),
			},
			wantErr: "repo flag not specified",
		},
		{
			cfg: &legacyconfig.Config{
				WorkRoot: t.TempDir(),
				Repo:     "myrepo",
			},
			wantErr: "repository does not exist",
		},
		{
			cfg: &legacyconfig.Config{
				WorkRoot:       t.TempDir(),
				Repo:           "myrepo",
				LibraryVersion: "1.0.0",
			},
			wantErr: "specified library version without library id",
		},
		{
			cfg: &legacyconfig.Config{
				WorkRoot: t.TempDir(),
				Repo:     "https://github.com/googleapis/language-repo",
			},
			wantErr: "remote branch is required when cloning",
		},
	} {
		t.Run(test.wantErr, func(t *testing.T) {
			cmd.Config = test.cfg
			err := cmd.Action(t.Context(), cmd)
			if err == nil {
				t.Fatal("expected an error")
			}
			if !strings.Contains(err.Error(), test.wantErr) {
				t.Errorf("error mismatch, want: %q, got: %q", test.wantErr, err.Error())
			}
		})
	}
}
