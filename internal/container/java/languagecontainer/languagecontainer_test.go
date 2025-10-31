// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package languagecontainer

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/container/java/languagecontainer/generate"
	"github.com/googleapis/librarian/internal/container/java/languagecontainer/release"
	"github.com/googleapis/librarian/internal/container/java/message"
)

func TestRun(t *testing.T) {
	tmpDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmpDir, "release-stage-request.json"), []byte("{}"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "generate-request.json"), []byte("{}"), 0644); err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name     string
		args     []string
		wantCode int
		wantErr  bool
	}{
		{
			name:     "unknown command",
			args:     []string{"foo"},
			wantCode: 1,
		},
		{
			name:     "build command",
			args:     []string{"build"},
			wantCode: 1, // Not implemented yet
		},
		{
			name:     "configure command",
			args:     []string{"configure"},
			wantCode: 1, // Not implemented yet
		},
		{
			name:     "generate command with default flags",
			args:     []string{"generate"},
			wantCode: 1, // Fails because default /librarian does not exist.
		},
		{
			name:     "generate command success",
			args:     []string{"generate", "-librarian", tmpDir},
			wantCode: 0,
		},
		{
			name:     "generate command failure",
			args:     []string{"generate", "-librarian", tmpDir},
			wantCode: 1,
			wantErr:  true,
		},
		{
			name:     "release-stage command success",
			args:     []string{"release-stage", "-librarian", tmpDir},
			wantCode: 0,
		},
		{
			name:     "release-stage command failure",
			args:     []string{"release-stage", "-librarian", tmpDir},
			wantCode: 1,
			wantErr:  true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			container := LanguageContainer{
				Generate: func(ctx context.Context, c *generate.Config) error {
					if tt.wantErr {
						return os.ErrNotExist
					}
					return nil
				},
				ReleaseStage: func(ctx context.Context, c *release.Config) (*message.ReleaseStageResponse, error) {
					if tt.wantErr {
						return nil, os.ErrNotExist
					}
					return &message.ReleaseStageResponse{}, nil
				},
			}
			if gotCode := Run(tt.args, &container); gotCode != tt.wantCode {
				t.Errorf("Run() = %v, want %v", gotCode, tt.wantCode)
			}
		})
	}
}

func TestRun_noArgs(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("The code did not panic")
		}
	}()
	Run([]string{}, &LanguageContainer{})
}

func TestRun_ReleaseStageWritesResponse(t *testing.T) {
	tmpDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmpDir, "release-stage-request.json"), []byte("{}"), 0644); err != nil {
		t.Fatal(err)
	}
	args := []string{"release-stage", "-librarian", tmpDir}
	want := &message.ReleaseStageResponse{Error: "test error"}
	container := LanguageContainer{
		ReleaseStage: func(ctx context.Context, c *release.Config) (*message.ReleaseStageResponse, error) {
			return want, nil
		},
	}

	if code := Run(args, &container); code != 0 {
		t.Errorf("Run() = %v, want 0", code)
	}

	responsePath := filepath.Join(tmpDir, "release-stage-response.json")
	bytes, err := os.ReadFile(responsePath)
	if err != nil {
		t.Fatal(err)
	}
	got := &message.ReleaseStageResponse{}
	if err := json.Unmarshal(bytes, got); err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("response mismatch (-want +got):\n%s", diff)
	}
}

