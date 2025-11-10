#!/bin/bash
# Copyright 2025 Google LLC
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

# This script builds and runs the sidekick gcloud generator.
# It is intended to be run from within the 'test_env' directory after
# the environment has been set up by the 'setup_test_env.sh' script.

set -e

echo "Building sidekick-dev binary..."
# Build the binary from the main cmd directory and place it in a local bin directory.
go build -o ./bin/sidekick-dev ../cmd/sidekick/main.go

echo "Running the gcloud command generator..."
# Run the newly built binary.
./bin/sidekick-dev refresh --output .

echo "✅ Generation complete."
echo "Generated files are in the 'parallelstore' directory."
