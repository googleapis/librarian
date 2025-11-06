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
	"errors"
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
		name                   string
		filesToWrite           map[string]string
		changedFiles           []string
		newAndDeletedFiles     []string
		libraryState           *config.LibraryState
		setup                  func(dir string) error
		protoFileToGUIDs       map[string][]string
		checkUnexpectedChanges bool
		wantErrMsg             string
	}{
		{
			name: "unrelated changes",
			filesToWrite: map[string]string{
				"related.go":    "// some generated code\n// test-change-guid-123",
				"unrelated.txt": "some other content",
			},
			protoFileToGUIDs:       map[string][]string{"some.proto": {"guid-123"}},
			checkUnexpectedChanges: true,
			wantErrMsg:             "found unrelated file changes: unrelated.txt",
		},
		{
			name: "unrelated changes outside source root",
			filesToWrite: map[string]string{
				"src/related.go": "// some generated code\n// test-change-guid-123",
				"unrelated.txt":  "some other content",
			},
			protoFileToGUIDs:       map[string][]string{"some.proto": {"guid-123"}},
			libraryState:           &config.LibraryState{SourceRoots: []string{"src"}},
			checkUnexpectedChanges: true,
			wantErrMsg:             "", // No error, because unrelated.txt is ignored.
		},
		{
			name: "missing change",
			filesToWrite: map[string]string{
				"somefile.go": "some content",
			},
			protoFileToGUIDs:       map[string][]string{"some.proto": {"guid-not-found"}},
			checkUnexpectedChanges: true,
			wantErrMsg:             "produced no corresponding generated file changes",
		},
		{
			name: "success",
			filesToWrite: map[string]string{
				"some.go": "// some generated code\n// test-change-guid-123",
			},
			protoFileToGUIDs:       map[string][]string{"some.proto": {"guid-123"}},
			checkUnexpectedChanges: true,
			wantErrMsg:             "",
		},
		{
			name: "expected no file changes, but found changes",
			filesToWrite: map[string]string{
				"somefile.go": "some content",
			},
			newAndDeletedFiles:     []string{"somefile.go"},
			protoFileToGUIDs:       map[string][]string{},
			checkUnexpectedChanges: true,
			wantErrMsg:             "expected no new or deleted files, but found",
		},
		{
			name:         "deleted file is a valid change when not checking for unexpected changes",
			filesToWrite: map[string]string{
				// "deleted.go" is not written to the filesystem
			},
			changedFiles:           []string{"deleted.go"},
			newAndDeletedFiles:     []string{"deleted.go"},
			protoFileToGUIDs:       map[string][]string{},
			checkUnexpectedChanges: false, // This is the key
			wantErrMsg:             "",    // No error expected
		},
		{
			name: "unreadable file causes an error",
			filesToWrite: map[string]string{
				"unreadable.go": "some content",
			},
			changedFiles: []string{"unreadable.go"},
			setup: func(dir string) error {
				// Make the file unreadable
				return os.Chmod(filepath.Join(dir, "unreadable.go"), 0000)
			},
			protoFileToGUIDs:       map[string][]string{},
			checkUnexpectedChanges: true,
			wantErrMsg:             "failed to read changed file",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			tmpDir := t.TempDir()
			var filesConsideredChanged []string
			for filename, content := range test.filesToWrite {
				path := filepath.Join(tmpDir, filename)
				if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
					t.Fatalf("failed to create directory for %s: %v", filename, err)
				}
				if err := os.WriteFile(path, []byte(content), 0644); err != nil {
					t.Fatalf("failed to write file %s: %v", filename, err)
				}
				filesConsideredChanged = append(filesConsideredChanged, filename)
			}

			if test.setup != nil {
				if err := test.setup(tmpDir); err != nil {
					t.Fatalf("setup failed: %v", err)
				}
			}

			mockRepo := &MockRepository{
				Dir:                     tmpDir,
				ChangedFilesValue:       filesConsideredChanged,
				NewAndDeletedFilesValue: test.newAndDeletedFiles,
			}
			if test.changedFiles != nil {
				mockRepo.ChangedFilesValue = test.changedFiles
			}

			runner := &testGenerateRunner{
				repo:                   mockRepo,
				checkUnexpectedChanges: test.checkUnexpectedChanges,
			}
			libraryState := test.libraryState
			if libraryState == nil {
				// Default to the root directory if not specified.
				libraryState = &config.LibraryState{SourceRoots: []string{""}}
			}
			err := runner.validateGenerateTest(nil, test.protoFileToGUIDs, libraryState)

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

	// Common setup for all test cases
	const (
		protoPath      = "google/cloud/aiplatform/v1"
		protoFilename  = "prediction_service.proto"
		initialCommit  = "abcdef1234567890abcdef1234567890abcdef12"
		libraryID      = "google-cloud-aiplatform-v1"
		checkoutErrStr = "checkout error"
		addAllErrStr   = "add all error"
		commitErrStr   = "commit error"
	)
	protoWithServiceAndMessage := `
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
	protoWithServiceOnly := `
syntax = "proto3";
package google.cloud.aiplatform.v1;
service PredictionService {}
`
	protoWithMessageOnly := `
syntax = "proto3";
package google.cloud.aiplatform.v1;
message PredictRequest {}
`

	for _, test := range []struct {
		name            string
		libraryState    *config.LibraryState
		mockRepo        *MockRepository
		protoContent    string
		setup           func(repoDir string) error
		wantErrMsg      string
		wantCommitCalls int
		wantGUIDCount   int
	}{
		{
			name: "success with service and message",
			libraryState: &config.LibraryState{
				ID:                  libraryID,
				LastGeneratedCommit: initialCommit,
				APIs:                []*config.API{{Path: protoPath}},
			},
			mockRepo:        &MockRepository{},
			protoContent:    protoWithServiceAndMessage,
			wantErrMsg:      "",
			wantCommitCalls: 1,
			wantGUIDCount:   2,
		},
		{
			name: "success with service only",
			libraryState: &config.LibraryState{
				ID:                  libraryID,
				LastGeneratedCommit: initialCommit,
				APIs:                []*config.API{{Path: protoPath}},
			},
			mockRepo:        &MockRepository{},
			protoContent:    protoWithServiceOnly,
			wantErrMsg:      "",
			wantCommitCalls: 1,
			wantGUIDCount:   1,
		},
		{
			name: "success with message only",
			libraryState: &config.LibraryState{
				ID:                  libraryID,
				LastGeneratedCommit: initialCommit,
				APIs:                []*config.API{{Path: protoPath}},
			},
			mockRepo:        &MockRepository{},
			protoContent:    protoWithMessageOnly,
			wantErrMsg:      "",
			wantCommitCalls: 1,
			wantGUIDCount:   1,
		},
		{
			name: "missing last generated commit",
			libraryState: &config.LibraryState{
				ID:                  libraryID,
				LastGeneratedCommit: "", // Missing commit
				APIs:                []*config.API{{Path: protoPath}},
			},
			mockRepo:        &MockRepository{},
			protoContent:    protoWithServiceAndMessage,
			wantErrMsg:      "last_generated_commit is not set",
			wantCommitCalls: 0,
			wantGUIDCount:   0,
		},
		{
			name: "checkout commit error",
			libraryState: &config.LibraryState{
				ID:                  libraryID,
				LastGeneratedCommit: initialCommit,
				APIs:                []*config.API{{Path: protoPath}},
			},
			mockRepo: &MockRepository{
				CheckoutCommitAndCreateBranchError: errors.New(checkoutErrStr),
			},
			protoContent:    protoWithServiceAndMessage,
			wantErrMsg:      checkoutErrStr,
			wantCommitCalls: 0,
			wantGUIDCount:   0,
		},
		{
			name: "add all error",
			libraryState: &config.LibraryState{
				ID:                  libraryID,
				LastGeneratedCommit: initialCommit,
				APIs:                []*config.API{{Path: protoPath}},
			},
			mockRepo: &MockRepository{
				AddAllError: errors.New(addAllErrStr),
			},
			protoContent:    protoWithServiceAndMessage,
			wantErrMsg:      addAllErrStr,
			wantCommitCalls: 0,
			wantGUIDCount:   0,
		},
		{
			name: "commit error",
			libraryState: &config.LibraryState{
				ID:                  libraryID,
				LastGeneratedCommit: initialCommit,
				APIs:                []*config.API{{Path: protoPath}},
			},
			mockRepo: &MockRepository{
				CommitError: errors.New(commitErrStr),
			},
			protoContent:    protoWithServiceAndMessage,
			wantErrMsg:      commitErrStr,
			wantCommitCalls: 1, // Commit is still called
			wantGUIDCount:   0,
		},
		{
			name: "empty proto file",
			libraryState: &config.LibraryState{
				ID:                  libraryID,
				LastGeneratedCommit: initialCommit,
				APIs:                []*config.API{{Path: protoPath}},
			},
			mockRepo:        &MockRepository{},
			protoContent:    "", // Empty content
			wantErrMsg:      "configured to generate, but nothing to generate",
			wantCommitCalls: 0,
			wantGUIDCount:   0,
		},
		{
			name: "proto file with no insertion point",
			libraryState: &config.LibraryState{
				ID:                  libraryID,
				LastGeneratedCommit: initialCommit,
				APIs:                []*config.API{{Path: protoPath}},
			},
			mockRepo: &MockRepository{},
			protoContent: `
syntax = "proto3";
package google.cloud.aiplatform.v1;
import "google/api/annotations.proto";
// no message, service or enum
`,
			wantErrMsg:      "configured to generate, but nothing to generate",
			wantCommitCalls: 0,
			wantGUIDCount:   0,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			repoDir := t.TempDir()
			test.mockRepo.Dir = repoDir

			// Setup proto files
			fullProtoDir := filepath.Join(repoDir, protoPath)
			if err := os.MkdirAll(fullProtoDir, 0755); err != nil {
				t.Fatalf("os.MkdirAll() error = %v", err)
			}
			fullProtoPath := filepath.Join(fullProtoDir, protoFilename)
			if err := os.WriteFile(fullProtoPath, []byte(test.protoContent), 0644); err != nil {
				t.Fatalf("os.WriteFile() error = %v", err)
			}

			if test.setup != nil {
				if err := test.setup(repoDir); err != nil {
					t.Fatalf("setup() error = %v", err)
				}
			}

			runner := &testGenerateRunner{
				sourceRepo: test.mockRepo,
			}
			protoFileToGUIDs, err := runner.prepareForGenerateTest(test.libraryState, test.libraryState.ID)

			// Check for error
			if test.wantErrMsg != "" {
				if err == nil {
					t.Fatalf("prepareForGenerateTest() did not return an error, but one was expected")
				}
				if !strings.Contains(err.Error(), test.wantErrMsg) {
					t.Errorf("prepareForGenerateTest() returned error %q, want error containing %q", err.Error(), test.wantErrMsg)
				}
			} else if err != nil {
				t.Fatalf("prepareForGenerateTest() returned unexpected error: %v", err)
			}

			// Check if a GUID was expected to be injected.
			if test.wantGUIDCount > 0 {
				if len(protoFileToGUIDs) != 1 {
					t.Fatalf("len(protoFileToGUIDs) = %d, want 1", len(protoFileToGUIDs))
				}
				// Check the number of GUIDs injected into the single proto file.
				injectedGUIDs := protoFileToGUIDs[filepath.Join(protoPath, protoFilename)]
				if len(injectedGUIDs) != test.wantGUIDCount {
					t.Errorf("len(injectedGUIDs) = %d, want %d", len(injectedGUIDs), test.wantGUIDCount)
				}
			} else {
				if len(protoFileToGUIDs) != 0 {
					t.Fatalf("len(protoFileToGUIDs) = %d, want 0", len(protoFileToGUIDs))
				}
			}

			// Check if the expected number of commits were made.
			if test.mockRepo.CommitCalls != test.wantCommitCalls {
				t.Errorf("mockRepo.CommitCalls = %d, want %d", test.mockRepo.CommitCalls, test.wantCommitCalls)
			}
		})
	}
}

func TestTestGenerateRunnerRun(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name                   string
		state                  *config.LibrarianState
		librarianConfig        *config.LibrarianConfig
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
			name: "generate blocked library is skipped",
			state: &config.LibrarianState{
				Libraries: []*config.LibraryState{
					{
						ID:                  "blocked-lib",
						LastGeneratedCommit: "initial-commit",
						APIs:                []*config.API{{Path: "google/blocked/v1"}},
					},
				},
			},
			librarianConfig: &config.LibrarianConfig{
				Libraries: []*config.LibraryConfig{
					{
						LibraryID:       "blocked-lib",
						GenerateBlocked: true,
					},
				},
			},
			libraryID:  "blocked-lib",
			wantErrMsg: "", // No error expected, as it should be skipped
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
			wantErrMsg:  "generation command failed: generate error",
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
			wantErrMsg:             "produced no corresponding generated file changes",
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
			wantErrMsg:  "generation tests failed for 2 libraries",
		},
		{
			name: "multiple libraries, some skipped",
			state: &config.LibrarianState{
				Libraries: []*config.LibraryState{

					{
						ID:                  "lib1",
						LastGeneratedCommit: "initial-commit",
						APIs:                []*config.API{{Path: "google/lib1/v1"}},
					},
					{
						ID:                  "lib2",
						LastGeneratedCommit: "initial-commit",
						APIs:                []*config.API{{Path: "google/lib2/v1"}},
					},
				},
			},
			librarianConfig: &config.LibrarianConfig{
				Libraries: []*config.LibraryConfig{
					{
						LibraryID:       "lib2",
						GenerateBlocked: true, // lib2 will be skipped
					},
				},
			},
			libraryID:  "", // Run for all libraries
			wantErrMsg: "generation tests failed for 1 libraries",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
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
				librarianConfig:        test.librarianConfig,
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
		})
	}
}

func TestCleanup(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		name                         string
		sourceRepo                   *MockRepository
		repo                         *MockRepository
		branchesToDelete             []string
		wantDeleteLocalBranchesCalls int
		wantResetHardCalls           int
		wantErrMsg                   string
	}{
		{
			name:                         "successful cleanup",
			sourceRepo:                   &MockRepository{},
			repo:                         &MockRepository{},
			branchesToDelete:             []string{"test-branch-1"},
			wantDeleteLocalBranchesCalls: 1,
			wantResetHardCalls:           1,
		},
		{
			name: "DeleteLocalBranches returns error",
			sourceRepo: &MockRepository{
				DeleteLocalBranchesError: errors.New("delete branch error"),
			},
			repo:                         &MockRepository{},
			branchesToDelete:             []string{"test-branch-2"},
			wantDeleteLocalBranchesCalls: 1,
			wantResetHardCalls:           0, // ResetHard should not be called
			wantErrMsg:                   "failed to delete branch",
		},
		{
			name:       "ResetHard returns error",
			sourceRepo: &MockRepository{},
			repo: &MockRepository{
				ResetHardError: errors.New("reset hard error"),
			},
			branchesToDelete:             []string{"test-branch-3"},
			wantDeleteLocalBranchesCalls: 1,
			wantResetHardCalls:           1,
			wantErrMsg:                   "failed to reset repo",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			runner := &testGenerateRunner{
				sourceRepo:       test.sourceRepo,
				repo:             test.repo,
				branchesToDelete: test.branchesToDelete,
			}

			err := runner.cleanup()

			if test.wantErrMsg != "" {
				if err == nil {
					t.Fatalf("cleanup() did not return an error, but one was expected containing %q", test.wantErrMsg)
				}
				if !strings.Contains(err.Error(), test.wantErrMsg) {
					t.Errorf("cleanup() returned error %q, want error containing %q", err.Error(), test.wantErrMsg)
				}
			} else if err != nil {
				t.Fatalf("cleanup() returned unexpected error: %v", err)
			}

			if test.sourceRepo.DeleteLocalBranchesCalls != test.wantDeleteLocalBranchesCalls {
				t.Errorf("DeleteLocalBranches was called %d times, want %d", test.sourceRepo.DeleteLocalBranchesCalls, test.wantDeleteLocalBranchesCalls)
			}
			if test.repo.ResetHardCalls != test.wantResetHardCalls {
				t.Errorf("ResetHard was called %d times, want %d", test.repo.ResetHardCalls, test.wantResetHardCalls)
			}
		})
	}
}
