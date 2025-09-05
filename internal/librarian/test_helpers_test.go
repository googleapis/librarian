package librarian

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunGit_Success(t *testing.T) {
	t.Parallel()
	repoDir := createTempGitRepo(t)
	createDummyFile(t, repoDir, "testfile.txt")

	runGit(t, repoDir, "add", "testfile.txt")
	runGit(t, repoDir, "commit", "-m", "Add testfile")

	output, err := exec.Command("git", "-C", repoDir, "log", "--oneline").Output()
	if err != nil {
		t.Fatalf("Failed to get git log: %v", err)
	}
	if !strings.Contains(string(output), "Add testfile") {
		t.Errorf("Expected 'Add testfile' in git log, got: %s", string(output))
	}
}

// Helper to create a temporary git repository for testing.
func createTempGitRepo(t *testing.T) string {
	t.Helper()
	repoDir := t.TempDir()
	runGit(t, repoDir, "init")
	return repoDir
}

// Helper to create a dummy file in a directory.
func createDummyFile(t *testing.T, dir, filename string) {
	t.Helper()
	filePath := filepath.Join(dir, filename)
	if err := os.WriteFile(filePath, []byte("dummy content"), 0644); err != nil {
		t.Fatalf("failed to create dummy file: %v", err)
	}
}