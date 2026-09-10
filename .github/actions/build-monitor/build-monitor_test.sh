#!/usr/bin/env bash
# Copyright 2026 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     https://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
MONITOR_SCRIPT="$SCRIPT_DIR/build-monitor.sh"

TEST_TMP="$(mktemp -d)"
trap 'rm -rf "$TEST_TMP"' EXIT

MOCK_BIN="$TEST_TMP/bin"
mkdir -p "$MOCK_BIN"
MOCK_LOG="$TEST_TMP/gh_calls.log"

# Create mock gh CLI
cat << 'EOF' > "$MOCK_BIN/gh"
#!/usr/bin/env bash
set -euo pipefail

LOG_FILE="${TEST_GH_LOG:-/dev/null}"
echo "$@" >> "$LOG_FILE"

case "${1:-}" in
  issue)
    subcmd="${2:-}"
    case "$subcmd" in
      list)
        if [[ -n "${MOCK_EXISTING_ISSUE:-}" ]]; then
          echo "$MOCK_EXISTING_ISSUE"
        else
          echo "[]"
        fi
        exit 0
        ;;
      create)
        # Check if we should simulate assignee failure
        if [[ "${FAIL_ON_ASSIGNEE:-false}" == "true" ]] && [[ "$*" =~ --assignee ]]; then
          echo "GraphQL error: could not add assignee" >&2
          exit 1
        fi
        # Check if we should simulate label failure
        if [[ "${FAIL_ON_LABEL:-false}" == "true" ]] && [[ "$*" =~ --label ]]; then
          echo "GraphQL error: label not found" >&2
          exit 1
        fi
        echo "https://github.com/${GH_REPO:-test/repo}/issues/42"
        exit 0
        ;;
      comment)
        exit 0
        ;;
      *)
        echo "Unknown gh issue command: $subcmd" >&2
        exit 1
        ;;
    esac
    ;;
  api)
    endpoint="${2:-}"
    case "$endpoint" in
      *check-suites/*/check-runs*)
        cat << 'JSON'
{
  "check_runs": [
    {
      "name": "build-docker-image",
      "conclusion": "failure",
      "details_url": "https://console.cloud.google.com/cloud-build/builds/12345"
    }
  ]
}
JSON
        exit 0
        ;;
      *actions/runs/*/jobs*)
        cat << 'JSON'
{
  "jobs": [
    {
      "name": "Integration Tests",
      "conclusion": "failure",
      "html_url": "https://github.com/test/repo/actions/runs/1/job/2"
    }
  ]
}
JSON
        exit 0
        ;;
      *commits/*/pulls*)
        cat << 'JSON'
[
  {
    "number": 101,
    "html_url": "https://github.com/test/repo/pull/101"
  }
]
JSON
        exit 0
        ;;
      *commits/*)
        if [[ "${MOCK_COMMIT_AUTHOR_BOT:-false}" == "true" ]]; then
          cat << 'JSON'
{
  "author": {
    "login": "release-please[bot]"
  }
}
JSON
        else
          cat << 'JSON'
{
  "author": {
    "login": "octocat"
  }
}
JSON
        fi
        exit 0
        ;;
      *)
        echo "{}"
        exit 0
        ;;
    esac
    ;;
  *)
    exit 0
    ;;
esac
EOF

chmod +x "$MOCK_BIN/gh"
export PATH="$MOCK_BIN:$PATH"
export TEST_GH_LOG="$MOCK_LOG"

TESTS_PASSED=0
TESTS_FAILED=0

run_test() {
  local test_name="$1"
  shift
  echo -n "Running $test_name... "
  rm -f "$MOCK_LOG"
  touch "$MOCK_LOG"
  if "$@"; then
    echo "PASSED"
    TESTS_PASSED=$((TESTS_PASSED + 1))
  else
    echo "FAILED"
    TESTS_FAILED=$((TESTS_FAILED + 1))
  fi
}

# Test 1: check_suite conclusion=success should skip
test_check_suite_success() {
  output=$(EVENT_NAME="check_suite" \
    EVENT_ACTION="completed" \
    CHECK_SUITE_CONCLUSION="success" \
    CHECK_SUITE_HEAD_BRANCH="main" \
    TARGET_BRANCH="main" \
    CHECK_SUITE_APP_NAME="Google Cloud Build" \
    CHECK_SUITE_APP_NAME_ACTUAL="Google Cloud Build" \
    GH_REPO="googleapis/test-repo" \
    bash "$MONITOR_SCRIPT")
  [[ "$output" =~ "Skipping" ]]
  ! grep -q "issue create" "$MOCK_LOG"
  ! grep -q "issue comment" "$MOCK_LOG"
}

# Test 2: check_suite branch != main should skip
test_check_suite_wrong_branch() {
  output=$(EVENT_NAME="check_suite" \
    EVENT_ACTION="completed" \
    CHECK_SUITE_CONCLUSION="failure" \
    CHECK_SUITE_HEAD_BRANCH="feat/something" \
    TARGET_BRANCH="main" \
    CHECK_SUITE_APP_NAME="Google Cloud Build" \
    CHECK_SUITE_APP_NAME_ACTUAL="Google Cloud Build" \
    GH_REPO="googleapis/test-repo" \
    bash "$MONITOR_SCRIPT")
  [[ "$output" =~ "Skipping" ]]
  ! grep -q "issue create" "$MOCK_LOG"
}

# Test 3: check_suite wrong app should skip
test_check_suite_wrong_app() {
  output=$(EVENT_NAME="check_suite" \
    EVENT_ACTION="completed" \
    CHECK_SUITE_CONCLUSION="failure" \
    CHECK_SUITE_HEAD_BRANCH="main" \
    TARGET_BRANCH="main" \
    CHECK_SUITE_APP_NAME="Google Cloud Build" \
    CHECK_SUITE_APP_NAME_ACTUAL="Dependabot" \
    CHECK_SUITE_APP_SLUG="dependabot" \
    GH_REPO="googleapis/test-repo" \
    bash "$MONITOR_SCRIPT")
  [[ "$output" =~ "Skipping" ]]
  ! grep -q "issue create" "$MOCK_LOG"
}

# Test 4: check_suite slug matching should succeed
test_check_suite_slug_match() {
  output=$(EVENT_NAME="check_suite" \
    EVENT_ACTION="completed" \
    CHECK_SUITE_CONCLUSION="failure" \
    CHECK_SUITE_HEAD_BRANCH="main" \
    TARGET_BRANCH="main" \
    CHECK_SUITE_APP_NAME="Google Cloud Build" \
    CHECK_SUITE_APP_NAME_ACTUAL="Google-Cloud-Build" \
    CHECK_SUITE_APP_SLUG="google-cloud-build" \
    CHECK_SUITE_ID="123" \
    CHECK_SUITE_HEAD_SHA="abcdef123456" \
    GH_REPO="googleapis/test-repo" \
    bash "$MONITOR_SCRIPT")
  [[ "$output" =~ "Created issue" ]]
  grep -q "issue create" "$MOCK_LOG"
}

# Test 5: check_suite wrong head repo should skip
test_check_suite_fork_repo() {
  output=$(EVENT_NAME="check_suite" \
    EVENT_ACTION="completed" \
    CHECK_SUITE_CONCLUSION="failure" \
    CHECK_SUITE_HEAD_BRANCH="main" \
    TARGET_BRANCH="main" \
    CHECK_SUITE_APP_NAME="Google Cloud Build" \
    CHECK_SUITE_APP_NAME_ACTUAL="Google Cloud Build" \
    GITHUB_REPOSITORY="fork-user/test-repo" \
    GH_REPO="googleapis/test-repo" \
    bash "$MONITOR_SCRIPT")
  [[ "$output" =~ "Skipping" ]]
  ! grep -q "issue create" "$MOCK_LOG"
}

# Test 6: check_suite with active open PR should skip
test_check_suite_open_pr_skip() {
  output=$(EVENT_NAME="check_suite" \
    EVENT_ACTION="completed" \
    CHECK_SUITE_CONCLUSION="failure" \
    CHECK_SUITE_HEAD_BRANCH="main" \
    TARGET_BRANCH="main" \
    CHECK_SUITE_APP_NAME="Google Cloud Build" \
    CHECK_SUITE_APP_NAME_ACTUAL="Google Cloud Build" \
    CHECK_SUITE_PULL_REQUESTS='[{"number":50,"state":"open","head":{"repo":{"full_name":"googleapis/test-repo"}}}]' \
    GH_REPO="googleapis/test-repo" \
    bash "$MONITOR_SCRIPT")
  [[ "$output" =~ "Skipping" ]]
  ! grep -q "issue create" "$MOCK_LOG"
}

# Test 7: check_suite with PR from fork repo should skip
test_check_suite_fork_pr_skip() {
  output=$(EVENT_NAME="check_suite" \
    EVENT_ACTION="completed" \
    CHECK_SUITE_CONCLUSION="failure" \
    CHECK_SUITE_HEAD_BRANCH="main" \
    TARGET_BRANCH="main" \
    CHECK_SUITE_APP_NAME="Google Cloud Build" \
    CHECK_SUITE_APP_NAME_ACTUAL="Google Cloud Build" \
    CHECK_SUITE_PULL_REQUESTS='[{"number":50,"state":"closed","head":{"repo":{"full_name":"attacker/test-repo"}}}]' \
    GH_REPO="googleapis/test-repo" \
    bash "$MONITOR_SCRIPT")
  [[ "$output" =~ "Skipping" ]]
  ! grep -q "issue create" "$MOCK_LOG"
}

# Test 8: check_suite non-completed action should skip
test_check_suite_non_completed_action() {
  output=$(EVENT_NAME="check_suite" \
    EVENT_ACTION="requested" \
    CHECK_SUITE_CONCLUSION="failure" \
    CHECK_SUITE_HEAD_BRANCH="main" \
    TARGET_BRANCH="main" \
    CHECK_SUITE_APP_NAME="Google Cloud Build" \
    CHECK_SUITE_APP_NAME_ACTUAL="Google Cloud Build" \
    GH_REPO="googleapis/test-repo" \
    bash "$MONITOR_SCRIPT")
  [[ "$output" =~ "Skipping" ]]
  ! grep -q "issue create" "$MOCK_LOG"
}

# Test 9: check_suite failure on main creates new issue and posts comment
test_check_suite_failure_new_issue() {
  output=$(EVENT_NAME="check_suite" \
    EVENT_ACTION="completed" \
    CHECK_SUITE_CONCLUSION="failure" \
    CHECK_SUITE_HEAD_BRANCH="main" \
    TARGET_BRANCH="main" \
    CHECK_SUITE_APP_NAME="Google Cloud Build" \
    CHECK_SUITE_APP_NAME_ACTUAL="Google Cloud Build" \
    CHECK_SUITE_ID="123" \
    CHECK_SUITE_HEAD_SHA="abcdef123456" \
    GH_REPO="googleapis/test-repo" \
    TEAM_MENTION="@googleapis/cloud-sdk-rust-team" \
    bash "$MONITOR_SCRIPT")
  [[ "$output" =~ "Created issue" ]]
  grep -q "issue create" "$MOCK_LOG"
  grep -q "issue comment 42" "$MOCK_LOG"
}

# Test 10: check_suite failure on main updates existing open issue
test_check_suite_failure_existing_issue() {
  output=$(EVENT_NAME="check_suite" \
    EVENT_ACTION="completed" \
    CHECK_SUITE_CONCLUSION="failure" \
    CHECK_SUITE_HEAD_BRANCH="main" \
    TARGET_BRANCH="main" \
    CHECK_SUITE_APP_NAME="Google Cloud Build" \
    CHECK_SUITE_APP_NAME_ACTUAL="Google Cloud Build" \
    CHECK_SUITE_ID="123" \
    CHECK_SUITE_HEAD_SHA="abcdef123456" \
    GH_REPO="googleapis/test-repo" \
    MOCK_EXISTING_ISSUE='[{"number":77,"title":"test-repo: Post-merge build failure on main"}]' \
    bash "$MONITOR_SCRIPT")
  [[ "$output" =~ "Found existing open issue #77" ]]
  ! grep -q "issue create" "$MOCK_LOG"
  grep -q "issue comment 77" "$MOCK_LOG"
}

# Test 11: check_suite exact title match prevents commenting on partial match
test_check_suite_exact_title_match_only() {
  output=$(EVENT_NAME="check_suite" \
    EVENT_ACTION="completed" \
    CHECK_SUITE_CONCLUSION="failure" \
    CHECK_SUITE_HEAD_BRANCH="main" \
    TARGET_BRANCH="main" \
    CHECK_SUITE_APP_NAME="Google Cloud Build" \
    CHECK_SUITE_APP_NAME_ACTUAL="Google Cloud Build" \
    CHECK_SUITE_ID="123" \
    CHECK_SUITE_HEAD_SHA="abcdef123456" \
    GH_REPO="googleapis/test-repo" \
    MOCK_EXISTING_ISSUE='[{"number":77,"title":"test-repo: Post-merge build failure on main (old/different)"}]' \
    bash "$MONITOR_SCRIPT")
  [[ "$output" =~ "Created issue" ]]
  grep -q "issue create" "$MOCK_LOG"
}

# Test 12: workflow_run conclusion=success should skip
test_workflow_run_success() {
  output=$(EVENT_NAME="workflow_run" \
    EVENT_ACTION="completed" \
    WORKFLOW_RUN_CONCLUSION="success" \
    WORKFLOW_RUN_HEAD_BRANCH="main" \
    TARGET_BRANCH="main" \
    WORKFLOW_RUN_EVENT="push" \
    WORKFLOW_RUN_EVENT_ACTUAL="push" \
    GH_REPO="googleapis/test-repo" \
    bash "$MONITOR_SCRIPT")
  [[ "$output" =~ "Skipping" ]]
  ! grep -q "issue create" "$MOCK_LOG"
}

# Test 13: workflow_run wrong branch should skip
test_workflow_run_wrong_branch() {
  output=$(EVENT_NAME="workflow_run" \
    EVENT_ACTION="completed" \
    WORKFLOW_RUN_CONCLUSION="failure" \
    WORKFLOW_RUN_HEAD_BRANCH="feat/something" \
    TARGET_BRANCH="main" \
    WORKFLOW_RUN_EVENT="push" \
    WORKFLOW_RUN_EVENT_ACTUAL="push" \
    GH_REPO="googleapis/test-repo" \
    bash "$MONITOR_SCRIPT")
  [[ "$output" =~ "Skipping" ]]
  ! grep -q "issue create" "$MOCK_LOG"
}

# Test 14: workflow_run pull_request event should skip
test_workflow_run_pr_event() {
  output=$(EVENT_NAME="workflow_run" \
    EVENT_ACTION="completed" \
    WORKFLOW_RUN_CONCLUSION="failure" \
    WORKFLOW_RUN_HEAD_BRANCH="main" \
    TARGET_BRANCH="main" \
    WORKFLOW_RUN_EVENT="push" \
    WORKFLOW_RUN_EVENT_ACTUAL="pull_request" \
    GH_REPO="googleapis/test-repo" \
    bash "$MONITOR_SCRIPT")
  [[ "$output" =~ "Skipping" ]]
  ! grep -q "issue create" "$MOCK_LOG"
}

# Test 15: workflow_run fork head repo should skip
test_workflow_run_fork_head_repo() {
  output=$(EVENT_NAME="workflow_run" \
    EVENT_ACTION="completed" \
    WORKFLOW_RUN_CONCLUSION="failure" \
    WORKFLOW_RUN_HEAD_BRANCH="main" \
    TARGET_BRANCH="main" \
    WORKFLOW_RUN_EVENT="push" \
    WORKFLOW_RUN_EVENT_ACTUAL="push" \
    WORKFLOW_RUN_HEAD_REPO="fork-user/test-repo" \
    GH_REPO="googleapis/test-repo" \
    bash "$MONITOR_SCRIPT")
  [[ "$output" =~ "Skipping" ]]
  ! grep -q "issue create" "$MOCK_LOG"
}

# Test 16: workflow_run non-completed action should skip
test_workflow_run_non_completed_action() {
  output=$(EVENT_NAME="workflow_run" \
    EVENT_ACTION="in_progress" \
    WORKFLOW_RUN_CONCLUSION="failure" \
    WORKFLOW_RUN_HEAD_BRANCH="main" \
    TARGET_BRANCH="main" \
    WORKFLOW_RUN_EVENT="push" \
    WORKFLOW_RUN_EVENT_ACTUAL="push" \
    GH_REPO="googleapis/test-repo" \
    bash "$MONITOR_SCRIPT")
  [[ "$output" =~ "Skipping" ]]
  ! grep -q "issue create" "$MOCK_LOG"
}

# Test 17: workflow_run failure creates issue
test_workflow_run_failure() {
  output=$(EVENT_NAME="workflow_run" \
    EVENT_ACTION="completed" \
    WORKFLOW_RUN_CONCLUSION="failure" \
    WORKFLOW_RUN_HEAD_BRANCH="main" \
    TARGET_BRANCH="main" \
    WORKFLOW_RUN_EVENT="push" \
    WORKFLOW_RUN_EVENT_ACTUAL="push" \
    WORKFLOW_RUN_ID="999" \
    WORKFLOW_RUN_NAME="CI" \
    WORKFLOW_RUN_HEAD_SHA="11223344" \
    WORKFLOW_RUN_URL="https://github.com/test/repo/actions/runs/999" \
    GH_REPO="googleapis/test-repo" \
    bash "$MONITOR_SCRIPT")
  [[ "$output" =~ "Created issue" ]]
  grep -q "issue create" "$MOCK_LOG"
}

# Test 18: workflow_run assigns WORKFLOW_RUN_ACTOR when commit author is a bot
test_workflow_run_actor_assignee() {
  output=$(EVENT_NAME="workflow_run" \
    EVENT_ACTION="completed" \
    WORKFLOW_RUN_CONCLUSION="failure" \
    WORKFLOW_RUN_HEAD_BRANCH="main" \
    TARGET_BRANCH="main" \
    WORKFLOW_RUN_EVENT="push" \
    WORKFLOW_RUN_EVENT_ACTUAL="push" \
    WORKFLOW_RUN_ID="999" \
    WORKFLOW_RUN_HEAD_SHA="11223344" \
    WORKFLOW_RUN_ACTOR="human-dev" \
    MOCK_COMMIT_AUTHOR_BOT="true" \
    GH_REPO="googleapis/test-repo" \
    bash "$MONITOR_SCRIPT")
  [[ "$output" =~ "Attempting to create issue assigned to human-dev" ]]
  grep -q "issue create" "$MOCK_LOG"
  grep -q -- "--assignee human-dev" "$MOCK_LOG"
}

# Test 19: assignee retry fallback
test_assignee_retry_fallback() {
  output=$(EVENT_NAME="workflow_run" \
    EVENT_ACTION="completed" \
    WORKFLOW_RUN_CONCLUSION="failure" \
    WORKFLOW_RUN_HEAD_BRANCH="main" \
    TARGET_BRANCH="main" \
    WORKFLOW_RUN_EVENT="push" \
    WORKFLOW_RUN_EVENT_ACTUAL="push" \
    WORKFLOW_RUN_ID="999" \
    GH_REPO="googleapis/test-repo" \
    INPUT_ASSIGNEE="external-contributor" \
    FAIL_ON_ASSIGNEE="true" \
    bash "$MONITOR_SCRIPT")
  [[ "$output" =~ "Attempting to create issue without assignee" ]]
  grep -q "issue create" "$MOCK_LOG"
}

# Test 20: label retry fallback when label doesn't exist in repo
test_label_retry_fallback() {
  output=$(EVENT_NAME="workflow_run" \
    EVENT_ACTION="completed" \
    WORKFLOW_RUN_CONCLUSION="failure" \
    WORKFLOW_RUN_HEAD_BRANCH="main" \
    TARGET_BRANCH="main" \
    WORKFLOW_RUN_EVENT="push" \
    WORKFLOW_RUN_EVENT_ACTUAL="push" \
    WORKFLOW_RUN_ID="999" \
    GH_REPO="googleapis/test-repo" \
    ISSUE_LABELS="nonexistent-label" \
    FAIL_ON_LABEL="true" \
    bash "$MONITOR_SCRIPT")
  [[ "$output" =~ "Attempting to create issue without label" ]]
  grep -q "issue create" "$MOCK_LOG"
}

# Test 21: empty labels does not pass --label flag
test_empty_labels_input() {
  output=$(EVENT_NAME="workflow_run" \
    EVENT_ACTION="completed" \
    WORKFLOW_RUN_CONCLUSION="failure" \
    WORKFLOW_RUN_HEAD_BRANCH="main" \
    TARGET_BRANCH="main" \
    WORKFLOW_RUN_EVENT="push" \
    WORKFLOW_RUN_EVENT_ACTUAL="push" \
    WORKFLOW_RUN_ID="999" \
    GH_REPO="googleapis/test-repo" \
    ISSUE_LABELS="" \
    bash "$MONITOR_SCRIPT")
  [[ "$output" =~ "Created issue" ]]
  ! grep -q -- "--label" "$MOCK_LOG"
}

# Test 22: unsupported event should skip
test_unsupported_event() {
  output=$(EVENT_NAME="pull_request" \
    GH_REPO="googleapis/test-repo" \
    bash "$MONITOR_SCRIPT")
  [[ "$output" =~ "Skipping" ]]
  ! grep -q "issue create" "$MOCK_LOG"
}

run_test "check_suite_success" test_check_suite_success
run_test "check_suite_wrong_branch" test_check_suite_wrong_branch
run_test "check_suite_wrong_app" test_check_suite_wrong_app
run_test "check_suite_slug_match" test_check_suite_slug_match
run_test "check_suite_fork_repo" test_check_suite_fork_repo
run_test "check_suite_open_pr_skip" test_check_suite_open_pr_skip
run_test "check_suite_fork_pr_skip" test_check_suite_fork_pr_skip
run_test "check_suite_non_completed_action" test_check_suite_non_completed_action
run_test "check_suite_failure_new_issue" test_check_suite_failure_new_issue
run_test "check_suite_failure_existing_issue" test_check_suite_failure_existing_issue
run_test "check_suite_exact_title_match_only" test_check_suite_exact_title_match_only
run_test "workflow_run_success" test_workflow_run_success
run_test "workflow_run_wrong_branch" test_workflow_run_wrong_branch
run_test "workflow_run_pr_event" test_workflow_run_pr_event
run_test "workflow_run_fork_head_repo" test_workflow_run_fork_head_repo
run_test "workflow_run_non_completed_action" test_workflow_run_non_completed_action
run_test "workflow_run_failure" test_workflow_run_failure
run_test "workflow_run_actor_assignee" test_workflow_run_actor_assignee
run_test "assignee_retry_fallback" test_assignee_retry_fallback
run_test "label_retry_fallback" test_label_retry_fallback
run_test "empty_labels_input" test_empty_labels_input
run_test "unsupported_event" test_unsupported_event

echo ""
echo "=== Test Summary ==="
echo "Passed: $TESTS_PASSED"
echo "Failed: $TESTS_FAILED"

if [[ "$TESTS_FAILED" -gt 0 ]]; then
  exit 1
fi
