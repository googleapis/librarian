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

package docuploader

import (
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/command"
	"github.com/googleapis/librarian/internal/testhelper"
)

func TestCreateArchive(t *testing.T) {
	var tarExe string
	opSys := runtime.GOOS
	switch opSys {
	case "darwin":
		tarExe = "gtar"
	default:
		tarExe = "tar"
	}
	testhelper.RequireCommand(t, tarExe)
	dir := t.TempDir()
	paths := []string{
		"other.txt",
		"otherdir/a.txt",
		"docs/a.txt",
		"docs/readme.txt",
		"docs/subdir/a.txt",
	}
	for _, path := range paths {
		fullPath := filepath.Join(dir, path)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(fullPath, []byte{}, 0644); err != nil {
			t.Fatal(err)
		}
		slog.Info("Created file", "file", fullPath)
	}

	sourceDir := filepath.Join(dir, "docs")
	targetFile := filepath.Join(dir, "target.tar.gz")

	if err := CreateArchive(t.Context(), tarExe, sourceDir, targetFile); err != nil {
		t.Fatal(err)
	}

	// Use tar again to find out what's in the archive...
	output, err := command.Output(t.Context(), "tar", "tzf", targetFile)
	if err != nil {
		t.Fatal(err)
	}
	gotFiles := strings.Split(strings.TrimSpace(output), "\n")
	slices.Sort(gotFiles)
	wantFiles := []string{
		"./",
		"a.txt",
		"readme.txt",
		"subdir/",
		"subdir/a.txt",
	}
	if diff := cmp.Diff(wantFiles, gotFiles); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestCreateArchive_Error(t *testing.T) {
	testhelper.RequireCommand(t, "tar")
	testhelper.RequireCommand(t, "echo")
	dir := t.TempDir()
	t.Parallel()

	for _, test := range []struct {
		name       string
		tarExe     string
		sourceDir  string
		targetFile string
		wantErr    error
	}{
		{
			name:       "tar error",
			tarExe:     "tar",
			sourceDir:  dir,
			targetFile: "/",
			wantErr:    errTarFailure,
		},
		{
			name:       "tar success but with unexpected output",
			tarExe:     "echo",
			sourceDir:  dir,
			targetFile: "/",
			wantErr:    errTarUnexpectedOutput,
		}} {
		t.Run(test.name, func(t *testing.T) {
			gotErr := CreateArchive(t.Context(), test.tarExe, test.sourceDir, test.targetFile)
			if !errors.Is(gotErr, test.wantErr) {
				t.Errorf("CreateArchive(%q, %q, %q) error = %v, wantErr %v", test.tarExe, test.sourceDir, test.targetFile, gotErr, test.wantErr)
			}
		})
	}
}
