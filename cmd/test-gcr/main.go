package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"golang.org/x/mod/semver"
)

const (
	minGoVersion  = "v1.25.0"
	minRustcVersion = "v1.85.0"
)

type config struct {
	gcrPath       string
	librarianPath string
	gcrBranch     string
	cargoArgs     string
	sidekickArgs  string
	dryRun        bool
	exec          executor
}

type executor interface {
	Run(cmd *exec.Cmd) error
	Output(cmd *exec.Cmd) ([]byte, error)
}

type cmdExecutor struct{}

func (c cmdExecutor) Run(cmd *exec.Cmd) error {
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func (c cmdExecutor) Output(cmd *exec.Cmd) ([]byte, error) {
	return cmd.Output()
}

func main() {
	cfg := config{
		exec: cmdExecutor{},
	}

	flag.StringVar(&cfg.gcrPath, "gcr-path", "", "Absolute path to the local google-cloud-rust repository clone (Required)")
	flag.StringVar(&cfg.librarianPath, "librarian-path", "", "Absolute path to the local librarian repository clone (Required)")
	flag.StringVar(&cfg.gcrBranch, "gcr-branch", "upstream/main", "The branch, tag, or commit in google-cloud-rust to check out")
	flag.StringVar(&cfg.cargoArgs, "cargo-args", "", "Additional space-separated arguments to pass to cargo test")
	flag.StringVar(&cfg.sidekickArgs, "sidekick-args", "", "A string containing space-separated arguments to be passed to the sidekick refresh command")
	flag.BoolVar(&cfg.dryRun, "dry-run", false, "Print commands that would be executed instead of running them")

	flag.Parse()

	if err := validateConfig(&cfg); err != nil {
		flag.Usage()
		log.Fatalf("Invalid arguments: %v", err)
	}

	fmt.Println("Configuration:")
	fmt.Printf("  gcr-path: %s\n", cfg.gcrPath)
	fmt.Printf("  librarian-path: %s\n", cfg.librarianPath)
	fmt.Printf("  gcr-branch: %s\n", cfg.gcrBranch)
	fmt.Printf("  cargo-args: 	'%s'\n", cfg.cargoArgs)
	fmt.Printf("  sidekick-args: 	'%s'\n", cfg.sidekickArgs)
	fmt.Printf("  dry-run: %t\n", cfg.dryRun)

	if err := checkDependencies(&cfg); err != nil {
		log.Fatalf("Dependency check failed: %v", err)
	}

	if err := prepareGCRRepo(&cfg); err != nil {
		log.Fatalf("Failed to prepare google-cloud-rust repo: %v", err)
	}

	if err := runSidekickAndTests(&cfg); err != nil {
		log.Fatalf("Failed to run sidekick and tests: %v", err)
	}

	fmt.Println("--- Successfully completed ---")
}

func validateConfig(cfg *config) error {
	if cfg.gcrPath == "" {
		return fmt.Errorf("--gcr-path is required")
	}
	if cfg.librarianPath == "" {
		return fmt.Errorf("--librarian-path is required")
	}

	if err := checkDirIsGitRepo(cfg, cfg.gcrPath, "google-cloud-rust"); err != nil {
		return fmt.Errorf("--gcr-path: %w", err)
	}
	if err := checkDirIsGitRepo(cfg, cfg.librarianPath, "librarian"); err != nil {
		return fmt.Errorf("--librarian-path: %w", err)
	}

	// Check for upstream remote
	if err := cfg.runCmd(cfg.gcrPath, "git", "remote", "get-url", "upstream"); err != nil {
		return fmt.Errorf("gcr-path (%s): failed to get remote 'upstream'. Please add it, e.g., git remote add upstream https://github.com/googleapis/google-cloud-rust.git", cfg.gcrPath)
	}
	if err := cfg.runCmd(cfg.librarianPath, "git", "remote", "get-url", "upstream"); err != nil {
		return fmt.Errorf("librarian-path (%s): failed to get remote 'upstream'. Please add it, e.g., git remote add upstream https://github.com/googleapis/librarian.git", cfg.librarianPath)
	}

	sidekickPath := filepath.Join(cfg.librarianPath, "cmd", "sidekick")
	if err := checkDir(sidekickPath); err != nil {
		return fmt.Errorf("sidekick command not found at %s: %w", sidekickPath, err)
	}

	return nil
}

func checkDirIsGitRepo(cfg *config, path string, repoName string) error {
	if err := checkDir(path); err != nil {
		return err
	}
	// Use a real executor for this check, as it's about the file system state
	realExec := cmdExecutor{}
	cmd := exec.Command("git", "rev-parse", "--is-inside-work-tree")
	cmd.Dir = path
	cmd.Stdout = nil // Suppress output
	cmd.Stderr = nil
	if err := realExec.Run(cmd); err != nil {
		return fmt.Errorf("path is not a git repository: %s. Please clone the %s repository", path, repoName)
	}
	return nil
}

func checkDir(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("directory not found: %s", path)
		}
		return fmt.Errorf("error accessing path %s: %w", path, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("path is not a directory: %s", path)
	}
	return nil
}

