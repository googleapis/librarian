#!/bin/bash
# Copyright 2019 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

set -xe
# 5 libraries to check
LIBS=(
  "secretmanager"
  "maps"
  "shopping"
  "bigquery"
  "orgpolicy"
)

# fail_on_output reads from stdin and fails if there is any output.
fail_on_output() {
  if read -r line; then
    echo "Error: Found unformatted files or unexpected output:" >&2
    echo "$line" >&2
    cat >&2
    return 1
  fi
}

for lib in "${LIBS[@]}"; do
  echo "=== Vetting ${lib} ==="
  pushd "$lib" > /dev/null

  go mod tidy
  git diff --exit-code go.mod go.sum

  goimports -l . 2>&1 | grep -vE "\.pb\.go" | fail_on_output
  golangci-lint run ./...
  popd > /dev/null
done

echo "Done vetting!"
