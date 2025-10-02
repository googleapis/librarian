package main

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// mockExecutor is a mock implementation of the executor interface for testing.
// It records the commands that would be executed and can be configured to return specific outputs or errors.
// - commands: Stores the arguments of each command executed.
// - outputs: Maps full command strings to a struct containing the byte output and error to return.
// - runErr: Maps command prefixes (like "git" or "go run") to errors to return when Run is called.
type mockExecutor struct {
	commands [][]string
	outputs  map[string]struct {
		out []byte
		err error
	}
	runErr map[string]error
}

func (m *mockExecutor) Run(cmd *exec.Cmd) error {
	m.commands = append(m.commands, cmd.Args)
	if m.runErr != nil {
		// Check for specific command errors based on a prefix match
		cmdString := strings.Join(cmd.Args, " ")
		for key, err := range m.runErr {
			if strings.HasPrefix(cmdString, key) {
				return err
			}
		}
	}
	return nil
}

func (m *mockExecutor) Output(cmd *exec.Cmd) ([]byte, error) {
	m.commands = append(m.commands, cmd.Args)
	key := strings.Join(cmd.Args, " ")
	if val, ok := m.outputs[key]; ok {
		return val.out, val.err
	}
	return nil, errors.New("mock output not found for: " + key)
}

func TestValidateConfig(t *testing.T) {
	tmpDir := t.TempDir()
	gcrDir := filepath.Join(tmpDir, "gcr")
	libDir := filepath.Join(tmpDir, "librarian")
	sidekickDir := filepath.Join(libDir, "cmd", "sidekick")

	if err := os.MkdirAll(gcrDir, 0755); err != nil {
		t.Fatalf("Failed to create temp gcr dir: %v", err)
	}
	if err := os.MkdirAll(sidekickDir, 0755); err != nil {
		t.Fatalf("Failed to create temp librarian dir: %v", err)
	}

	// Create dummy git repos
	cmdGcr := exec.Command("git", "init")
	cmdGcr.Dir = gcrDir
	if err := cmdGcr.Run(); err != nil {
		t.Fatalf("Failed to init git repo in %s: %v", gcrDir, err)
	}
	cmdLib := exec.Command("git", "init")
	cmdLib.Dir = libDir
	if err := cmdLib.Run(); err != nil {
		t.Fatalf("Failed to init git repo in %s: %v", libDir, err)
	}

	tests := []struct {
		name    string
		cfg     config
		runErr  map[string]error
		wantErr bool
	}{
		{
			name: "valid",
			cfg: config{
				gcrPath:       gcrDir,
				librarianPath: libDir,
			},
		},
		{
			name: "missing_gcr_path",
			cfg: config{
				librarianPath: libDir,
			},
			wantErr: true,
		},
		{
			name: "gcr_not_git",
			cfg: config{
				gcrPath:       t.TempDir(), // Empty dir
				librarianPath: libDir,
			},
			wantErr: true,
		},
		{
			name: "lib_not_git",
			cfg: config{
				gcrPath:       gcrDir,
				librarianPath: t.TempDir(), // Empty dir
			},
			wantErr: true,
		},
		{
			name: "gcr_no_upstream",
			cfg: config{
				gcrPath:       gcrDir,
				librarianPath: libDir,
			},
			runErr:  map[string]error{"git remote get-url upstream": errors.New("no upstream")},
			wantErr: true,
		},
		{
			name: "lib_no_upstream",
			cfg: config{
				gcrPath:       gcrDir,
				librarianPath: libDir,
			},
			runErr:  map[string]error{"git remote get-url upstream": errors.New("no upstream")},
			wantErr: true,
		},
		{
			name: "no_sidekick_cmd",
			cfg: config{
				gcrPath:       gcrDir,
				librarianPath: t.TempDir(), // Empty lib dir
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create fresh dirs for each test to avoid state leakage
			testDir := t.TempDir()
			gcrDir := filepath.Join(testDir, "gcr")
			libDir := filepath.Join(testDir, "librarian")
			sidekickDir := filepath.Join(libDir, "cmd", "sidekick")

			if err := os.MkdirAll(gcrDir, 0755); err != nil {
				t.Fatalf("Failed to create temp gcr dir: %v", err)
			}
			if err := os.MkdirAll(sidekickDir, 0755); err != nil {
				t.Fatalf("Failed to create temp librarian dir: %v", err)
			}

			tc := tt.cfg
			// Override paths with test-specific temp dirs
			if tc.gcrPath != "" { tc.gcrPath = gcrDir }
			if tc.librarianPath != "" { tc.librarianPath = libDir }

			// Init git repos if paths are set
			if tc.gcrPath != "" && tt.name != "gcr_not_git" {
				cmdGcr := exec.Command("git", "init")
				cmdGcr.Dir = tc.gcrPath
				if err := cmdGcr.Run(); err != nil {
					t.Fatalf("Failed to init git repo in %s: %v", tc.gcrPath, err)
				}
				// Note: We don't add upstream remote here, mockExecutor will handle the expectation
			}
			if tc.librarianPath != "" && tt.name != "lib_not_git" {
				cmdLib := exec.Command("git", "init")
				cmdLib.Dir = tc.librarianPath
				if err := cmdLib.Run(); err != nil {
					t.Fatalf("Failed to init git repo in %s: %v", tc.librarianPath, err)
				}
				// Note: We don't add upstream remote here, mockExecutor will handle the expectation
			}
			if tc.librarianPath != "" && tt.name == "no_sidekick_cmd" {
				if err := os.RemoveAll(sidekickDir); err != nil {
					t.Fatalf("Failed to remove sidekick dir: %v", err)
				}
			}

			mockExec := &mockExecutor{runErr: tt.runErr}
			tc.exec = mockExec

			err := validateConfig(&tc)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func createTempFile(t *testing.T, dir, name string) string {
	t.Helper()
	filePath := filepath.Join(dir, name)
	if _, err := os.Create(filePath); err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	return filePath
}

func TestRunCmd(t *testing.T) {
	tests := []struct {
		name         string
		dryRun       bool
		execErr      map[string]error
		wantErr      bool
		wantCommands [][]string
	}{
		{
			name:         "real_run_success",
			dryRun:       false,
			wantCommands: [][]string{{"echo", "hello"}},
		},
		{
			name:    "real_run_fail",
			dryRun:  false,
			execErr: map[string]error{"echo": errors.New("exec failed")},
			wantErr: true,
			wantCommands: [][]string{{"echo", "hello"}},
		},
		{
			name:         "dry_run",
			dryRun:       true,
			wantCommands: nil, // No commands should be executed by the mockExecutor
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockExec := &mockExecutor{runErr: tt.execErr}
			cfg := config{dryRun: tt.dryRun, exec: mockExec}
			err := cfg.runCmd("", "echo", "hello")
			if (err != nil) != tt.wantErr {
				t.Errorf("runCmd() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !reflect.DeepEqual(mockExec.commands, tt.wantCommands) {
				t.Errorf("runCmd() commands = %v, want %v", mockExec.commands, tt.wantCommands)
			}
		})
	}
}

func TestCheckDependencies(t *testing.T) {
	tests := []struct {
		name         string
		mockOutputs  map[string]struct {
			out []byte
			err error
		}
		wantErr      bool
		wantCommands [][]string
	}{
		{
			name: "success",
			mockOutputs: map[string]struct {
				out []byte
				err error
			}{
				"go version":        {out: []byte("go version go1.25.0 linux/amd64")},
				"rustc --version": {out: []byte("rustc 1.85.0 (abcdef 2025-09-20)")},
			},
			wantCommands: [][]string{{"go", "version"}, {"rustc", "--version"}},
		},
		{
			name: "go_too_old",
			mockOutputs: map[string]struct {
				out []byte
				err error
			}{
				"go version":        {out: []byte("go version go1.24.0 linux/amd64")},
				"rustc --version": {out: []byte("rustc 1.85.0")},
			},
			wantErr:      true,
			wantCommands: [][]string{{"go", "version"}},
		},
		{
			name: "rustc_too_old",
			mockOutputs: map[string]struct {
				out []byte
				err error
			}{
				"go version":        {out: []byte("go version go1.25.0")},
				"rustc --version": {out: []byte("rustc 1.84.0")},
			},
			wantErr:      true,
			wantCommands: [][]string{{"go", "version"}, {"rustc", "--version"}},
		},
		{
			name: "go_parse_fail",
			mockOutputs: map[string]struct {
				out []byte
				err error
			}{
				"go version": {out: []byte("invalid go version output")},
			},
			wantErr:      true,
			wantCommands: [][]string{{"go", "version"}},
		},
		{
			name: "rustc_parse_fail",
			mockOutputs: map[string]struct {
				out []byte
				err error
			}{
				"go version":        {out: []byte("go version go1.25.0")},
				"rustc --version": {out: []byte("invalid rustc version output")},
			},
			wantErr:      true,
			wantCommands: [][]string{{"go", "version"}, {"rustc", "--version"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockExec := &mockExecutor{outputs: tt.mockOutputs}
			cfg := config{exec: mockExec}
			err := checkDependencies(&cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("checkDependencies() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !reflect.DeepEqual(mockExec.commands, tt.wantCommands) {
				t.Errorf("checkDependencies() commands = %v, want %v", mockExec.commands, tt.wantCommands)
			}
		})
	}
}

func TestPrepareGCRRepo(t *testing.T) {
	gcrPath := "/tmp/gcr"
	tests := []struct {
		name         string
		cfg          config
		mockOutputs  map[string]struct {
			out []byte
			err error
		}
		runErr       map[string]error
		wantErr      bool
		wantCommands [][]string
	}{
		{
			name: "success",
			cfg:  config{gcrPath: gcrPath, gcrBranch: "test-branch"},
			mockOutputs: map[string]struct {
				out []byte
				err error
			}{
				"git status --porcelain":                      {out: []byte("")},
				"git ls-files --others --exclude-standard": {out: []byte("")},
			},
			wantCommands: [][]string{
				{"git", "status", "--porcelain"},
				{"git", "ls-files", "--others", "--exclude-standard"},
				{"git", "fetch", "upstream"},
				{"git", "rev-parse", "--verify", "--quiet", "test-branch"},
				{"git", "reset", "--hard", "test-branch"},
			},
		},
		{
			name: "dirty_repo",
			cfg:  config{gcrPath: gcrPath},
			mockOutputs: map[string]struct {
				out []byte
				err error
			}{
				"git status --porcelain": {out: []byte(" M file.txt")},
			},
			wantErr: true,
			wantCommands: [][]string{{"git", "status", "--porcelain"}},
		},
		{
			name: "untracked_files",
			cfg:  config{gcrPath: gcrPath},
			mockOutputs: map[string]struct {
				out []byte
				err error
			}{
				"git status --porcelain":                      {out: []byte("")},
				"git ls-files --others --exclude-standard": {out: []byte("?? new.txt")},
			},
			wantErr: true,
			wantCommands: [][]string{
				{"git", "status", "--porcelain"},
				{"git", "ls-files", "--others", "--exclude-standard"},
			},
		},
		{
			name: "fetch_fails",
			cfg:  config{gcrPath: gcrPath},
			mockOutputs: map[string]struct {
				out []byte
				err error
			}{
				"git status --porcelain":                      {out: []byte("")},
				"git ls-files --others --exclude-standard": {out: []byte("")},
			},
			runErr:  map[string]error{"git fetch upstream": errors.New("fetch failed")},
			wantErr: true,
			wantCommands: [][]string{
				{"git", "status", "--porcelain"},
				{"git", "ls-files", "--others", "--exclude-standard"},
				{"git", "fetch", "upstream"},
			},
		},
		{
			name: "branch_not_found",
			cfg:  config{gcrPath: gcrPath, gcrBranch: "nonexistent-branch"},
			mockOutputs: map[string]struct {
				out []byte
				err error
			}{
				"git status --porcelain":                      {out: []byte("")},
				"git ls-files --others --exclude-standard": {out: []byte("")},
			},
			runErr:  map[string]error{"git rev-parse --verify --quiet nonexistent-branch": errors.New("branch not found")},
			wantErr: true,
			wantCommands: [][]string{
				{"git", "status", "--porcelain"},
				{"git", "ls-files", "--others", "--exclude-standard"},
				{"git", "fetch", "upstream"},
				{"git", "rev-parse", "--verify", "--quiet", "nonexistent-branch"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockExec := &mockExecutor{outputs: tt.mockOutputs, runErr: tt.runErr}
			tc := tt.cfg
			tc.exec = mockExec
			err := prepareGCRRepo(&tc)
			if (err != nil) != tt.wantErr {
				t.Errorf("prepareGCRRepo() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !reflect.DeepEqual(mockExec.commands, tt.wantCommands) {
				t.Errorf("prepareGCRRepo() commands = %v, want %v", mockExec.commands, tt.wantCommands)
			}
		})
	}
}

func TestRunSidekickAndTests(t *testing.T) {
	gcrPath := "/tmp/gcr"
	libPath := "/tmp/lib"

	tests := []struct {
		name         string
		cfg          config
		mockOutputs  map[string]struct {
			out []byte
			err error
		}
		runErr       map[string]error
		wantErr      bool
		wantCommands [][]string
	}{
		{
			name: "success",
			cfg:  config{gcrPath: gcrPath, librarianPath: libPath},
			mockOutputs: map[string]struct {
				out []byte
				err error
			}{
				"git status --porcelain": {out: []byte("M somefile.rs")}, // Changes after sidekick
			},
			wantCommands: [][]string{
				{"go", "run", "./cmd/sidekick", "refresh", "-project-root", gcrPath, "-output", "src/generated/showcase"},
				{"git", "status", "--porcelain"},
				{"cargo", "fmt", "-p", "google-cloud-showcase-v1beta1"},
				{"cargo", "test", "-p", "google-cloud-showcase-v1beta1"},
				{"cargo", "test", "-p", "integration-tests", "--features", "integration-tests/run-showcase-tests"},
				// Cleanup commands
				{"git", "reset", "--hard", "HEAD"},
				{"git", "clean", "-fdx"},
			},
		},
		{
			name: "success_with_args",
			cfg: config{
				gcrPath:       gcrPath,
				librarianPath: libPath,
				sidekickArgs:  "--verbose",
				cargoArgs:     "--quiet",
			},
			mockOutputs: map[string]struct {
				out []byte
				err error
			}{
				"git status --porcelain": {out: []byte("M somefile.rs")},
			},
			wantCommands: [][]string{
				{"go", "run", "./cmd/sidekick", "refresh", "-project-root", gcrPath, "-output", "src/generated/showcase", "--verbose"},
				{"git", "status", "--porcelain"},
				{"cargo", "fmt", "-p", "google-cloud-showcase-v1beta1"},
				{"cargo", "test", "-p", "google-cloud-showcase-v1beta1", "--quiet"},
				{"cargo", "test", "-p", "integration-tests", "--features", "integration-tests/run-showcase-tests", "--quiet"},
				{"git", "reset", "--hard", "HEAD"},
				{"git", "clean", "-fdx"},
			},
		},
		{
			name: "sidekick_fails",
			cfg:  config{gcrPath: gcrPath, librarianPath: libPath},
			runErr: map[string]error{"go run ./cmd/sidekick": errors.New("sidekick failed")},
			wantErr: true,
			wantCommands: [][]string{
				{"go", "run", "./cmd/sidekick", "refresh", "-project-root", gcrPath, "-output", "src/generated/showcase"},
				{"git", "reset", "--hard", "HEAD"}, // Cleanup runs
				{"git", "clean", "-fdx"},
			},
		},
		{
			name: "no_changes",
			cfg:  config{gcrPath: gcrPath, librarianPath: libPath},
			mockOutputs: map[string]struct {
				out []byte
				err error
			}{
				"git status --porcelain": {out: []byte("")},
			},
			wantErr: true,
			wantCommands: [][]string{
				{"go", "run", "./cmd/sidekick", "refresh", "-project-root", gcrPath, "-output", "src/generated/showcase"},
				{"git", "status", "--porcelain"},
				{"git", "reset", "--hard", "HEAD"},
				{"git", "clean", "-fdx"},
			},
		},
		{
			name: "cargo_fmt_fails",
			cfg:  config{gcrPath: gcrPath, librarianPath: libPath},
			mockOutputs: map[string]struct {
				out []byte
				err error
			}{
				"git status --porcelain": {out: []byte("M somefile.rs")},
			},
			runErr: map[string]error{"cargo fmt": errors.New("fmt failed")},
			wantErr: true,
			wantCommands: [][]string{
				{"go", "run", "./cmd/sidekick", "refresh", "-project-root", gcrPath, "-output", "src/generated/showcase"},
				{"git", "status", "--porcelain"},
				{"cargo", "fmt", "-p", "google-cloud-showcase-v1beta1"},
				{"git", "reset", "--hard", "HEAD"},
				{"git", "clean", "-fdx"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockExec := &mockExecutor{outputs: tt.mockOutputs, runErr: tt.runErr}
			tc := tt.cfg
			tc.exec = mockExec
			err := runSidekickAndTests(&tc)
			if (err != nil) != tt.wantErr {
				t.Errorf("runSidekickAndTests() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !reflect.DeepEqual(mockExec.commands, tt.wantCommands) {
				t.Errorf("runSidekickAndTests() commands = %v, want %v", mockExec.commands, tt.wantCommands)
			}
		})
	}
}