#!/usr/bin/env bash
#
# test-gcr.sh: Script to test local librarian/sidekick changes against google-cloud-rust.
# This script automates the process of:
# 1. Preparing a local google-cloud-rust clone.
# 2. Regenerating the showcase client using a local librarian/sidekick build.
# 3. Running tests on the regenerated client.
# 4. Cleaning up the google-cloud-rust clone.

set -euo pipefail

# Displays usage information for the script.
usage() {
  cat <<EOF
Usage: $0 --gcr-path <path> --librarian-path <path> [options]

Options:
  --gcr-path <path>         Absolute path to the local google-cloud-rust repository clone. (Required)
  --librarian-path <path>   Absolute path to the local librarian repository clone. (Required)
  --cargo-args <string>     Additional space-separated arguments to pass to cargo test.
  --non-interactive         Run in non-interactive mode. On failure, always clean up without prompting.
  --dry-run                 Print commands that would be executed instead of running them.
  --test                    Run the script's self-tests.
  -h, --help                Show this help message.
EOF
}

# Parses command line arguments and sets global variables.
parse_args() {
  GCR_PATH=""
  LIBRARIAN_PATH=""
  CARGO_ARGS=""
  NON_INTERACTIVE=false
  DRY_RUN=false
  RUN_TESTS=false

  while [[ $# -gt 0 ]]; do
    case "$1" in
      --gcr-path)
        GCR_PATH="$2"
        shift 2
        ;; 
      --librarian-path)
        LIBRARIAN_PATH="$2"
        shift 2
        ;; 
      --cargo-args)
        CARGO_ARGS="$2"
        shift 2
        ;; 
      --non-interactive)
        NON_INTERACTIVE=true
        shift
        ;; 
      --dry-run)
        DRY_RUN=true
        shift
        ;; 
      --test)
        RUN_TESTS=true
        shift
        ;; 
      -h | --help)
        usage
        exit 0
        ;; 
      *)
        echo "Unknown option: $1"
        usage
        exit 1
        ;; 
    esac
  done

  # Validate required arguments if not running self-tests.
  if [[ "$RUN_TESTS" == "false" ]]; then
    if [[ -z "$GCR_PATH" ]]; then
      echo "Error: --gcr-path is required."
      usage
      exit 1
    fi
    if [[ -z "$LIBRARIAN_PATH" ]]; then
      echo "Error: --librarian-path is required."
      usage
      exit 1
    fi
  fi
}

# --- Script Functions ---

# Executes a command, respecting the --dry-run flag.
# @param {string} cmd The command to execute.
run_cmd() {
  local cmd="$1"
  echo "Running command: $cmd"
  if [[ "$DRY_RUN" == "true" ]]; then
    echo "[DRY RUN] Would execute: $cmd"
    return 0
  fi
  eval "$cmd"
}

# Checks for required dependencies (git, go, cargo, rustc) and their versions.
check_dependencies() {
  echo "Checking dependencies..."
  command -v git >/dev/null 2>&1 || { echo >&2 "git is not installed. Aborting."; exit 1; }
  command -v go >/dev/null 2>&1 || { echo >&2 "go is not installed. Aborting."; exit 1; }
  command -v cargo >/dev/null 2>&1 || { echo >&2 "cargo is not installed. Aborting."; exit 1; }
  command -v rustc >/dev/null 2>&1 || { echo >&2 "rustc is not installed. Aborting."; exit 1; }

  # Check Go version
  local go_version
  go_version=$(go version | awk '{print $3}' | sed 's/go//')
  local required_go_version="1.25.0"
  if ! printf "%s\n%s\n" "$required_go_version" "$go_version" | sort -V -C; then
    echo "Warning: go version $go_version is less than required $required_go_version. Proceeding, but errors may occur."
  fi

  # Check Rust version
  local rustc_version
  rustc_version=$(rustc --version | awk '{print $2}')
  local required_rustc_version="1.85.0"
  if ! printf "%s\n%s\n" "$required_rustc_version" "$rustc_version" | sort -V -C; then
    echo "Warning: rustc version $rustc_version is less than required $required_rustc_version. Proceeding, but errors may occur."
  fi
  echo "Dependencies OK."
}

