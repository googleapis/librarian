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

package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/fetch"
)

func TestRun(t *testing.T) {
	// Save global variables and restore them after tests.
	oldFetchSource := fetchSource
	oldFetchSourceWithCommit := fetchSourceWithCommit
	defer func() {
		fetchSource = oldFetchSource
		fetchSourceWithCommit = oldFetchSourceWithCommit
	}()

	tmpDir := t.TempDir()
	repoPath := filepath.Join(tmpDir, "google-cloud-node")
	if err := os.MkdirAll(filepath.Join(repoPath, "packages"), 0755); err != nil {
		t.Fatal(err)
	}

	for _, test := range []struct {
		name       string
		args       []string
		wantSource *config.Source
		wantErr    bool
	}{
		{
			name: "googleapis flag sets local source",
			args: []string{"-googleapis", tmpDir, repoPath},
			wantSource: &config.Source{
				Dir: tmpDir,
			},
		},
		{
			name: "commit flag sets commit source",
			args: []string{"-commit", "abcd123", repoPath},
			wantSource: &config.Source{
				Commit: "abcd123",
				Dir:    "/fake/path/abcd123",
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			r := &runner{
				fetchSource: func(ctx context.Context) (*config.Source, error) {
					return test.wantSource, nil
				},
				fetchSourceWithCommit: func(ctx context.Context, endpoints *fetch.Endpoints, commitish string) (*config.Source, error) {
					return test.wantSource, nil
				},
			}

			err := r.run(t.Context(), test.args)
			if (err != nil) != test.wantErr {
				t.Fatalf("run() error = %v, wantErr %v", err, test.wantErr)
			}

			if err == nil {
				// The run method should have updated the globals.
				gotSource, _ := fetchSource(t.Context())
				if diff := cmp.Diff(test.wantSource, gotSource, cmpopts.IgnoreUnexported(config.Source{})); diff != "" {
					t.Errorf("fetchSource mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}
}