func TestRun_ReleaseStageReadsContextArgs(t *testing.T) {
	tmpDir := t.TempDir()
	librarianDir := filepath.Join(tmpDir, "librarian")
	if err := os.Mkdir(librarianDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(librarianDir, "release-stage-request.json"), []byte("{}"), 0644); err != nil {
		t.Fatal(err)
	}
	repoDir := filepath.Join(tmpDir, "repo")
	if err := os.Mkdir(repoDir, 0755); err != nil {
		t.Fatal(err)
	}
	outputDir := filepath.Join(tmpDir, "output")
	if err := os.Mkdir(outputDir, 0755); err != nil {
		t.Fatal(err)
	}
	args := []string{"release-stage", "-librarian", librarianDir, "-repo", repoDir, "-output", outputDir}
	var gotConfig *release.Config
	container := LanguageContainer{
		ReleaseStage: func(ctx context.Context, c *release.Config) (*message.ReleaseStageResponse, error) {
			gotConfig = c
			return &message.ReleaseStageResponse{}, nil
		},
	}
	if code := Run(args, &container); code != 0 {
		t.Errorf("Run() = %v, want 0", code)
	}
	if got, want := gotConfig.Context.LibrarianDir, librarianDir; got != want {
		t.Errorf("gotConfig.Context.LibrarianDir = %q, want %q", got, want)
	}
	if got, want := gotConfig.Context.RepoDir, repoDir; got != want {
		t.Errorf("gotConfig.Context.RepoDir = %q, want %q", got, want)
	}
	if got, want := gotConfig.Context.OutputDir, outputDir; got != want {
		t.Errorf("gotConfig.Context.OutputDir = %q, want %q", got, want)
	}
}

func TestRun_GenerateReadsContextArgs(t *testing.T) {
	tmpDir := t.TempDir()
	librarianDir := filepath.Join(tmpDir, "librarian")
	if err := os.Mkdir(librarianDir, 0755); err != nil {
		t.Fatal(err)
	}
	// generate.NewConfig reads generate-request.json.
	if err := os.WriteFile(filepath.Join(librarianDir, "generate-request.json"), []byte("{}"), 0644); err != nil {
		t.Fatal(err)
	}
	inputDir := filepath.Join(tmpDir, "input")
	if err := os.Mkdir(inputDir, 0755); err != nil {
		t.Fatal(err)
	}
	outputDir := filepath.Join(tmpDir, "output")
	if err := os.Mkdir(outputDir, 0755); err != nil {
		t.Fatal(err)
	}
	sourceDir := filepath.Join(tmpDir, "source")
	if err := os.Mkdir(sourceDir, 0755); err != nil {
		t.Fatal(err)
	}
	args := []string{"generate", "-librarian", librarianDir, "-input", inputDir, "-output", outputDir, "-source", sourceDir}
	var gotConfig *generate.Config
	container := LanguageContainer{
		Generate: func(ctx context.Context, c *generate.Config) error {
			gotConfig = c
			return nil
		},
	}
	if code := Run(args, &container); code != 0 {
		t.Errorf("Run() = %v, want 0", code)
	}
	if got, want := gotConfig.Context.LibrarianDir, librarianDir; got != want {
		t.Errorf("gotConfig.Context.LibrarianDir = %q, want %q", got, want)
	}
	if got, want := gotConfig.Context.InputDir, inputDir; got != want {
		t.Errorf("gotConfig.Context.InputDir = %q, want %q", got, want)
	}
	if got, want := gotConfig.Context.OutputDir, outputDir; got != want {
		t.Errorf("gotConfig.Context.OutputDir = %q, want %q", got, want)
	}
	if got, want := gotConfig.Context.SourceDir, sourceDir; got != want {
		t.Errorf("gotConfig.Context.SourceDir = %q, want %q", got, want)
	}
}

func TestRun_unimplementedCommands(t *testing.T) {
	tests := []struct {
		name      string
		args      []string
		container *LanguageContainer
	}{
		{
			name: "generate is nil",
			args: []string{"generate"},
			container: &LanguageContainer{
				ReleaseStage: func(context.Context, *release.Config) (*message.ReleaseStageResponse, error) {
					return nil, nil
				},
			},
		},
		{
			name: "release-stage is nil",
			args: []string{"release-stage"},
			container: &LanguageContainer{
				Generate: func(context.Context, *generate.Config) error {
					return nil
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if gotCode := Run(tt.args, tt.container); gotCode != 1 {
				t.Errorf("Run() = %v, want 1", gotCode)
			}
		})
	}
}