func getwd() string {
	wd, err := os.Getwd()
	if err != nil {
		log.Printf("Warning: failed to get current working directory: %v", err)
		return "<unknown>"
	}
	return wd
}

func isReadOnlyCommand(name string, args ...string) bool {
	if name == "go" && len(args) > 0 && args[0] == "version" { return true }
	if name == "rustc" && len(args) > 0 && args[0] == "--version" { return true }
	if name == "git" {
		if len(args) > 0 {
			switch args[0] {
			case "rev-parse", "remote", "status", "ls-files":
				return true
			}
		}
	}
	return false
}

func (c *config) runCmd(dir string, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	displayDir := dir
	if displayDir == "" {
		displayDir = getwd()
	}
	cmd.Dir = dir
	readOnly := isReadOnlyCommand(name, args...)

	fmt.Printf("Running command: %s %s (in %s)\n", name, strings.Join(args, " "), displayDir)
	if (c.dryRun && !readOnly) {
		fmt.Println("  (Dry run, skipped execution)")
		return nil
	}
	return c.exec.Run(cmd)
}

func (c *config) runCmdOutput(dir string, name string, args ...string) ([]byte, error) {
	cmd := exec.Command(name, args...)
	displayDir := dir
	if displayDir == "" {
		displayDir = getwd()
	}
	cmd.Dir = dir
	readOnly := isReadOnlyCommand(name, args...)

	fmt.Printf("Running command for output: %s %s (in %s)\n", name, strings.Join(args, " "), displayDir)
	if (c.dryRun && !readOnly) {
		fmt.Println("  (Dry run, skipped execution)")
		// Return mock dry run output for version checks if they were not run
		switch name {
		case "go":
			return []byte("go version go1.25.0 linux/amd64"), nil
		case "rustc":
			return []byte("rustc 1.85.0 (2025-09-20)"), nil
		}
		return []byte{}, nil
	}
	return c.exec.Output(cmd)
}

var goVersionRegex = regexp.MustCompile(`go version go(\d+\.\d+\.\d+)(?:\s|$)`)
var rustcVersionRegex = regexp.MustCompile(`rustc (\d+\.\d+\.\d+)`)

func checkDependencies(cfg *config) error {
	fmt.Println("--- Checking Dependencies ---")
	// Check Go version
	out, err := cfg.runCmdOutput("", "go", "version")
	if err != nil {
		return fmt.Errorf("go not found: %w", err)
	}
	match := goVersionRegex.FindStringSubmatch(string(out))
	if len(match) < 2 {
		return fmt.Errorf("could not parse go version from: %s", string(out))
	}
	goVersion := "v" + match[1]
	if semver.Compare(goVersion, minGoVersion) < 0 {
		return fmt.Errorf("go version %s is less than minimum required %s", goVersion, minGoVersion)
	}
	fmt.Printf("  Go version: %s (OK)\n", goVersion)

	// Check Rustc version
	out, err = cfg.runCmdOutput("", "rustc", "--version")
	if err != nil {
		return fmt.Errorf("rustc not found: %w", err)
	}
	match = rustcVersionRegex.FindStringSubmatch(string(out))
	if len(match) < 2 {
		return fmt.Errorf("could not parse rustc version from: %s", string(out))
	}
	rustcVersion := "v" + match[1]
	if semver.Compare(rustcVersion, minRustcVersion) < 0 {
		return fmt.Errorf("rustc version %s is less than minimum required %s", rustcVersion, minRustcVersion)
	}
	fmt.Printf("  Rustc version: %s (OK)\n", rustcVersion)

	return nil
}

