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

GH_REPO="${GH_REPO:-${GITHUB_REPOSITORY:-}}"
TARGET_BRANCH="${TARGET_BRANCH:-main}"
CHECK_SUITE_APP_NAME="${CHECK_SUITE_APP_NAME:-Google Cloud Build}"
WORKFLOW_RUN_EVENT="${WORKFLOW_RUN_EVENT:-push}"
ISSUE_LABELS="${ISSUE_LABELS-:rotating_light: critical}"
EVENT_NAME="${EVENT_NAME:-}"
EVENT_ACTION="${EVENT_ACTION:-}"

if [[ -z "$GH_REPO" ]]; then
  echo "Error: GH_REPO (or GITHUB_REPOSITORY) is required." >&2
  exit 1
fi

# Prevent execution on forks when target repo is specified
if [[ -n "${GITHUB_REPOSITORY:-}" && "$GITHUB_REPOSITORY" != "$GH_REPO" ]]; then
  echo "Current repository '$GITHUB_REPOSITORY' does not match target repo '$GH_REPO'. Skipping."
  exit 0
fi

# Ensure only completed events are processed
if [[ -n "$EVENT_ACTION" && "$EVENT_ACTION" != "completed" ]]; then
  echo "Event action '$EVENT_ACTION' is not completed. Skipping."
  exit 0
fi

slugify() {
  echo "$1" | tr '[:upper:]' '[:lower:]' | tr ' _' '--' | tr -cd '[:alnum:]-'
}

COMMIT_SHA=""
BODY=""

