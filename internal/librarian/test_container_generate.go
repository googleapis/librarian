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
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/gitrepo"
)

type testGenerateRunner struct {
	library                string
	repo                   gitrepo.Repository
	sourceRepo             gitrepo.Repository
	state                  *config.LibrarianState
	workRoot               string
	containerClient        ContainerClient
	checkUnexpectedChanges bool
}

func newTestGenerateRunner(cfg *config.Config) (*testGenerateRunner, error) {
	runner, err := newCommandRunner(cfg)
	if err != nil {
		return nil, err
	}
	return &testGenerateRunner{
		library:                cfg.Library,
		repo:                   runner.repo,
		sourceRepo:             runner.sourceRepo,
		state:                  runner.state,
		workRoot:               runner.workRoot,
		containerClient:        runner.containerClient,
		checkUnexpectedChanges: cfg.CheckUnexpectedChanges,
	}, nil
}

func (r *testGenerateRunner) run(ctx context.Context) error {
	// remember repo head for cleanup after test
	sourceRepoHead, err := r.sourceRepo.HeadHash()
	if err != nil {
		return fmt.Errorf("failed to get source repo head: %w", err)
	}

	outputDir := filepath.Join(r.workRoot, "output")
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to make output directory, %s: %w", outputDir, err)
	}

	if r.library != "" {
		return r.runAndCleanupTest(ctx, r.library, sourceRepoHead, outputDir)
	}

	var failed []string
	for _, library := range r.state.Libraries {
		if err := r.runAndCleanupTest(ctx, library.ID, sourceRepoHead, outputDir); err != nil {
			slog.Error("test failed for library", "library", library.ID, "error", err)
			failed = append(failed, library.ID)
		}
	}
	if len(failed) > 0 {
		return fmt.Errorf("%d test(s) failed for libraries: %s", len(failed), strings.Join(failed, ", "))
	}
	return nil
}

func (r *testGenerateRunner) runAndCleanupTest(ctx context.Context, libraryID, sourceRepoHead, outputDir string) error {
	defer func() {
		slog.Info("cleaning up after test", "library", libraryID)
		if err := r.sourceRepo.Checkout(sourceRepoHead); err != nil {
			slog.Error("failed to checkout source repo head during cleanup", "error", err)
		}
		if err := r.repo.ResetHard(); err != nil {
			slog.Error("failed to reset repo during cleanup", "error", err)
		}
	}()
	return testSingleLibrary(ctx, libraryID, r.repo, r.sourceRepo, r.state, r.containerClient, r.checkUnexpectedChanges, outputDir)
}

// testSingleLibrary runs a generation test for a single library.
// It prepares the source repository, runs generation, and validates the output.
// It does NOT perform any cleanup or setup of output directories.
func testSingleLibrary(ctx context.Context, libraryID string, repo gitrepo.Repository, sourceRepo gitrepo.Repository, state *config.LibrarianState, containerClient ContainerClient, checkUnexpectedChanges bool, outputDir string) error {
	slog.Info("running test for", "library", libraryID)
	libraryState := state.LibraryByID(libraryID)
	if libraryState == nil {
		return fmt.Errorf("library %q not found in state", libraryID)
	}
	protoFileToGUID, err := prepareForGenerateTest(libraryState, libraryID, sourceRepo)
	if err != nil {
		return err
	}

	// We capture the error here and pass it to the validation step.
	generateErr := generateSingleLibrary(ctx, containerClient, state, libraryState, repo, sourceRepo, outputDir)

	if err := validateGenerateTest(generateErr, repo, protoFileToGUID, checkUnexpectedChanges); err != nil {
		return err
	}

	return nil
}

// prepareForGenerateTest sets up the source repository for a generation test. It
// checks out a new branch from the library's last generated commit, injects unique
// GUIDs as comments into the relevant proto files, and commits these temporary
// changes. It returns a map of the modified proto file paths to the GUIDs that
// were injected.
func prepareForGenerateTest(libraryState *config.LibraryState, libraryID string, sourceRepo gitrepo.Repository) (map[string]string, error) {
	if libraryState.LastGeneratedCommit == "" {
		return nil, fmt.Errorf("last_generated_commit is not set for library %q", libraryID)
	}

	branchName := "test-generate-" + uuid.New().String()
	if err := sourceRepo.CheckoutCommitAndCreateBranch(branchName, libraryState.LastGeneratedCommit); err != nil {
		return nil, err
	}

	protoFiles, err := findProtoFiles(libraryState, sourceRepo)
	if err != nil {
		return nil, err
	}

	protoFileToGUID, err := injectTestGUIDsIntoProtoFiles(protoFiles, sourceRepo.GetDir())
	if err != nil {
		return nil, err
	}

	if err := sourceRepo.AddAll(); err != nil {
		return nil, err
	}
	if err := sourceRepo.Commit("test(changes): temporary changes for generate test"); err != nil {
		return nil, err
	}

	return protoFileToGUID, nil
}

