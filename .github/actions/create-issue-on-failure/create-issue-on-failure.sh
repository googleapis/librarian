#!/usr/bin/env bash

set -e

# Search for an existing open issue with the same title
EXISTING_ISSUE=$(gh issue list --repo "$GITHUB_REPO" --search "$ISSUE_TITLE in:title" --state open --json number --jq '.[0].number')

if [[ -n "$EXISTING_ISSUE" && "$EXISTING_ISSUE" != "null" ]]; then
  echo "Found existing open issue #$EXISTING_ISSUE. Adding comment."
  gh issue comment "$EXISTING_ISSUE" --repo "$GITHUB_REPO" --body "The build failed again. Workflow Run: $GITHUB_SERVER_URL/$GITHUB_REPO/actions/runs/$GITHUB_RUN_ID"
else
  echo "Creating new issue."
  ISSUE_LINK=$(gh issue create \
    --title "$ISSUE_TITLE" \
    --body "$ISSUE_BODY" \
    --label "$ISSUE_LABELS" \
    --assignee "$ISSUE_ASSIGNEE" \
    --repo "$GITHUB_REPO")

  if [[ -n "$ISSUE_MENTION" ]]; then
    ISSUE_NUM=${ISSUE_LINK##*/}
    gh issue comment "$ISSUE_NUM" --repo "$GITHUB_REPO" --body "$ISSUE_MENTION The workflow failed. Workflow Run: $GITHUB_SERVER_URL/$GITHUB_REPO/actions/runs/$GITHUB_RUN_ID"
  fi
fi
