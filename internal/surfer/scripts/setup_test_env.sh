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

# This script sets up a self-contained test environment for the gcloud
# command generator. It should be run from the root of the 'librarian' repository.

set -e

echo "Creating test environment in ./test_env..."

# 1. Create the directory structure in the current directory.
if [ ! -d "test_env" ]; then
  mkdir -p test_env
else
  echo "test_env directory already exists"
  exit 1
fi

# 2. Clone GoogleAPIs if it doesn't already exist.
echo "Setting up Google APIs protos..."
git clone --depth 1 https://github.com/googleapis/googleapis.git ./test_env/googleapis
mv ./test_env/googleapis/google ./test_env/
rm -rf ./test_env/googleapis

# 3. Copy the gcloud.yaml configuration.
echo "Copying gcloud.yaml..."
cp ./internal/surfer/gcloud/testdata/parallelstore/gcloud.yaml ./test_env/



# 5. Create the test.sh script inside the test_env directory.
echo "Creating test.sh script..."
cat > ./test_env/test.sh << 'EOL'
#!/bin/bash
# This script builds and runs the surfer gcloud generator.
# It is intended to be run from within the 'test_env' directory.

set -e

echo "Building surfer-dev binary..."
# Build the binary from the main cmd directory and place it in a local bin directory.
go build -o ./test_env/bin/surfer-dev ./cmd/surfer/main.go

echo "Running the gcloud command generator..."
# Run the newly built binary.
(cd test_env && ./bin/surfer-dev generate ./test_env/gcloud.yaml --googleapis . --out .)

echo "✅ Generation complete."
echo "Generated files are in the 'parallelstore' directory."
EOL

# Make the test script executable.
chmod +x ./test_env/test.sh

echo "✅ Test environment setup complete."
echo "To run the generator, execute the following command:"
echo "./test_env/test.sh"