case "$EVENT_NAME" in
  check_suite)
    # Validate conclusion
    case "${CHECK_SUITE_CONCLUSION:-}" in
      failure|timed_out|cancelled)
        ;;
      *)
        echo "Check suite conclusion '${CHECK_SUITE_CONCLUSION:-}' is not a failure. Skipping."
        exit 0
        ;;
    esac

    # Validate target branch
    if [[ "${CHECK_SUITE_HEAD_BRANCH:-}" != "$TARGET_BRANCH" ]]; then
      echo "Check suite branch '${CHECK_SUITE_HEAD_BRANCH:-}' does not match target branch '$TARGET_BRANCH'. Skipping."
      exit 0
    fi

    # Validate app name or slug
    APP_NAME_ACTUAL="${CHECK_SUITE_APP_NAME_ACTUAL:-}"
    APP_SLUG_ACTUAL="${CHECK_SUITE_APP_SLUG:-}"
    EXPECTED_SLUG=$(slugify "$CHECK_SUITE_APP_NAME")
    ACTUAL_NAME_SLUG=$(slugify "$APP_NAME_ACTUAL")
    if [[ "$APP_NAME_ACTUAL" != "$CHECK_SUITE_APP_NAME" && "$APP_SLUG_ACTUAL" != "$EXPECTED_SLUG" && "$ACTUAL_NAME_SLUG" != "$EXPECTED_SLUG" ]]; then
      echo "Check suite app '$APP_NAME_ACTUAL' (slug: '$APP_SLUG_ACTUAL') does not match expected app '$CHECK_SUITE_APP_NAME'. Skipping."
      exit 0
    fi

    # Security check: Filter out check suites that belong to open pull requests or PRs from forks
    if [[ -n "${CHECK_SUITE_PULL_REQUESTS:-}" && "$CHECK_SUITE_PULL_REQUESTS" != "[]" ]]; then
      IS_OPEN_OR_FORK_PR=$(echo "$CHECK_SUITE_PULL_REQUESTS" | jq -r --arg repo "$GH_REPO" '
        [ .[]? | select(.state == "open" or .head.repo.full_name != $repo) ] | length > 0
      ' 2>/dev/null || echo "false")
      if [[ "$IS_OPEN_OR_FORK_PR" == "true" ]]; then
        echo "Check suite is associated with an active pull request or fork. Skipping."
        exit 0
      fi
    fi

    COMMIT_SHA="${CHECK_SUITE_HEAD_SHA:-}"

    # Extract failed check runs from the check suite
    FAILED_BUILDS=""
    if [[ -n "${CHECK_SUITE_ID:-}" ]]; then
      CHECK_RUNS_JSON=""
      if RUNS_OUT=$(gh api "repos/$GH_REPO/check-suites/$CHECK_SUITE_ID/check-runs" --paginate 2>/dev/null); then
        CHECK_RUNS_JSON="$RUNS_OUT"
      fi

      if [[ -n "$CHECK_RUNS_JSON" ]]; then
        FAILED_RUNS=$(echo "$CHECK_RUNS_JSON" | jq -c '.check_runs[]? | select(.conclusion == "failure" or .conclusion == "timed_out" or .conclusion == "cancelled") | {name: .name, url: (.details_url // .html_url // "")}' 2>/dev/null || true)
        if [[ -n "$FAILED_RUNS" ]]; then
          while IFS= read -r run_item; do
            if [[ -n "$run_item" ]]; then
              name=$(echo "$run_item" | jq -r '.name' 2>/dev/null || true)
              url=$(echo "$run_item" | jq -r '.url' 2>/dev/null || true)
              if [[ -n "$name" ]]; then
                if [[ -n "$url" ]]; then
                  FAILED_BUILDS="$FAILED_BUILDS"$'\n'"- [\`$name\`]($url)"
                else
                  FAILED_BUILDS="$FAILED_BUILDS"$'\n'"- \`$name\`"
                fi
              fi
            fi
          done < <(echo "$FAILED_RUNS")
        fi
      fi
    fi

    if [[ -z "$FAILED_BUILDS" ]]; then
      CHECK_FALLBACK_URL="https://github.com/$GH_REPO/commit/$COMMIT_SHA/checks"
      FAILED_BUILDS=$'\n'"- [Check Suite Logs]($CHECK_FALLBACK_URL)"
    fi

    BODY="**Failure in $CHECK_SUITE_APP_NAME on \`$TARGET_BRANCH\`:**

**Failed Builds:**$FAILED_BUILDS

**Commit:** $COMMIT_SHA"
    ;;

  workflow_run)
    # Validate conclusion
    case "${WORKFLOW_RUN_CONCLUSION:-}" in
      failure|timed_out|cancelled|startup_failure)
        ;;
      *)
        echo "Workflow run conclusion '${WORKFLOW_RUN_CONCLUSION:-}' is not a failure. Skipping."
        exit 0
        ;;
    esac

    # Validate target branch
    if [[ "${WORKFLOW_RUN_HEAD_BRANCH:-}" != "$TARGET_BRANCH" ]]; then
      echo "Workflow run branch '${WORKFLOW_RUN_HEAD_BRANCH:-}' does not match target branch '$TARGET_BRANCH'. Skipping."
      exit 0
    fi

    # Validate triggering event (must be push by default to ignore pull_request runs)
    if [[ "${WORKFLOW_RUN_EVENT_ACTUAL:-}" != "$WORKFLOW_RUN_EVENT" ]]; then
      echo "Workflow run triggering event '${WORKFLOW_RUN_EVENT_ACTUAL:-}' does not match expected event '$WORKFLOW_RUN_EVENT'. Skipping."
      exit 0
    fi

    # Validate repository to prevent triggers from fork workflow runs
    if [[ -n "${WORKFLOW_RUN_HEAD_REPO:-}" && "$WORKFLOW_RUN_HEAD_REPO" != "$GH_REPO" ]]; then
      echo "Workflow run repository '$WORKFLOW_RUN_HEAD_REPO' does not match target repo '$GH_REPO'. Skipping."
      exit 0
    fi

    # Security check: Filter out workflow runs that belong to open pull requests
    if [[ -n "${WORKFLOW_RUN_PULL_REQUESTS:-}" && "$WORKFLOW_RUN_PULL_REQUESTS" != "[]" ]]; then
      IS_OPEN_PR=$(echo "$WORKFLOW_RUN_PULL_REQUESTS" | jq -r '
        [ .[]? | select(.state == "open") ] | length > 0
      ' 2>/dev/null || echo "false")
      if [[ "$IS_OPEN_PR" == "true" ]]; then
        echo "Workflow run is associated with an active pull request. Skipping."
        exit 0
      fi
    fi

    COMMIT_SHA="${WORKFLOW_RUN_HEAD_SHA:-}"

    # Extract failed jobs from the workflow run
    FAILED_JOBS=""
    if [[ -n "${WORKFLOW_RUN_ID:-}" ]]; then
      JOBS_JSON=""
      if JOBS_OUT=$(gh api "repos/$GH_REPO/actions/runs/$WORKFLOW_RUN_ID/jobs" --paginate 2>/dev/null); then
        JOBS_JSON="$JOBS_OUT"
      fi

      if [[ -n "$JOBS_JSON" ]]; then
        FAILED_ITEMS=$(echo "$JOBS_JSON" | jq -c '.jobs[]? | select(.conclusion == "failure" or .conclusion == "timed_out" or .conclusion == "cancelled" or .conclusion == "startup_failure") | {name: .name, url: (.html_url // "")}' 2>/dev/null || true)
        if [[ -n "$FAILED_ITEMS" ]]; then
          while IFS= read -r item; do
            if [[ -n "$item" ]]; then
              name=$(echo "$item" | jq -r '.name' 2>/dev/null || true)
              url=$(echo "$item" | jq -r '.url' 2>/dev/null || true)
              if [[ -n "$name" ]]; then
                if [[ -n "$url" ]]; then
                  FAILED_JOBS="$FAILED_JOBS"$'\n'"- [\`$name\`]($url)"
                else
                  FAILED_JOBS="$FAILED_JOBS"$'\n'"- \`$name\`"
                fi
              fi
            fi
          done < <(echo "$FAILED_ITEMS")
        fi
      fi
    fi

    if [[ -z "$FAILED_JOBS" ]]; then
      if [[ -n "${WORKFLOW_RUN_URL:-}" ]]; then
        FAILED_JOBS=$'\n'"- [Workflow Run Logs]($WORKFLOW_RUN_URL)"
      else
        FAILED_JOBS=$'\n'"- Workflow run failed (details url unavailable)"
      fi
    fi

    WORKFLOW_NAME="${WORKFLOW_RUN_NAME:-GitHub Actions}"
    WORKFLOW_URL="${WORKFLOW_RUN_URL:-}"
    if [[ -n "$WORKFLOW_URL" ]]; then
      WORKFLOW_TEXT="**Workflow:** $WORKFLOW_NAME ([Run Logs]($WORKFLOW_URL))"
    else
      WORKFLOW_TEXT="**Workflow:** $WORKFLOW_NAME"
    fi

    BODY="**Failure in GitHub Actions on \`$TARGET_BRANCH\`:**

$WORKFLOW_TEXT

**Failed Jobs:**$FAILED_JOBS

**Commit:** $COMMIT_SHA"
    ;;

  push|workflow_dispatch)
    COMMIT_SHA="${GITHUB_SHA:-}"
    RUN_URL="${GITHUB_SERVER_URL:-https://github.com}/$GH_REPO/actions/runs/${GITHUB_RUN_ID:-}"
    BODY="**CI failure on \`$TARGET_BRANCH\`:**