func prepareGCRRepo(cfg *config) error {
	fmt.Println("--- Preparing google-cloud-rust repo ---")

	// Check for uncommitted changes
	out, err := cfg.runCmdOutput(cfg.gcrPath, "git", "status", "--porcelain")
	if err != nil {
		return fmt.Errorf("failed to check git status: %w", err)
	}
	if len(strings.TrimSpace(string(out))) > 0 {
		return fmt.Errorf("google-cloud-rust repo has uncommitted changes:\n%s", string(out))
	}

	// Check for untracked files
	out, err = cfg.runCmdOutput(cfg.gcrPath, "git", "ls-files", "--others", "--exclude-standard")
	if err != nil {
		return fmt.Errorf("failed to check for untracked files: %w", err)
	}
	if len(strings.TrimSpace(string(out))) > 0 {
		return fmt.Errorf("google-cloud-rust repo has untracked files:\n%s", string(out))
	}

	// Fetch updates
	if err := cfg.runCmd(cfg.gcrPath, "git", "fetch", "upstream"); err != nil {
		return fmt.Errorf("failed to fetch upstream: %w", err)
	}

	// Validate branch exists
	if err := cfg.runCmd(cfg.gcrPath, "git", "rev-parse", "--verify", "--quiet", cfg.gcrBranch); err != nil {
		return fmt.Errorf("branch %s not found in google-cloud-rust repo (%s). Please check the branch name and ensure you have fetched the latest changes from upstream", cfg.gcrBranch, cfg.gcrPath)
	}

	// Reset to target branch
	if err := cfg.runCmd(cfg.gcrPath, "git", "reset", "--hard", cfg.gcrBranch); err != nil {
		return fmt.Errorf("failed to reset to branch %s: %w", cfg.gcrBranch, err)
	}

	fmt.Println("--- google-cloud-rust repo prepared ---")
	return nil
}

func runSidekickAndTests(cfg *config) error {
	fmt.Println("--- Running Sidekick & Tests ---")

	// Cleanup function to reset the repo
	defer func() {
		fmt.Println("--- Cleaning up google-cloud-rust repo ---")
		if err := cfg.runCmd(cfg.gcrPath, "git", "reset", "--hard", "HEAD"); err != nil {
			log.Printf("Warning: failed to reset repo: %v", err)
		}
		if err := cfg.runCmd(cfg.gcrPath, "git", "clean", "-fdx"); err != nil {
			log.Printf("Warning: failed to clean repo: %v", err)
		}
	}()

	// Regenerate Code
	sidekickCmdArgs := []string{
		"run", "./cmd/sidekick", "refresh",
		"-project-root", cfg.gcrPath,
		"-output", "src/generated/showcase",
	}
	if cfg.sidekickArgs != "" {
		sidekickCmdArgs = append(sidekickCmdArgs, strings.Fields(cfg.sidekickArgs)...)
	}
	if err := cfg.runCmd(cfg.librarianPath, "go", sidekickCmdArgs...); err != nil {
		return fmt.Errorf("failed to run sidekick refresh: %w", err)
	}

	// No-Op Change Check
	out, err := cfg.runCmdOutput(cfg.gcrPath, "git", "status", "--porcelain")
	if err != nil {
		return fmt.Errorf("failed to check git status after sidekick: %w", err)
	}
	if len(strings.TrimSpace(string(out))) == 0 && !cfg.dryRun {
		return fmt.Errorf("no changes generated by sidekick. Test failed")
	}

	// Format changes
	if err := cfg.runCmd(cfg.gcrPath, "cargo", "fmt", "-p", "google-cloud-showcase-v1beta1"); err != nil {
		return fmt.Errorf("failed to format showcase: %w", err)
	}

	// Run Tests
	cargoTestArgs := strings.Fields(cfg.cargoArgs)

	showcaseTestArgs := append([]string{"test", "-p", "google-cloud-showcase-v1beta1"}, cargoTestArgs...)
	if err := cfg.runCmd(cfg.gcrPath, "cargo", showcaseTestArgs...); err != nil {
		return fmt.Errorf("failed to run showcase tests: %w", err)
	}

	integrationTestArgs := append([]string{"test", "-p", "integration-tests", "--features", "integration-tests/run-showcase-tests"}, cargoTestArgs...)
	if err := cfg.runCmd(cfg.gcrPath, "cargo", integrationTestArgs...); err != nil {
		return fmt.Errorf("failed to run integration tests: %w", err)
	}

	fmt.Println("--- Sidekick & Tests Completed ---")
	return nil
}