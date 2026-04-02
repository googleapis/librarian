#!/bin/bash
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

# Fail on any error
set -e

# Display commands being run
set -x

# Fail if a dependency was added without the necessary go.mod/go.sum change
# being part of the commit.
go mod tidy
find . -name go.mod -execdir go mod tidy \;

# Documentation for the :^ pathspec can be found at:
# https://git-scm.com/docs/gitglossary#Documentation/gitglossary.txt-aiddefpathspecapathspec
git diff --exit-code '*go.mod' :^internal/generated/snippets
git diff --exit-code '*go.sum' :^internal/generated/snippets

# Formatting check
# Filter out spanner/key_recipe_cache_test.go as it is currently untidy upstream.
# We only check for the 'diff ' lines to identify which files need formatting.
golangci-lint fmt --diff --enable goimports,gofmt ./... 2>&1 | \
    grep "^diff " | \
    grep -vE "spanner/key_recipe_cache_test.go" > fmt_issues.txt || true

if [ -s fmt_issues.txt ]; then
  echo "Found untidy files:"
  cat fmt_issues.txt
  rm fmt_issues.txt
  exit 1
fi
rm fmt_issues.txt

# Linting check
# Add comprehensive filters for revive and staticcheck noise.
# We suppress issued lines to make filtering more reliable.
golangci-lint run --default none --enable-only revive,staticcheck --output.text.print-issued-lines=false ./... 2>&1 | (
  grep -vE "gen\.go" |
    grep -vE "receiver name [a-zA-Z]+[0-9]* should be consistent with previous receiver name" |
    grep -vE "exported const AllUsers|AllAuthenticatedUsers|RoleOwner|SSD|HDD|PRODUCTION|DEVELOPMENT should have comment" |
    grep -v "exported func Value returns unexported type pretty.val, which can be annoying to use" |
    grep -vE "exported func (Increment|FieldTransformIncrement|FieldTransformMinimum|FieldTransformMaximum) returns unexported type firestore.transform, which can be annoying to use" |
    grep -v "ExecuteStreamingSql" |
    grep -v "MethodExecuteSql should be MethodExecuteSQL" |
    grep -vE " executeStreamingSql(Min|Rnd)Time" |
    grep -vE " executeSql(Min|Rnd)Time" |
    grep -vE "pubsub\/pstest\/fake\.go.+should have comment or be unexported" |
    grep -vE "pubsub\/subscription\.go.+ type name will be used as pubsub.PubsubWrapper by other packages" |
    grep -v "ClusterId" |
    grep -v "InstanceId" |
    grep -v "firestore.arrayUnion" |
    grep -v "firestore.arrayRemove" |
    grep -v "maxAttempts" |
    grep -v "firestore.commitResponse" |
    grep -v "UptimeCheckIpIterator" |
    grep -vE "apiv[0-9]+" |
    grep -v "ALL_CAPS" |
    grep -v "go-cloud-debug-agent" |
    grep -v "mock_test" |
    grep -v "internal/testutil/funcmock.go" |
    grep -v "internal/backoff" |
    grep -v "internal/trace" |
    grep -v "internal/gapicgen/generator" |
    grep -v "internal/generated/snippets" |
    grep -v "a blank import should be only in a main or test package" |
    grep -v "method ExecuteSql should be ExecuteSQL" |
    grep -vE "spanner/spansql/(sql|types).go:.*should have comment" |
    grep -vE "\.pb\.go:" |
    grep -v "third_party/go/doc" |
    grep -v SA1019 |
    grep -v internal/btree/btree.go |
    grep -v httpreplay/internal/proxy/debug.go |
    grep -v third_party/pkgsite/synopsis.go |
    grep -vE "redefines-builtin-id" |
    grep -vE "package-comments" |
    grep -vE "unused-parameter" |
    grep -vE "empty-block" |
    grep -vE "QF10[0-9]+" |
    grep -vE "SA1024" |
    grep -vE "grpc.Dial" |
    grep -vE "grpc.WithInsecure" |
    grep -vE "grpc.WithBlock" |
    grep -vE "CredentialsFromJSON" |
    grep -vE "internal/pretty" |
    grep -vE "var-declaration" |
    grep -vE "unnecessary-else" |
    grep -vE "trace\.go" |
    grep -vE "rpcreplay" |
    grep -vE "^[0-9]+ issues?:?$" |
    grep -vE "^\* (revive|staticcheck): [0-9]+$"
) > lint_issues.txt || true

if [ -s lint_issues.txt ]; then
  echo "Found lint issues:"
  cat lint_issues.txt
  rm lint_issues.txt
  exit 1
fi
rm lint_issues.txt

echo "Done vetting!"