// findProtoFiles recursively searches for all files with the .proto extension within
// the API paths specified in the library's state.
func findProtoFiles(libraryState *config.LibraryState, repo gitrepo.Repository) ([]string, error) {
	var protoFiles []string
	repoPath := repo.GetDir()
	for _, api := range libraryState.APIs {
		root := filepath.Join(repoPath, api.Path)
		err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() || !strings.HasSuffix(info.Name(), ".proto") {
				return nil
			}
			relPath, err := filepath.Rel(repoPath, path)
			if err != nil {
				return err
			}
			protoFiles = append(protoFiles, relPath)
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	return protoFiles, nil
}

// injectTestGUIDsIntoProtoFiles injects a unique GUID into each one proto file
// provided. It returns a map of file paths to the GUIDs that were successfully injected.
func injectTestGUIDsIntoProtoFiles(protoFiles []string, repoPath string) (map[string]string, error) {
	protoFileToGUID := make(map[string]string)
	for _, protoFile := range protoFiles {
		guid, err := injectGUIDIntoProto(filepath.Join(repoPath, protoFile))
		if err != nil {
			return nil, fmt.Errorf("failed to inject GUID into %s: %w", protoFile, err)
		}
		if guid != "" {
			protoFileToGUID[protoFile] = guid
		}
	}
	return protoFileToGUID, nil
}

// injectGUIDIntoProto adds a unique GUID comment to a single proto file to simulate
// a change. It finds a suitable insertion point (e.g., before a message, enum, or
// service definition) and writes the modified content back to the file. It returns
// the GUID that was injected or an empty string if no suitable insertion point was
// found.
func injectGUIDIntoProto(absPath string) (string, error) {
	content, err := os.ReadFile(absPath)
	if err != nil {
		return "", err
	}
	lines := strings.Split(string(content), "\n")
	if len(content) == 0 {
		return "", nil
	}

	insertionLine := findProtoInsertionLine(lines)
	if insertionLine == -1 {
		// No suitable line found to inject the comment.
		return "", nil
	}

	guid := uuid.New().String()
	comment := "// test-change-" + guid
	var newLines []string
	newLines = append(newLines, lines[:insertionLine]...)
	newLines = append(newLines, comment)
	newLines = append(newLines, lines[insertionLine:]...)

	output := strings.Join(newLines, "\n")
	if err := os.WriteFile(absPath, []byte(output), 0644); err != nil {
		return "", err
	}
	return guid, nil
}

// findProtoInsertionLine determines the best line number to inject a test comment
// in a proto file. It searches for the first occurrence of a top-level message,
// enum, or service definition.
func findProtoInsertionLine(lines []string) int {
	searchTerms := []string{"message ", "enum ", "service "}
	for i, line := range lines {
		for _, term := range searchTerms {
			if strings.HasPrefix(strings.TrimSpace(line), term) {
				return i
			}
		}
	}
	return -1
}

// validateGenerateTest checks the results of the generation process. It ensures
// that the generation command did not fail, that every injected proto change
// resulted in a corresponding change in the generated code, and optionally
// verifies that no other unexpected files were added, deleted, or modified.
func validateGenerateTest(generateErr error, repo gitrepo.Repository, protoFileToGUID map[string]string, checkUnexpectedChanges bool) error {
	slog.Info("running test validation for library")
	if generateErr != nil {
		return fmt.Errorf("generation failed: %w", generateErr)
	}

	// Get the list of uncommitted changed files from the worktree.
	changedFiles, err := repo.ChangedFiles()
	if err != nil {
		return fmt.Errorf("failed to get changed files from worktree: %w", err)
	}

	if checkUnexpectedChanges {
		newAndDeleted, err := repo.NewAndDeletedFiles()
		if err != nil {
			return fmt.Errorf("failed to get new and deleted files: %w", err)
		}
		if len(newAndDeleted) > 0 {
			return fmt.Errorf("expected no new or deleted files, but found: %s", strings.Join(newAndDeleted, ", "))
		}
		slog.Debug("validation succeeded: no new or deleted files")
	}

	guidsToFind := make(map[string]bool)
	for _, guid := range protoFileToGUID {
		guidsToFind[guid] = false
	}
	filesWithGUIDs := make(map[string]bool)
	repoDir := repo.GetDir()

	for _, filePath := range changedFiles {
		fullPath := filepath.Join(repoDir, filePath)
		content, err := os.ReadFile(fullPath)
		if err != nil {
			if os.IsNotExist(err) { // The file was deleted, which is a valid change.
				continue
			}
			return fmt.Errorf("failed to read changed file %s: %w", filePath, err)
		}

		contentStr := string(content)
		wasModifiedByTest := false
		for guid := range guidsToFind {
			if strings.Contains(contentStr, guid) {
				guidsToFind[guid] = true
				wasModifiedByTest = true
			}
		}
		if wasModifiedByTest {
			filesWithGUIDs[filePath] = true
		}
	}

	for protoFile, guid := range protoFileToGUID {
		if !guidsToFind[guid] {
			return fmt.Errorf("proto change in %s (GUID %s) did not result in any generated file changes", protoFile, guid)
		}
	}
	slog.Debug("validation succeeded: all proto changes resulted in generated file changes")

	if checkUnexpectedChanges {
		var unrelatedChanges []string
		for _, filePath := range changedFiles {
			if !filesWithGUIDs[filePath] {
				unrelatedChanges = append(unrelatedChanges, filePath)
			}
		}
		if len(unrelatedChanges) > 0 {
			return fmt.Errorf("found unrelated file changes: %s", strings.Join(unrelatedChanges, ", "))
		}
		slog.Debug("validation succeeded: no unrelated file changes found")
	}

	slog.Info("all validation checks passed")
	return nil
}