# Cleans up the google-cloud-rust repository by resetting changes.
# Prompts the user if not in non-interactive mode and an error occurred.
# @param {integer} exit_code The exit code of the script.
cleanup_gcr() {
  local exit_code=$1 # Passed from trap

  if [[ $exit_code -eq 0 ]]; then
    echo "Script finished successfully. Cleaning up google-cloud-rust..."
    run_cmd "git reset --hard HEAD"
    run_cmd "git clean -fdx"
  else
    echo "An error occurred (exit code: $exit_code)."
    if [[ "$NON_INTERACTIVE" == "true" ]]; then
      echo "Non-interactive mode: Cleaning up google-cloud-rust..."
      run_cmd "git reset --hard HEAD"
      run_cmd "git clean -fdx"
    else
      read -p "Keep changes in google-cloud-rust? (y/N): " choice
      case "$choice" in
        y|Y ) echo "Keeping changes.";;
        * )   echo "Cleaning up google-cloud-rust...";
              run_cmd "git reset --hard HEAD";
              run_cmd "git clean -fdx";;
      esac
    fi
  fi
  return $exit_code
}

# Main function to orchestrate the script's workflow.
main() {
  parse_args "$@"

  if [[ "$RUN_TESTS" == "true" ]]; then
    run_tests
    exit 0
  fi

  check_dependencies

  # Set up trap to ensure cleanup_gcr is called on exit, error, or interrupt.
  trap 'cleanup_gcr $?' EXIT ERR INT TERM

  echo "Changing directory to $GCR_PATH"
  cd "$GCR_PATH" || exit 1

  # Verify that the provided paths are git repositories.
  if ! git rev-parse --is-inside-work-tree > /dev/null 2>&1; then
    echo "Error: $GCR_PATH is not a git repository." >&2
    exit 1
  fi
  if ! git -C "$LIBRARIAN_PATH" rev-parse --is-inside-work-tree > /dev/null 2>&1; then
    echo "Error: $LIBRARIAN_PATH is not a git repository." >&2
    exit 1
  fi

  echo "GCR_PATH: $GCR_PATH"
  echo "LIBRARIAN_PATH: $LIBRARIAN_PATH"
  echo "CARGO_ARGS: $CARGO_ARGS"
  echo "NON_INTERACTIVE: $NON_INTERACTIVE"
  echo "DRY_RUN: $DRY_RUN"

  echo "Preparing google-cloud-rust repository..."
  # Check for dirty working tree.
  if [[ -n $(git status --porcelain) ]]; then
    echo "Error: google-cloud-rust repository has uncommitted changes." >&2
    exit 1
  fi
  if [[ -n $(git ls-files --others --exclude-standard) ]]; then
    echo "Error: google-cloud-rust repository has untracked files." >&2
    exit 1
  fi

  # Reset to the latest upstream version.
  run_cmd "git fetch upstream"
  run_cmd "git reset --hard upstream/main"

  echo "Regenerating showcase client..."
  # Build and run sidekick from the local librarian clone to regenerate code.
  local sidekick_cmd="go -C \"$LIBRARIAN_PATH\" run ./cmd/sidekick refresh -project-root \"$PWD\" -output src/generated/showcase"
  run_cmd "$sidekick_cmd"

  # Check if sidekick generated any changes.
  if [[ -z $(git status --porcelain) ]]; then
    echo "Error: No changes generated by sidekick. Test failed." >&2
    exit 1
  fi

  # Format the generated code.
  run_cmd "cargo fmt -p google-cloud-showcase-v1beta1"

  echo "Generated changes diff:"
  run_cmd "git --no-pager diff"

  echo "Running tests..."
  # Run unit tests for the showcase crate.
  run_cmd "cargo test -p google-cloud-showcase-v1beta1 $CARGO_ARGS"
  # Run integration tests related to the showcase client.
  run_cmd "cargo test -p integration-tests --features integration-tests/run-showcase-tests $CARGO_ARGS"

  echo "Script finished successfully."
}

# --- Test Infrastructure ---
FAILURE_COUNT=0

# Assertion helper: checks if two strings are equal.
# @param {string} expected The expected value.
# @param {string} actual The actual value.
# @param {string} message (Optional) Description of the assertion.
assert_eq() {
  local expected="$1"
  local actual="$2"
  local message="${3:-""}"
  if [[ "$expected" != "$actual" ]]; then
    echo "Assertion failed: $message - Expected: '$expected', Actual: '$actual'"
    return 1
  fi
  echo "PASS: $message"
  return 0
}

# Assertion helper: checks if a command fails (exits non-zero).
# @param {string} command_str The command to execute.
# @param {string} message Description of the assertion.
assert_fail() {
  local command_str="$1"
  local message="$2"
  local output
  if output=$(eval "$command_str" 2>&1); then
    echo "Assertion failed: $message - Expected failure, but succeeded with output: $output"
    return 1
  else
    echo "PASS: $message - Command failed as expected."
    return 0
  fi
}

