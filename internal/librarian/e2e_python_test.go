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

package librarian

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/testhelper"
)

func TestPythonWorkflow(t *testing.T) {
	testhelper.RequireCommand(t, "git")
	testhelper.RequireCommand(t, "protoc")
	testhelper.RequireCommand(t, "protoc-gen-python_gapic")
	testhelper.RequireCommand(t, "python3")
	requirePythonModule(t, "synthtool")

	for _, test := range []struct {
		name  string
		steps [][]string
		want  wantLibrary
	}{
		{
			name: "generate",
			steps: [][]string{
				{"librarian", "generate", testLibraryName},
			},
			want: wantLibrary{
				libraryName: testLibraryName,
			},
		},
		{
			name: "generate all",
			steps: [][]string{
				{"librarian", "generate", "--all"},
			},
			want: wantLibrary{
				libraryName: testLibraryName,
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			setupPythonRepo(t)
			for i, args := range test.steps {
				if err := Run(t.Context(), args...); err != nil {
					t.Fatalf("step %d (%v) failed: %v", i, args, err)
				}
			}
			verifyLibrary(t, test.want)
		})
	}
}

func requirePythonModule(t *testing.T, module string) {
	t.Helper()
	cmd := exec.Command("python3", "-c", fmt.Sprintf("import %s", module))
	if err := cmd.Run(); err != nil {
		t.Skipf("skipping test because Python module %s is not installed", module)
	}
}

func setupPythonRepo(t *testing.T) *config.Config {
	t.Helper()

	cfg := testConfig(t, languagePython, googleapisTestDir(t))
	cfg.Libraries = []*config.Library{
		{
			Name:    testLibraryName,
			Version: "1.0.0",
			Output:  testOutputDir,
			Channels: []*config.Channel{
				{Path: "google/cloud/secretmanager/v1"},
			},
		},
	}
	setupRepo(t, cfg, func(t *testing.T) {
		if err := os.WriteFile("README.md", []byte("# Test Repo"), 0644); err != nil {
			t.Fatal(err)
		}
		for _, lib := range cfg.Libraries {
			if err := os.MkdirAll(lib.Output, 0755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(lib.Output, "README.rst"), []byte(""), 0644); err != nil {
				t.Fatal(err)
			}
		}
	})
	return cfg
}
