// Copyright 2025 Google LLC
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

package legacylibrarian

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/googleapis/librarian/internal/legacylibrarian/legacyconfig"
	"github.com/googleapis/librarian/internal/legacylibrarian/legacygitrepo"
)

var errGenerateBlocked = errors.New("generation is blocked for library")

type testGenerateRunner struct {
	library                string
	repo                   legacygitrepo.Repository
	sourceRepo             legacygitrepo.Repository
	state                  *legacyconfig.LibrarianState
	librarianConfig        *legacyconfig.LibrarianConfig
	workRoot               string
	containerClient        ContainerClient
	checkUnexpectedChanges bool
	branchesToDelete       []string
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

// runTests executes the generation test for one or all libraries.
//
// If a single library is specified, it runs the test for that library. If the
// test is skipped (due to errGenerateBlocked), it logs and exits successfully.
// On failure, it returns an error, preserving the generated files for
// debugging. On success, it cleans up the temporary work directory.
//
// If no specific library is given, it runs tests for all libraries. It keeps
// track of failed and skipped tests. If any tests fail, it returns an error
// listing the failed libraries, preserving the generated files. If all tests
// pass or are skipped, it cleans up the work directory.
func (r *testGenerateRunner) runTests(ctx context.Context, sourceRepoHead string) error {
	outputDir := filepath.Join(r.workRoot, "test-output")
	if err := os.Mkdir(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to make output directory, %s: %w", outputDir, err)
	}
	if r.library != "" {
		err := r.testSingleLibrary(ctx, r.library, sourceRepoHead, outputDir)
		if errors.Is(err, errGenerateBlocked) {
			slog.Info("test skipped for library due to generate_blocked", "library", r.library)
			return nil
		}
		if err != nil {
			return fmt.Errorf("test failed for library %s, keeping changes for debugging: %w", r.library, err)
		}
		slog.Info("test succeeded for library", "library", r.library)
		if err := r.cleanup(); err != nil {
			return err
		}
		return nil
	}

	slog.Info("running tests for all libraries")
	var failed []string
	var skippedCount int
	for _, library := range r.state.Libraries {
		err := r.testSingleLibrary(ctx, library.ID, sourceRepoHead, outputDir)
		if errors.Is(err, errGenerateBlocked) {
			slog.Info("test skipped for library due to generate_blocked", "library", library.ID)
			skippedCount++
			continue
		}
		if err != nil {
			slog.Error("test failed for library", "library", library.ID, "error", err)
			failed = append(failed, library.ID)
		} else {
			slog.Debug("test succeeded for library", "library", library.ID)
		}
	}
	if len(failed) > 0 {
		slog.Info("some tests failed, keeping changes for debugging", "branches", r.branchesToDelete)
		return fmt.Errorf("generation tests failed for %d libraries: %s", len(failed), strings.Join(failed, ", "))
	}

	if skippedCount > 0 {
		slog.Info("generation tests completed", "skipped_libraries", skippedCount)
	} else {
		slog.Info("generation tests succeeded for all libraries")
	}
	if err := r.cleanup(); err != nil {
		return err
	}
	return nil
}

func (r *testGenerateRunner) cleanup() error {
	// Delete branches created during test in source repo
	if err := r.sourceRepo.DeleteLocalBranches(r.branchesToDelete); err != nil {
		return fmt.Errorf("failed to delete branch during cleanup: %w", err)
	}
	// Reset the code repo worktree to discard temp test changes at success
	if err := r.repo.ResetHard(); err != nil {
		return fmt.Errorf("failed to reset repo during cleanup: %w", err)
	}
	return nil
}

// testSingleLibrary runs a generation test for a single library.
// The test performs the following steps:
//
// 1.  Prepares the source repository:
//   - Checks out the `last_generated_commit` from the source repository.
//   - Injects unique GUIDs as comments into the first `message`/`enum` definition
//     and the first `service` definition found in each proto file to simulate a change.
//   - Commits these temporary changes to a new branch.
//
// 2.  Runs the `generate` command for the specified library.
// 3.  Validates the output:
//   - Ensures that the generation command did not fail.
//   - Verifies that every injected GUID appears in the generated output,
//     confirming that the simulated changes triggered a corresponding update.
//   - Optionally, checks for any unexpected file additions, deletions, or modifications.
//
// 4.  Cleans up the source repository by checking out the original commit.
//
// Note: This function does not delete the temporary branch created in the source
// repository or reset the worktree in the code repository; these cleanup actions
// are handled by the caller.
func (r *testGenerateRunner) testSingleLibrary(ctx context.Context, libraryID, sourceRepoHead, outputDir string) error {
	defer func() {
		slog.Debug("resetting source repo to original commit", "library", libraryID)
		if err := r.sourceRepo.Checkout(sourceRepoHead); err != nil {
			slog.Error("failed to checkout source repo head during cleanup", "error", err)
		}
	}()
	slog.Info("running generation test", "library", libraryID)
	libraryState := r.state.LibraryByID(libraryID)
	if libraryState == nil {
		return fmt.Errorf("library %q not found in state", libraryID)
	}

	if r.librarianConfig.IsGenerationBlocked(libraryID) {
		return errGenerateBlocked
	}

	protoFileToGUIDs, err := r.prepareForGenerateTest(libraryState, libraryID)
	if err != nil {
		return fmt.Errorf("failed in test preparing steps: %w", err)
	}

	// We capture the error here and pass it to the validation step.
	generateErr := generateSingleLibrary(ctx, r.containerClient, r.state, libraryState, r.repo, r.sourceRepo, outputDir)

	if err := r.validateGenerateTest(generateErr, protoFileToGUIDs, libraryState); err != nil {
		return fmt.Errorf("failed in test validation steps: %w", err)
	}

	return nil
}

// prepareForGenerateTest sets up the source repository for a generation test. It
// checks out a new branch from the library's last generated commit, injects unique
// GUIDs as comments into the relevant proto files, and commits these temporary
// changes. It returns a map of the modified proto file paths to the slice of
// GUIDs that were injected.
func (r *testGenerateRunner) prepareForGenerateTest(libraryState *legacyconfig.LibraryState, libraryID string) (map[string][]string, error) {
	if libraryState.LastGeneratedCommit == "" {
		return nil, fmt.Errorf("last_generated_commit is not set for library %q", libraryID)
	}

	branchName := "test-generate-" + uuid.New().String()
	if err := r.sourceRepo.CheckoutCommitAndCreateBranch(branchName, libraryState.LastGeneratedCommit); err != nil {
		return nil, err
	}
	r.branchesToDelete = append(r.branchesToDelete, branchName)

	protoFiles, err := findProtoFiles(libraryState, r.sourceRepo)
	if err != nil {
		return nil, fmt.Errorf("failed finding proto files: %w", err)
	}

	protoFileToGUIDs, err := injectTestGUIDsIntoProtoFiles(protoFiles, r.sourceRepo.GetDir())
	if err != nil {
		return nil, fmt.Errorf("failed to inject test GUIDs into proto files: %w", err)
	}

	if len(protoFileToGUIDs) == 0 {
		return nil, fmt.Errorf("library %q configured to generate, but nothing to generate", libraryID)
	}

	if err := r.sourceRepo.AddAll(); err != nil {
		return nil, err
	}
	if err := r.sourceRepo.Commit("test(changes): temporary changes for generate test"); err != nil {
		return nil, err
	}

	return protoFileToGUIDs, nil
}

// findProtoFiles recursively finds all .proto files within the API paths specified in
// the library's state. If no .proto files are found, it returns an empty slice
// and a nil error. An error is returned if any of the file system walks fail.
func findProtoFiles(libraryState *legacyconfig.LibraryState, repo legacygitrepo.Repository) ([]string, error) {
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

// injectTestGUIDsIntoProtoFiles injects unique GUIDs into each proto file
// provided. It returns a map of file paths to the GUIDs that were successfully
// injected.
func injectTestGUIDsIntoProtoFiles(protoFiles []string, repoPath string) (map[string][]string, error) {
	protoFileToGUIDs := make(map[string][]string)
	for _, protoFile := range protoFiles {
		guids, err := injectGUIDsIntoProto(filepath.Join(repoPath, protoFile))
		if err != nil {
			return nil, fmt.Errorf("failed to inject GUID into %s: %w", protoFile, err)
		}
		if len(guids) > 0 {
			protoFileToGUIDs[protoFile] = guids
		}
	}
	return protoFileToGUIDs, nil
}

// injectGUIDsIntoProto adds unique GUID comments to a single proto file to
// simulate a change. It finds suitable insertion points (before a message, enum,
// or service definition) and writes the modified content back to the file. It
// returns the GUIDs that were injected.
func injectGUIDsIntoProto(absPath string) ([]string, error) {
	content, err := os.ReadFile(absPath)
	if err != nil {
		return nil, err
	}
	lines := strings.Split(string(content), "\n")
	if len(content) == 0 {
		return nil, nil
	}

	commentsByLine := make(map[int][]string)
	var injectedGUIDs []string
	// find the first occurrence of message/enum, and the first occurrence of service separately
	// because they usually correspond to separate generated files.
	if guid := prepareGUIDInjection(lines, []string{"message ", "enum "}, commentsByLine); guid != "" {
		injectedGUIDs = append(injectedGUIDs, guid)
	}
	if guid := prepareGUIDInjection(lines, []string{"service "}, commentsByLine); guid != "" {
		injectedGUIDs = append(injectedGUIDs, guid)
	}

	if len(injectedGUIDs) == 0 {
		return nil, nil
	}

	var newLines []string
	for i, line := range lines {
		if comments, ok := commentsByLine[i]; ok {
			newLines = append(newLines, comments...)
		}
		newLines = append(newLines, line)
	}

	output := strings.Join(newLines, "\n")
	if err := os.WriteFile(absPath, []byte(output), 0644); err != nil {
		return nil, err
	}
	return injectedGUIDs, nil
}

// prepareGUIDInjection finds the first occurrence of any of the provided search
// terms and, if found, injects a new GUID comment into the commentsByLine map and
// returns the generated GUID.
func prepareGUIDInjection(lines []string, searchTerms []string, commentsByLine map[int][]string) string {
	insertionLine := findProtoInsertionLine(lines, searchTerms)
	if insertionLine != -1 {
		guid := uuid.New().String()
		comment := "// test-change-" + guid
		commentsByLine[insertionLine] = append(commentsByLine[insertionLine], comment)
		return guid
	}
	return ""
}

// findProtoInsertionLine determines the best line number to inject a test comment
// in a proto file. It searches for the first occurrence of a top-level definition
// matching one of the search terms.
func findProtoInsertionLine(lines []string, searchTerms []string) int {
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
func (r *testGenerateRunner) validateGenerateTest(generateErr error, protoFileToGUIDs map[string][]string, libraryState *legacyconfig.LibraryState) error {
	slog.Debug("validating generation results")
	if generateErr != nil {
		return fmt.Errorf("the generation command failed: %w", generateErr)
	}

	// Get the list of uncommitted changed files from the worktree.
	changedFiles, err := r.repo.ChangedFiles()
	if err != nil {
		return fmt.Errorf("failed to get changed files from working tree: %w", err)
	}

	changedFiles = filterFilesBySourceRoots(changedFiles, libraryState.SourceRoots)

	if r.checkUnexpectedChanges {
		newAndDeleted, err := r.repo.NewAndDeletedFiles()
		if err != nil {
			return fmt.Errorf("failed to get new and deleted files: %w", err)
		}
		newAndDeleted = filterFilesBySourceRoots(newAndDeleted, libraryState.SourceRoots)
		if len(newAndDeleted) > 0 {
			return fmt.Errorf("expected no new or deleted files, but found: %s", strings.Join(newAndDeleted, ", "))
		}
		slog.Debug("validation succeeded: no new or deleted files")
	}

	guidsToFind := make(map[string]bool)
	for _, guids := range protoFileToGUIDs {
		for _, guid := range guids {
			guidsToFind[guid] = false
		}
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

	for protoFile, guids := range protoFileToGUIDs {
		for _, guid := range guids {
			if !guidsToFind[guid] {
				return fmt.Errorf("change in proto file %s (GUID %s) produced no corresponding generated file changes", protoFile, guid)
			}
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

func filterFilesBySourceRoots(files []string, sourceRoots []string) []string {
	var filteredFiles []string
	for _, file := range files {
		for _, sourceRoot := range sourceRoots {
			relPath, err := filepath.Rel(sourceRoot, file)
			if err == nil && !strings.HasPrefix(relPath, "..") {
				filteredFiles = append(filteredFiles, file)
				break
			}
		}
	}
	return filteredFiles
}
