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

package gem

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
)

func TestInstall(t *testing.T) {
	tmpDir := t.TempDir()
	stubLogPath := filepath.Join(tmpDir, "gem_invocations.log")
	stubContent := fmt.Sprintf(`#!/bin/sh
echo "gem $@" >> %q
`, stubLogPath)
	stubDir := filepath.Join(tmpDir, "bin")
	if err := os.MkdirAll(stubDir, 0755); err != nil {
		t.Fatal(err)
	}
	stubPath := filepath.Join(stubDir, "gem")
	if err := os.WriteFile(stubPath, []byte(stubContent), 0755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", stubDir)
	binDir := "/tmp/ruby_tools/bin"
	installDir := "/tmp/ruby_tools"
	for _, test := range []struct {
		name    string
		tools   []*config.GemTool
		wantLog string
	}{
		{
			name: "install gem with version",
			tools: []*config.GemTool{
				{Name: "rubocop", Version: "1.50.0"},
			},
			wantLog: "gem install rubocop -v 1.50.0 --bindir /tmp/ruby_tools/bin --install-dir /tmp/ruby_tools --no-document",
		},
		{
			name: "install multiple gems",
			tools: []*config.GemTool{
				{Name: "rubocop", Version: "1.50.0"},
				{Name: "rake", Version: "13.0.6"},
			},
			wantLog: `gem install rubocop -v 1.50.0 --bindir /tmp/ruby_tools/bin --install-dir /tmp/ruby_tools --no-document
gem install rake -v 13.0.6 --bindir /tmp/ruby_tools/bin --install-dir /tmp/ruby_tools --no-document`,
		},
		{
			name:    "empty or nil tools",
			tools:   []*config.GemTool{},
			wantLog: "",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			_ = os.Remove(stubLogPath)
			if err := Install(t.Context(), test.tools, binDir, installDir); err != nil {
				t.Fatal(err)
			}
			data, _ := os.ReadFile(stubLogPath)
			gotLog := strings.TrimSpace(string(data))
			if diff := cmp.Diff(test.wantLog, gotLog); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestInstall_Error(t *testing.T) {
	tmpDir := t.TempDir()
	stubDir := filepath.Join(tmpDir, "bin")
	if err := os.MkdirAll(stubDir, 0755); err != nil {
		t.Fatal(err)
	}
	stubPath := filepath.Join(stubDir, "gem")
	if err := os.WriteFile(stubPath, []byte("#!/bin/sh\nexit 1\n"), 0755); err != nil {
		t.Fatal(err)
	}

	for _, test := range []struct {
		name    string
		tools   []*config.GemTool
		setup   func(t *testing.T)
		wantErr error
	}{
		{
			name: "gem command fails",
			tools: []*config.GemTool{
				{Name: "failgem", Version: "1.0.0"},
			},
			setup: func(t *testing.T) {
				t.Setenv("PATH", stubDir)
			},
			wantErr: errInstall,
		},
		{
			name: "missing name",
			tools: []*config.GemTool{
				{Name: "", Version: "1.0.0"},
			},
			wantErr: errInvalidGem,
		},
		{
			name: "missing version",
			tools: []*config.GemTool{
				{Name: "rubocop", Version: ""},
			},
			wantErr: errInvalidGem,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			if test.setup != nil {
				test.setup(t)
			}
			err := Install(t.Context(), test.tools, "", "")
			if !errors.Is(err, test.wantErr) {
				t.Errorf("Install() error = %v, wantErr = %v", err, test.wantErr)
			}
		})
	}
}