**Workflow Run:** $RUN_URL

**Commit:** $COMMIT_SHA"
    ;;

  *)
    echo "Event '$EVENT_NAME' is not monitored. Skipping."
    exit 0
    ;;
esac

# Find associated Pull Requests
PR_TEXT=""
if [[ -n "$COMMIT_SHA" ]]; then
  PRS=""
  if PRS_OUT=$(gh api "repos/$GH_REPO/commits/$COMMIT_SHA/pulls" 2>/dev/null); then
    PRS=$(echo "$PRS_OUT" | jq -r '.[]? | .number // empty' 2>/dev/null || true)
  fi
  if [[ -n "$PRS" ]]; then
    for pr in $(echo "$PRS" | tr ' ' '\n' | sort -u -n); do
      if [[ -n "$pr" && "$pr" != "null" ]]; then
        PR_TEXT="$PR_TEXT"$'\n'"- #$pr"
      fi
    done
  fi
fi

# Fall back to merged pull requests from event payload if commit PRs query was empty
if [[ -z "$PR_TEXT" ]]; then
  PAYLOAD_PRS="${CHECK_SUITE_PULL_REQUESTS:-${WORKFLOW_RUN_PULL_REQUESTS:-}}"
  if [[ -n "$PAYLOAD_PRS" && "$PAYLOAD_PRS" != "[]" ]]; then
    PRS=$(echo "$PAYLOAD_PRS" | jq -r '.[]? | .number // empty' 2>/dev/null || true)
    if [[ -n "$PRS" ]]; then
      for pr in $(echo "$PRS" | tr ' ' '\n' | sort -u -n); do
        if [[ -n "$pr" && "$pr" != "null" ]]; then
          PR_TEXT="$PR_TEXT"$'\n'"- #$pr"
        fi
      done
    fi
  fi
fi

if [[ -n "$PR_TEXT" ]]; then
  BODY="$BODY

**Associated Pull Requests:**$PR_TEXT"
fi

# Determine issue title
if [[ -n "${INPUT_ISSUE_TITLE:-}" ]]; then
  TITLE="$INPUT_ISSUE_TITLE"
