#!/bin/bash
# Sets up the test_env directory by building the required binaries and fetching
# the complete set of protobuf dependencies from the googleapis repository.

set -e

# Navigate to the project root
cd "$(dirname "$0")/.."

echo "Building surfer-dev binary..."
mkdir -p test_env/bin
go build -o ./test_env/bin/surfer-dev ./cmd/surfer/main.go

echo "Setting up googleapis dependencies..."
if [ ! -d "test_env/googleapis" ]; then
    echo "Cloning the googleapis repository (depth 1)..."
    git clone --depth 1 https://github.com/googleapis/googleapis.git test_env/googleapis
else
    echo "Updating existing googleapis repository..."
    (cd test_env/googleapis && git pull)
fi

echo "Ensuring test_env/google points to the full googleapis/google directory..."
# If test_env/google exists but is not a symlink (e.g., it's the partial directory checked into git), remove it.
if [ -d "test_env/google" ] && [ ! -L "test_env/google" ]; then
    echo "Removing existing partial test_env/google directory to replace with full symlink..."
    rm -rf test_env/google
fi

# Create a symlink to the complete google directory from the cloned repo
if [ ! -e "test_env/google" ]; then
    ln -s googleapis/google test_env/google
    echo "Symlink created."
else
    echo "Symlink already exists."
fi

echo ""
echo "✅ test_env is fully set up and ready to use!"

