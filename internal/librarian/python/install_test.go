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

package python

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/testhelper"
	"github.com/googleapis/librarian/internal/tool/pip"
)

func TestInstall(t *testing.T) {
	setupStubPip(t, "#!/bin/sh\n")

	tools := &config.Tools{
		Pip: []*config.PipTool{
			{Name: "ruff", Version: "0.14.14"},
		},
	}
	if err := Install(t.Context(), tools); err != nil {
		t.Fatal(err)
	}
}

func TestInstall_Error(t *testing.T) {
	for _, test := range []struct {
		name    string
		tools   *config.Tools
		setup   func(t *testing.T)
		wantErr error
	}{
		{
			name:    "nil tools config",
			tools:   nil,
			wantErr: ErrNoToolsSpecified,
		},
		{
			name:    "empty pip tools",
			tools:   &config.Tools{},
			wantErr: ErrNoToolsSpecified,
		},
		{
			name: "local path not found",
			tools: &config.Tools{
				Pip: []*config.PipTool{
					{LocalPath: "/path/does/not/exist"},
				},
			},
			wantErr: pip.ErrLocalPathNotFound,
		},
		{
			name: "pip execution fails",
			tools: &config.Tools{
				Pip: []*config.PipTool{
					{Name: "invalid-tool"},
				},
			},
			setup: func(t *testing.T) {
				setupStubPip(t, "#!/bin/sh\nexit 1\n")
			},
			wantErr: pip.ErrInstall,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			if test.setup != nil {
				test.setup(t)
			}
			err := Install(t.Context(), test.tools)
			if !errors.Is(err, test.wantErr) {
				t.Fatalf("Install() error = %v, wantErr = %v", err, test.wantErr)
			}
		})
	}
}

func setupStubPip(t *testing.T, script string) {
	t.Helper()
	bin := t.TempDir()
	testhelper.WriteExecutable(t, filepath.Join(bin, "pip"), script)
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
}
