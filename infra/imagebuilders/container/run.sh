#!/bin/bash
# Copyright 2025 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

echo "in release container"
cat /librarian/release-stage-request.json

ls -al /librarian
ls -al /repo
ls -al /output

new_version=$(jq -r '.libraries[0].version' /librarian/release-stage-request.json)
echo "release version: ${new_version}"
mkdir /output/internal/
mkdir /output/internal/cli/
echo "${new_version}" > /output/internal/cli/version.txt

ls -al /output

echo "writing empty response"
echo "{}" > /librarian/release-stage-response.json