else
  REPO_NAME="${GH_REPO##*/}"
  TITLE="$REPO_NAME: Post-merge build failure on $TARGET_BRANCH"
fi

# Check for existing open issue with the exact same title (prevent partial token match)
EXISTING_ISSUE=""
if ISSUES_OUT=$(gh issue list --repo "$GH_REPO" --search "\"$TITLE\" in:title" --state open --json number,title 2>/dev/null); then
  EXISTING_ISSUE=$(echo "$ISSUES_OUT" | jq -r --arg title "$TITLE" '[.[]? | select(.title == $title)][0].number // empty' 2>/dev/null || true)
fi

if [[ -n "$EXISTING_ISSUE" && "$EXISTING_ISSUE" != "null" ]]; then
  echo "Found existing open issue #$EXISTING_ISSUE. Adding comment."
  COMMENT_BODY="The build failed again on \`$TARGET_BRANCH\`.

$BODY"
  gh issue comment "$EXISTING_ISSUE" --repo "$GH_REPO" --body "$COMMENT_BODY"
  exit 0
fi

echo "No open issue found with title '$TITLE'. Creating new issue."

# Resolve assignee: explicitly provided -> commit author -> workflow run actor -> event actor
ASSIGNEE="${INPUT_ASSIGNEE:-}"
if [[ -z "$ASSIGNEE" && -n "$COMMIT_SHA" ]]; then
  COMMIT_AUTHOR=""
  if COMMIT_OUT=$(gh api "repos/$GH_REPO/commits/$COMMIT_SHA" 2>/dev/null); then
    COMMIT_AUTHOR=$(echo "$COMMIT_OUT" | jq -r '.author.login // empty' 2>/dev/null || true)
  fi
  if [[ -n "$COMMIT_AUTHOR" && ! "$COMMIT_AUTHOR" =~ \[bot\]$ ]]; then
    ASSIGNEE="$COMMIT_AUTHOR"
  fi
fi
if [[ -z "$ASSIGNEE" && -n "${WORKFLOW_RUN_ACTOR:-}" && ! "$WORKFLOW_RUN_ACTOR" =~ \[bot\]$ ]]; then
  ASSIGNEE="$WORKFLOW_RUN_ACTOR"
fi
if [[ -z "$ASSIGNEE" && -n "${GITHUB_ACTOR:-}" && ! "$GITHUB_ACTOR" =~ \[bot\]$ ]]; then
  ASSIGNEE="$GITHUB_ACTOR"
fi

create_issue() {
  local target_assignee="$1"
  local target_labels="$2"
  local cmd=(gh issue create --title "$TITLE" --body "$BODY" --repo "$GH_REPO")
  if [[ -n "$target_labels" ]]; then
    cmd+=(--label "$target_labels")
  fi
  if [[ -n "$target_assignee" ]]; then
    cmd+=(--assignee "$target_assignee")
  fi
  "${cmd[@]}"
}

ISSUE_LINK=""

# Tier 1: Try with assignee and labels
if [[ -n "$ASSIGNEE" ]]; then
  echo "Attempting to create issue assigned to $ASSIGNEE..."
  ISSUE_LINK=$(create_issue "$ASSIGNEE" "$ISSUE_LABELS" 2>/dev/null || true)
fi

# Tier 2: Retry without assignee (e.g. assignee is not a repo collaborator)
if [[ -z "$ISSUE_LINK" && -n "$ISSUE_LABELS" ]]; then
  echo "Attempting to create issue without assignee..."
  ISSUE_LINK=$(create_issue "" "$ISSUE_LABELS" 2>/dev/null || true)
fi

# Tier 3: Retry without labels (e.g. label does not exist in repository)
if [[ -z "$ISSUE_LINK" ]]; then
  echo "Attempting to create issue without label..."
  ISSUE_LINK=$(create_issue "" "" 2>/dev/null || true)
fi

if [[ -z "$ISSUE_LINK" ]]; then
  echo "Error: Failed to create issue in $GH_REPO." >&2
  exit 1
fi

ISSUE_LINK=$(echo "$ISSUE_LINK" | tr -d '[:space:]')
echo "Created issue: $ISSUE_LINK"

if [[ -n "${TEAM_MENTION:-}" ]]; then
  ISSUE_NUM="${ISSUE_LINK##*/}"
  echo "Adding team mention comment for $TEAM_MENTION..."
  gh issue comment "$ISSUE_NUM" --repo "$GH_REPO" --body "$TEAM_MENTION A critical issue has been created, please respond immediately."
fi
