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

package librarian

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/googleapis/librarian/internal/config"
)

func TestNewTestGenerateRunner(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name       string
		cfg        *config.Config
		wantErr    bool
		wantErrMsg string
	}{
		{
			name: "valid config",
			cfg: &config.Config{
				Repo:     newTestGitRepo(t).GetDir(),
				WorkRoot: t.TempDir(),
				Image:    "gcr.io/test/test-image",
			},
		},
		{
			name: "missing image",
			cfg: &config.Config{
				Repo:     "https://github.com/googleapis/librarian.git",
				WorkRoot: t.TempDir(),
			},
			wantErr: true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			_, err := newTestGenerateRunner(test.cfg)
			if test.wantErr {
				if err == nil {
					t.Fatalf("newTestGenerateRunner() error = %v, wantErr %v", err, test.wantErr)
				}
				if !strings.Contains(err.Error(), test.wantErrMsg) {
					t.Fatalf("want error message: %s, got: %s", test.wantErrMsg, err.Error())
				}
				return
			}
		})
	}
}

func TestValidateGenerateTest(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name               string
		filesToWrite       map[string]string
		newAndDeletedFiles []string
		protoFileToGUID    map[string]string
		wantErrMsg         string
	}{
		{
			name: "unrelated changes",
			filesToWrite: map[string]string{
				"related.go":    "// some generated code\n// test-change-guid-123",
				"unrelated.txt": "some other content",
			},
			protoFileToGUID: map[string]string{"some.proto": "guid-123"},
			wantErrMsg:      "found unrelated file changes: unrelated.txt",
		},
		{
			name: "missing change",
			filesToWrite: map[string]string{
				"somefile.go": "some content",
			},
			protoFileToGUID: map[string]string{"some.proto": "guid-not-found"},
			wantErrMsg:      "did not result in any generated file changes",
		},
		{
			name: "success",
			filesToWrite: map[string]string{
				"some.go": "// some generated code\n// test-change-guid-123",
			},
			protoFileToGUID: map[string]string{"some.proto": "guid-123"},
			wantErrMsg:      "",
		},
		{
			name: "expected no file changes, but found changes",
			filesToWrite: map[string]string{
				"somefile.go": "some content",
			},
			newAndDeletedFiles: []string{"somefile.go"},
			protoFileToGUID:    map[string]string{},
			wantErrMsg:         "expected no new or deleted files, but found",
		},
	} {
		tt := test
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tmpDir := t.TempDir()
			var changedFiles []string
			for filename, content := range tt.filesToWrite {
				path := filepath.Join(tmpDir, filename)
				if err := os.WriteFile(path, []byte(content), 0644); err != nil {
					t.Fatalf("failed to write file %s: %v", filename, err)
				}
				changedFiles = append(changedFiles, filename)
			}
			mockRepo := &MockRepository{
				Dir:                     tmpDir,
				ChangedFilesValue:       changedFiles,
				NewAndDeletedFilesValue: tt.newAndDeletedFiles,
			}

			err := validateGenerateTest(nil, mockRepo, tt.protoFileToGUID)

			if tt.wantErrMsg != "" {
				if err == nil {
					t.Fatalf("validateGenerateTest() did not return an error, but one was expected")
				}
				if !strings.Contains(err.Error(), tt.wantErrMsg) {
					t.Errorf("validateGenerateTest() returned error %q, want error containing %q", err.Error(), tt.wantErrMsg)
				}
			} else if err != nil {
				t.Fatalf("validateGenerateTest() returned unexpected error: %v", err)
			}
		})
	}
}
