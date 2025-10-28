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

func (r *testGenerateRunner) run(ctx context.Context) error {
	sourceRepoHead, err := r.sourceRepo.HeadHash()
	if err != nil {
		return fmt.Errorf("failed to get source repo head: %w", err)
	}

	if err := os.MkdirAll(filepath.Join(r.workRoot, "output"), 0755); err != nil {
		return fmt.Errorf("failed to create output directory under %s: %w", r.workRoot, err)
	}

	return r.runTests(ctx, sourceRepoHead)
}

func (r *testGenerateRunner) runTests(ctx context.Context, sourceRepoHead string) error {
	outputDir := filepath.Join(r.workRoot, "output")
	if r.library != "" {
		if err := r.runAndCleanupTest(ctx, r.library, sourceRepoHead, outputDir); err != nil {
			return fmt.Errorf("test failed for library %s: %w", r.library, err)
		}
		slog.Info("test succeeded for library", "library", r.library)
		return nil
	}

	slog.Info("running tests for all libraries")
	var failed []string
	for _, library := range r.state.Libraries {
		if err := r.runAndCleanupTest(ctx, library.ID, sourceRepoHead, outputDir); err != nil {
			slog.Error("test failed for library", "library", library.ID, "error", err)
			failed = append(failed, library.ID)
		} else {
			slog.Debug("test succeeded for library", "library", library.ID)
		}
	}
	if len(failed) > 0 {
		return fmt.Errorf("generation tests failed for %d libraries: %s", len(failed), strings.Join(failed, ", "))
	}
	slog.Info("generation tests succeeded for all libraries")
	return nil
}

func (r *testGenerateRunner) runAndCleanupTest(ctx context.Context, libraryID, sourceRepoHead, outputDir string) error {
	defer func() {
		slog.Debug("cleaning up after test", "library", libraryID)
		if err := r.sourceRepo.Checkout(sourceRepoHead); err != nil {
			slog.Error("failed to checkout source repo head during cleanup", "error", err)
		}
		if err := r.repo.ResetHard(); err != nil {
			slog.Error("failed to reset repo during cleanup", "error", err)
		}
	}()
	return r.testSingleLibrary(ctx, libraryID, outputDir)
}

// testSingleLibrary runs a generation test for a single library.
// It prepares the source repository, runs generation, and validates the output.
// It does NOT perform any cleanup or setup of output directories.
func (r *testGenerateRunner) testSingleLibrary(ctx context.Context, libraryID string, outputDir string) error {
	slog.Info("running generation test", "library", libraryID)
	libraryState := r.state.LibraryByID(libraryID)
	if libraryState == nil {
		return fmt.Errorf("library %q not found in state", libraryID)
	}
	protoFileToGUID, err := r.prepareForGenerateTest(libraryState, libraryID)
	if err != nil {
		return fmt.Errorf("failed in test preparing steps: %w", err)
	}

	// We capture the error here and pass it to the validation step.
	generateErr := generateSingleLibrary(ctx, r.containerClient, r.state, libraryState, r.repo, r.sourceRepo, outputDir)

	if err := r.validateGenerateTest(generateErr, protoFileToGUID); err != nil {
		return fmt.Errorf("failed in test validation steps: %w", err)
	}

	return nil
}

// prepareForGenerateTest sets up the source repository for a generation test. It
// checks out a new branch from the library's last generated commit, injects unique
// GUIDs as comments into the relevant proto files, and commits these temporary
// changes. It returns a map of the modified proto file paths to the GUIDs that
// were injected.
func (r *testGenerateRunner) prepareForGenerateTest(libraryState *config.LibraryState, libraryID string) (map[string]string, error) {
	if libraryState.LastGeneratedCommit == "" {
		return nil, fmt.Errorf("last_generated_commit is not set for library %q", libraryID)
	}

	branchName := "test-generate-" + uuid.New().String()
	if err := r.sourceRepo.CheckoutCommitAndCreateBranch(branchName, libraryState.LastGeneratedCommit); err != nil {
		return nil, err
	}

	protoFiles, err := findProtoFiles(libraryState, r.sourceRepo)
	if err != nil {
		return nil, fmt.Errorf("failed finding proto files: %w", err)
	}

	protoFileToGUID, err := injectTestGUIDsIntoProtoFiles(protoFiles, r.sourceRepo.GetDir())
	if err != nil {
		return nil, fmt.Errorf("failed to inject test GUIDs into proto files: %w", err)
	}

	if err := r.sourceRepo.AddAll(); err != nil {
		return nil, err
	}
	if err := r.sourceRepo.Commit("test(changes): temporary changes for generate test"); err != nil {
		return nil, err
	}

	return protoFileToGUID, nil
}

// findProtoFiles recursively finds all .proto files within the API paths specified in
// the library's state. If no .proto files are found, it returns an empty slice
// and a nil error. An error is returned if any of the file system walks fail.
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
func (r *testGenerateRunner) validateGenerateTest(generateErr error, protoFileToGUID map[string]string) error {
	slog.Debug("validating generation results")
	if generateErr != nil {
		return fmt.Errorf("the generation command failed: %w", generateErr)
	}

	// Get the list of uncommitted changed files from the worktree.
	changedFiles, err := r.repo.ChangedFiles()
	if err != nil {
		return fmt.Errorf("failed to get changed files from working tree: %w", err)
	}

	if r.checkUnexpectedChanges {
		newAndDeleted, err := r.repo.NewAndDeletedFiles()
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
	repoDir := r.repo.GetDir()

	for _, filePath := range changedFiles {
		fullPath := filepath.Join(repoDir, filePath)
		content, err := os.ReadFile(fullPath)
		if err != nil {
			if os.IsNotExist(err) { // The file was deleted, ignoring if not checkUnexpectedChanges
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
			return fmt.Errorf("change in proto file %s (GUID %s) produced no corresponding generated file changes", protoFile, guid)
		}
	}
	slog.Debug("validation succeeded: all proto changes resulted in generated file changes")

	if r.checkUnexpectedChanges {
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

	slog.Debug("all generation validation passed")
	return nil
}