# Assertion helper: checks if a string contains a substring.
# @param {string} expected_substring The substring to look for.
# @param {string} actual_output The string to search within.
# @param {string} message (Optional) Description of the assertion.
assert_output_contains() {
  local expected_substring="$1"
  local actual_output="$2"
  local message="${3:-""}"
  if [[ "$actual_output" != *"$expected_substring"* ]]; then
    echo "Assertion failed: $message - Expected output to contain '$expected_substring', Actual: '$actual_output'"
    return 1
  fi
  echo "PASS: $message"
  return 0
}

# Runs all self-tests for the script.
run_tests() {
  echo "Running self-tests..."
  FAILURE_COUNT=0

  test_parse_args_required || { echo "FAIL: test_parse_args_required"; FAILURE_COUNT=$((FAILURE_COUNT + 1)); }
  test_parse_args_options || { echo "FAIL: test_parse_args_options"; FAILURE_COUNT=$((FAILURE_COUNT + 1)); }
  test_parse_args_missing_gcr || { echo "FAIL: test_parse_args_missing_gcr"; FAILURE_COUNT=$((FAILURE_COUNT + 1)); }
  test_parse_args_missing_librarian || { echo "FAIL: test_parse_args_missing_librarian"; FAILURE_COUNT=$((FAILURE_COUNT + 1)); }
  test_parse_args_unknown_option || { echo "FAIL: test_parse_args_unknown_option"; FAILURE_COUNT=$((FAILURE_COUNT + 1)); }

  test_check_dependencies_runs || { echo "FAIL: test_check_dependencies_runs"; FAILURE_COUNT=$((FAILURE_COUNT + 1)); }

  test_run_cmd_dry_run || { echo "FAIL: test_run_cmd_dry_run"; FAILURE_COUNT=$((FAILURE_COUNT + 1)); }
  test_run_cmd_exec || { echo "FAIL: test_run_cmd_exec"; FAILURE_COUNT=$((FAILURE_COUNT + 1)); }

  test_cleanup_gcr_dry_run || { echo "FAIL: test_cleanup_gcr_dry_run"; FAILURE_COUNT=$((FAILURE_COUNT + 1)); }

  test_main_sidekick_command || { echo "FAIL: test_main_sidekick_command"; FAILURE_COUNT=$((FAILURE_COUNT + 1)); }
  test_main_cargo_test_commands || { echo "FAIL: test_main_cargo_test_commands"; FAILURE_COUNT=$((FAILURE_COUNT + 1)); }
  test_main_git_fetch_dry_run || { echo "FAIL: test_main_git_fetch_dry_run"; FAILURE_COUNT=$((FAILURE_COUNT + 1)); }
  test_main_git_reset_dry_run || { echo "FAIL: test_main_git_reset_dry_run"; FAILURE_COUNT=$((FAILURE_COUNT + 1)); }

  if [[ $FAILURE_COUNT -gt 0 ]]; then
    echo "$FAILURE_COUNT test(s) failed."
    exit 1
  else
    echo "All tests passed."
  fi
}

# --- Tests ---
# Tests that parse_args works with all required arguments.
test_parse_args_required() {
  local output
  output=$(parse_args --gcr-path /tmp --librarian-path /tmp 2>&1)
  assert_eq "" "$output" "parse_args with required args"
}

# Tests that parse_args correctly sets variables for all options.
test_parse_args_options() {
  parse_args --gcr-path /gcr --librarian-path /lib --cargo-args "--foo --bar" --non-interactive --dry-run
  assert_eq "/gcr" "$GCR_PATH" "GCR_PATH option" || return 1
  assert_eq "/lib" "$LIBRARIAN_PATH" "LIBRARIAN_PATH option" || return 1
  assert_eq "--foo --bar" "$CARGO_ARGS" "CARGO_ARGS option" || return 1
  assert_eq "true" "$NON_INTERACTIVE" "NON_INTERACTIVE option" || return 1
  assert_eq "true" "$DRY_RUN" "DRY_RUN option" || return 1
}

# Tests that parse_args fails if --gcr-path is missing.
test_parse_args_missing_gcr() {
  assert_fail "parse_args --librarian-path /tmp" "parse_args missing --gcr-path"
}

# Tests that parse_args fails if --librarian-path is missing.
test_parse_args_missing_librarian() {
  assert_fail "parse_args --gcr-path /tmp" "parse_args missing --librarian-path"
}

# Tests that parse_args fails with an unknown option.
test_parse_args_unknown_option() {
  assert_fail "parse_args --gcr-path /tmp --librarian-path /tmp --unknown" "parse_args unknown option"
}

# Tests that check_dependencies runs without error.
test_check_dependencies_runs() {
  # Just check that it runs without error in the current environment
  # Assuming the necessary tools (git, go, cargo, rustc) are on the PATH
  if check_dependencies > /dev/null 2>&1; then
    echo "PASS: check_dependencies runs without error"
    return 0
  else
    echo "FAIL: check_dependencies failed"
    return 1
  fi
}

