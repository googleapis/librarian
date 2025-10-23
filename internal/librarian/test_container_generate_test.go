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
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/googleapis/librarian/internal/config"
)

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
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			tmpDir := t.TempDir()
			var changedFiles []string
			for filename, content := range test.filesToWrite {
				path := filepath.Join(tmpDir, filename)
				if err := os.WriteFile(path, []byte(content), 0644); err != nil {
					t.Fatalf("failed to write file %s: %v", filename, err)
				}
				changedFiles = append(changedFiles, filename)
			}
			mockRepo := &MockRepository{
				Dir:                     tmpDir,
				ChangedFilesValue:       changedFiles,
				NewAndDeletedFilesValue: test.newAndDeletedFiles,
			}

			err := validateGenerateTest(nil, mockRepo, test.protoFileToGUID, true)

			if test.wantErrMsg != "" {
				if err == nil {
					t.Fatalf("validateGenerateTest() did not return an error, but one was expected")
				}
				if !strings.Contains(err.Error(), test.wantErrMsg) {
					t.Errorf("validateGenerateTest() returned error %q, want error containing %q", err.Error(), test.wantErrMsg)
				}
			} else if err != nil {
				t.Fatalf("validateGenerateTest() returned unexpected error: %v", err)
			}
		})
	}
}
func TestPrepareForGenerateTest(t *testing.T) {
	t.Parallel()
	// Create a temporary directory to act as a mock git repository.
	repoDir := t.TempDir()

	// Create a sample proto file within the mock repository.
	// This represents the API definition that will be processed.
	protoPath := "google/cloud/aiplatform/v1"
	protoFilename := "prediction_service.proto"
	fullProtoDir := filepath.Join(repoDir, protoPath)
	if err := os.MkdirAll(fullProtoDir, 0755); err != nil {
		t.Fatalf("os.MkdirAll() error = %v", err)
	}
	protoContent := `
syntax = "proto3";

package google.cloud.aiplatform.v1;

import "google/api/annotations.proto";

service PredictionService {
  rpc Predict(PredictRequest) returns (PredictResponse) {
    option (google.api.http) = {
      post: "/v1/{endpoint=projects/*/locations/*/endpoints/*}:predict"
      body: "*"
    };
  }
}

message PredictRequest {}
message PredictResponse {}
`
	fullProtoPath := filepath.Join(fullProtoDir, protoFilename)
	if err := os.WriteFile(fullProtoPath, []byte(protoContent), 0644); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}

	// Setup mock repository and library state configuration.
	initialCommit := "abcdef1234567890abcdef1234567890abcdef12"
	mockRepo := &MockRepository{
		Dir: repoDir,
	}
	libraryState := &config.LibraryState{
		ID:                  "google-cloud-aiplatform-v1",
		LastGeneratedCommit: initialCommit,
		APIs: []*config.API{
			{
				Path: protoPath,
			},
		},
	}
	libraryID := "google-cloud-aiplatform-v1"

	// Execute the function under test.
	protoFileToGUID, err := prepareForGenerateTest(libraryState, libraryID, mockRepo)
	if err != nil {
		t.Fatalf("prepareForGenerateTest() error = %v", err)
	}

	// Check that the function returned a map with the correct proto file and a GUID.
	if len(protoFileToGUID) != 1 {
		t.Fatalf("len(protoFileToGUID) = %d, want 1", len(protoFileToGUID))
	}

	var guid string
	for proto, g := range protoFileToGUID {
		if proto != filepath.Join(protoPath, protoFilename) {
			t.Errorf("protoFileToGUID key = %q, want %q", proto, filepath.Join(protoPath, protoFilename))
		}
		guid = g
	}

	// Check that the proto file was modified to include the GUID.
	newContent, err := os.ReadFile(fullProtoPath)
	if err != nil {
		t.Fatalf("os.ReadFile() error = %v", err)
	}
	if !strings.Contains(string(newContent), guid) {
		t.Errorf("proto file does not contain GUID %q", guid)
	}

	// Check that a new commit was made in the mock repository.
	if mockRepo.CommitCalls != 1 {
		t.Errorf("mockRepo.CommitCalls = %d, want 1", mockRepo.CommitCalls)
	}
}

