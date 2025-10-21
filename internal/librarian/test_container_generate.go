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
	"github.com/googleapis/librarian/internal/docker"
	"github.com/googleapis/librarian/internal/gitrepo"
)

type testGenerateRunner struct {
	branch                 string
	image                  string
	library                string
	repo                   gitrepo.Repository
	sourceRepo             gitrepo.Repository
	state                  *config.LibrarianState
	librarianConfig        *config.LibrarianConfig
	workRoot               string
	containerClient        ContainerClient
	ghClient               GitHubClient
	checkUnexpectedChanges bool
}

func newTestGenerateRunner(cfg *config.Config) (*testGenerateRunner, error) {
	runner, err := newCommandRunner(cfg)
	if err != nil {
		return nil, err
	}
	return &testGenerateRunner{
		branch:                 cfg.Branch,
		image:                  runner.image,
		library:                cfg.Library,
		repo:                   runner.repo,
		sourceRepo:             runner.sourceRepo,
		state:                  runner.state,
		librarianConfig:        runner.librarianConfig,
		workRoot:               runner.workRoot,
		containerClient:        runner.containerClient,
		ghClient:               runner.ghClient,
		checkUnexpectedChanges: cfg.CheckUnexpectedChanges,
	}, nil
}

func (r *testGenerateRunner) run(ctx context.Context) error {
	slog.Debug("prepare for test", "library", r.library)
	protoFileToGUID, err := prepareForGenerateTest(r.state, r.library, r.sourceRepo)
	if err != nil {
		return err
	}

	outputDir := filepath.Join(r.workRoot, "output")
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to make output directory, %s: %w", outputDir, err)
	}

	// We capture the error here and pass it to the validation step.
	slog.Debug("run generate", "library", r.library)
	_, generateErr := r.runGenerateCommand(ctx, r.library, outputDir)

	slog.Debug("validate", "library", r.library)
	if err := validateGenerateTest(generateErr, r.repo, protoFileToGUID, r.checkUnexpectedChanges); err != nil {
		return err
	}

	return nil
}

func validateGenerateTest(generateErr error, repo gitrepo.Repository, protoFileToGUID map[string]string, checkUnexpectedChanges bool) error {
	slog.Info("running validation")
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
		slog.Info("validation succeeded: no new or deleted files")
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
	slog.Info("validation succeeded: all proto changes resulted in generated file changes")

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
		slog.Info("validation succeeded: no unrelated file changes found")
	}

	slog.Info("all validation checks passed")
	return nil
}

func prepareForGenerateTest(state *config.LibrarianState, libraryID string, sourceRepo gitrepo.Repository) (map[string]string, error) {
	libraryState := findLibraryByID(state, libraryID)
	if libraryState == nil {
		return nil, fmt.Errorf("library %q not found in state", libraryID)
	}
	lastGeneratedCommit := libraryState.LastGeneratedCommit
	if lastGeneratedCommit == "" {
		return nil, fmt.Errorf("last_generated_commit is not set for library %q", libraryID)
	}

	branchName := "test-generate-" + uuid.New().String()
	slog.Info(fmt.Sprintf("checking out new branch %s from %s", branchName, lastGeneratedCommit))
	if err := sourceRepo.Checkout(lastGeneratedCommit); err != nil {
		return nil, err
	}
	if err := sourceRepo.CreateBranchAndCheckout(branchName); err != nil {
		return nil, err
	}

	protoFiles := []string{}
	sourceRepoPath := sourceRepo.GetDir()
	for _, API := range libraryState.APIs {
		root := filepath.Join(sourceRepoPath, API.Path)
		err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() && strings.HasSuffix(info.Name(), ".proto") {
				relPath, err := filepath.Rel(sourceRepoPath, path)
				if err != nil {
					return err
				}
				protoFiles = append(protoFiles, relPath)
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	}

	protoFileToGUID := make(map[string]string)
	for _, protoFile := range protoFiles {
		absPath := filepath.Join(sourceRepoPath, protoFile)
		content, err := os.ReadFile(absPath)
		if err != nil {
			return nil, err
		}
		lines := strings.Split(string(content), "\n")
		if len(lines) == 0 {
			continue
		}

		insertionLine := -1
		searchTerms := []string{"message ", "enum ", "service "}
		for _, term := range searchTerms {
			for i, line := range lines {
				if strings.HasPrefix(strings.TrimSpace(line), term) {
					insertionLine = i
					break
				}
			}
			if insertionLine != -1 {
				break
			}
		}

		if insertionLine != -1 {
			guid := uuid.New().String()
			protoFileToGUID[protoFile] = guid
			comment := "// test-change-" + guid

			var newLines []string
			newLines = append(newLines, lines[:insertionLine]...)
			newLines = append(newLines, comment)
			newLines = append(newLines, lines[insertionLine:]...)

			output := strings.Join(newLines, "\n")
			if err := os.WriteFile(absPath, []byte(output), 0644); err != nil {
				return nil, err
			}
		}
	}

	slog.Info("committing test changes")
	if err := sourceRepo.AddAll(); err != nil {
		return nil, err
	}
	if err := sourceRepo.Commit("test(changes): temporary changes for generate test"); err != nil {
		return nil, err
	}

	return protoFileToGUID, nil
}

func (r *testGenerateRunner) runGenerateCommand(ctx context.Context, libraryID, outputDir string) (string, error) {
	apiRoot, err := filepath.Abs(r.sourceRepo.GetDir())
	if err != nil {
		return "", err
	}

	generateRequest := &docker.GenerateRequest{
		ApiRoot:   apiRoot,
		LibraryID: libraryID,
		Output:    outputDir,
		RepoDir:   r.repo.GetDir(),
		State:     r.state,
	}
	slog.Info("Performing generation for library", "id", libraryID, "outputDir", outputDir)
	if err := r.containerClient.Generate(ctx, generateRequest); err != nil {
		return "", err
	}

	// Read the library state from the response and check for generator-side errors.
	libraryState, err := readLibraryState(
		filepath.Join(generateRequest.RepoDir, config.LibrarianDir, config.GenerateResponse))
	if err != nil {
		return "", fmt.Errorf("failed to read library state from generator response: %w", err)
	}
	if libraryState.ErrorMessage != "" {
		return "", fmt.Errorf("generator container returned an error: %s", libraryState.ErrorMessage)
	}

	if err := cleanAndCopyLibrary(r.state, r.repo.GetDir(), libraryID, outputDir); err != nil {
		return "", err
	}

	slog.Info("Generation succeeds", "id", libraryID)
	return libraryID, nil
}