# Tests run_cmd in dry-run mode.
test_run_cmd_dry_run() {
  export DRY_RUN=true
  local test_file="/tmp/test-gcr-dry-run.txt"
  trap 'rm -f "$test_file"' RETURN
  rm -f "$test_file"

  local output=$(run_cmd "echo 'Hello' > \"$test_file\" ")
  assert_output_contains "[DRY RUN] Would execute: echo 'Hello' > \"$test_file\" " "$output" "run_cmd dry run output" || return 1
  if [[ -f "$test_file" ]]; then
    echo "Assertion failed: run_cmd dry run should not create file"
    return 1
  fi
  unset DRY_RUN
}

# Tests run_cmd in execution mode.
test_run_cmd_exec() {
  DRY_RUN=false
  test_file="/tmp/test-gcr-run-cmd.txt"
  trap 'rm -f "$test_file"' RETURN
  rm -f "$test_file"

  local cmd="echo 'Hello World' > \"$test_file\""
  local output=$(run_cmd "$cmd")

  assert_output_contains "Running command: $cmd" "$output" "run_cmd exec output" || return 1

  if [[ ! -f "$test_file" ]]; then
    echo "Assertion failed: run_cmd should create file"
    return 1
  fi
  local content=$(cat "$test_file")
  assert_eq "Hello World" "$content" "run_cmd file content" || return 1
}

# Tests cleanup_gcr in dry-run mode.
test_cleanup_gcr_dry_run() {
  export DRY_RUN=true
  export NON_INTERACTIVE=true

  local output=$(cleanup_gcr 1 2>&1)
  assert_output_contains "[DRY RUN] Would execute: git reset --hard HEAD" "$output" "cleanup_gcr dry run error reset" || return 1
  assert_output_contains "[DRY RUN] Would execute: git clean -fdx" "$output" "cleanup_gcr dry run error clean" || return 1

  output=$(cleanup_gcr 0 2>&1)
  assert_output_contains "[DRY RUN] Would execute: git reset --hard HEAD" "$output" "cleanup_gcr dry run success reset" || return 1
  assert_output_contains "[DRY RUN] Would execute: git clean -fdx" "$output" "cleanup_gcr dry run success clean" || return 1

  unset DRY_RUN NON_INTERACTIVE
}

# Tests the sidekick command construction in dry-run mode.
test_main_sidekick_command() {
  export DRY_RUN=true
  LIBRARIAN_PATH="/lib"
  GCR_PATH="/gcr"
  PWD="/gcr"
  # Construct the command as in main()
  local sidekick_cmd="go -C \"$LIBRARIAN_PATH\" run ./cmd/sidekick refresh -project-root \"$PWD\" -output src/generated/showcase"
  local output=$(run_cmd "$sidekick_cmd")

  assert_output_contains "go -C \"/lib\" run ./cmd/sidekick refresh" "$output" "main sidekick command" || return 1
  assert_output_contains "-project-root \"/gcr\"" "$output" "main sidekick project root" || return 1
  assert_output_contains "-output src/generated/showcase" "$output" "main sidekick output" || return 1
  unset DRY_RUN
}

# Tests the cargo test command constructions in dry-run mode.
test_main_cargo_test_commands() {
  export DRY_RUN=true
  CARGO_ARGS="--foo --bar"
  local output1=$(run_cmd "cargo test -p google-cloud-showcase-v1beta1 $CARGO_ARGS")
  assert_output_contains "cargo test -p google-cloud-showcase-v1beta1 --foo --bar" "$output1" "main cargo test 1" || return 1

  local output2=$(run_cmd "cargo test -p integration-tests --features integration-tests/run-showcase-tests $CARGO_ARGS")
  assert_output_contains "cargo test -p integration-tests --features integration-tests/run-showcase-tests --foo --bar" "$output2" "main cargo test 2" || return 1
  unset DRY_RUN
}

# Tests the git fetch command in dry-run mode.
test_main_git_fetch_dry_run() {
  export DRY_RUN=true
  local output=$(run_cmd "git fetch upstream")
  assert_output_contains "git fetch upstream" "$output" "main git fetch dry run" || return 1
  unset DRY_RUN
}

# Tests the git reset command in dry-run mode.
test_main_git_reset_dry_run() {
  export DRY_RUN=true
  local output=$(run_cmd "git reset --hard upstream/main")
  assert_output_contains "git reset --hard upstream/main" "$output" "main git reset dry run" || return 1
  unset DRY_RUN
}

main "$@"