func TestTestGenerateRunnerRun(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name                   string
		state                  *config.LibrarianState
		libraryID              string
		prepareErr             error
		generateErr            error
		validateErr            error
		wantErrMsg             string
		checkUnexpectedChanges bool
		repoChangedFiles       []string
	}{
		{
			name:       "library not found",
			state:      &config.LibrarianState{},
			libraryID:  "non-existent-library",
			wantErrMsg: "library \"non-existent-library\" not found in state",
		},
		{
			name: "prepareForGenerateTest error",
			state: &config.LibrarianState{
				Libraries: []*config.LibraryState{
					{
						ID:                  "google-cloud-aiplatform-v1",
						LastGeneratedCommit: "initial-commit",
						APIs: []*config.API{
							{
								Path: "google/cloud/aiplatform/v1",
							},
						},
					},
				},
			},
			libraryID:  "google-cloud-aiplatform-v1",
			prepareErr: fmt.Errorf("checkout error"),
			wantErrMsg: "checkout error",
		},
		{
			name: "generateSingleLibrary error",
			state: &config.LibrarianState{
				Libraries: []*config.LibraryState{
					{
						ID:                  "google-cloud-aiplatform-v1",
						LastGeneratedCommit: "initial-commit",
						APIs: []*config.API{
							{
								Path: "google/cloud/aiplatform/v1",
							},
						},
					},
				},
			},
			libraryID:   "google-cloud-aiplatform-v1",
			generateErr: fmt.Errorf("generate error"),
			wantErrMsg:  "generation failed: generate error",
		},
		{
			name: "validateGenerateTest error",
			state: &config.LibrarianState{
				Libraries: []*config.LibraryState{
					{
						ID:                  "google-cloud-aiplatform-v1",
						LastGeneratedCommit: "initial-commit",
						APIs: []*config.API{
							{
								Path: "google/cloud/aiplatform/v1",
							},
						},
					},
				},
			},
			libraryID:              "google-cloud-aiplatform-v1",
			checkUnexpectedChanges: true,
			repoChangedFiles:       []string{"unrelated.txt"},
			wantErrMsg:             "did not result in any generated file changes",
		},
		{
			name: "multiple library failures",
			state: &config.LibrarianState{
				Libraries: []*config.LibraryState{
					{
						ID:                  "lib1",
						LastGeneratedCommit: "initial-commit",
						APIs: []*config.API{
							{Path: "google/lib1/v1"},
						},
					},
					{
						ID:                  "lib2",
						LastGeneratedCommit: "initial-commit",
						APIs: []*config.API{
							{Path: "google/lib2/v1"},
						},
					},
				},
			},
			libraryID:   "", // Run for all libraries.
			generateErr: fmt.Errorf("generate error"),
			wantErrMsg:  "tests failed for libraries: lib1, lib2",
		},
	} {
		// 1. Setup the runner with mocked dependencies based on the test case.
		// Create a temporary directory to act as a mock git repository.
		repoDir := t.TempDir()

		// Create dummy proto files within the mock repository if the test case needs them.
		// This is needed because the prepare step searches for .proto files to modify.
		if test.state != nil {
			for _, lib := range test.state.Libraries {
				if len(lib.APIs) > 0 {
					protoPath := lib.APIs[0].Path
					protoFilename := "service.proto"
					fullProtoDir := filepath.Join(repoDir, protoPath)
					if err := os.MkdirAll(fullProtoDir, 0755); err != nil {
						t.Fatalf("os.MkdirAll() error = %v", err)
					}
					protoContent := "service MyService {}"
					if err := os.WriteFile(filepath.Join(fullProtoDir, protoFilename), []byte(protoContent), 0644); err != nil {
						t.Fatalf("os.WriteFile() error = %v", err)
					}
				}
			}
		}

		// Set up the mock repositories and clients with behavior defined by the test case.
		mockSourceRepo := &MockRepository{
			Dir:                                repoDir,
			CheckoutCommitAndCreateBranchError: test.prepareErr,
		}
		mockRepoDir := t.TempDir()
		for _, f := range test.repoChangedFiles {
			if err := os.WriteFile(filepath.Join(mockRepoDir, f), []byte("some content"), 0644); err != nil {
				t.Fatalf("failed to write file: %v", err)
			}
		}
		mockRepo := &MockRepository{
			Dir:               mockRepoDir,
			ChangedFilesValue: test.repoChangedFiles,
		}
		mockContainerClient := &mockContainerClient{
			generateErr: test.generateErr,
		}

		// Create testGenerateRunner with the mocked dependencies.
		runner := &testGenerateRunner{
			library:                test.libraryID,
			repo:                   mockRepo,
			sourceRepo:             mockSourceRepo,
			state:                  test.state,
			workRoot:               t.TempDir(),
			containerClient:        mockContainerClient,
			checkUnexpectedChanges: test.checkUnexpectedChanges,
		}

		// 2. Execute the run method.
		err := runner.run(context.Background())

		// 3. Assert the outcome.
		if test.wantErrMsg != "" {
			if err == nil {
				t.Fatal("runner.run() did not return an error, but one was expected")
			}
			if !strings.Contains(err.Error(), test.wantErrMsg) {
				t.Errorf("runner.run() returned error %q, want error containing %q", err.Error(), test.wantErrMsg)
			}
		} else if err != nil {
			t.Fatalf("runner.run() returned unexpected error: %v", err)
		}
	}
}
