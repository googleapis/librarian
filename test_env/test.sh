#!/bin/bash
# This script builds and runs the surfer gcloud generator.
# It is intended to be run from the root of the 'librarian' repository.

set -e

echo "Building surfer-dev binary..."
# Build the binary from the main cmd directory and place it in a local bin directory.
go build -o ./test_env/bin/surfer-dev ./cmd/surfer/main.go

(
  cd test_env
  
  echo "Splitting configs..."
  go run split_yaml.go ../internal/surfer/gcloud/testdata/parallelstore/gcloud.yaml parallelstore
  # go run split_yaml.go ../internal/surfer/gcloud/testdata/memorystore/gcloud.yaml memorystore
  # go run split_yaml.go ../internal/surfer/gcloud/testdata/parametermanager/gcloud.yaml parametermanager
  go run split_yaml.go ../internal/surfer/gcloud/testdata/iam/gcloud.yaml iam
  go run split_yaml.go ../internal/surfer/gcloud/testdata/developer_connect/gcloud.yaml developer_connect

  echo "Running the gcloud command generator for Parallelstore..."
  ./bin/surfer-dev generate parallelstore_ga.yaml --googleapis . --out . --proto-files-include-list google/cloud/parallelstore/v1/parallelstore.proto
  ./bin/surfer-dev generate parallelstore_beta.yaml --googleapis . --out . --proto-files-include-list google/cloud/parallelstore/v1beta/parallelstore.proto

  # echo "Running the gcloud command generator for Memorystore..."
  # ./bin/surfer-dev generate memorystore_ga.yaml --googleapis . --out . --proto-files-include-list google/cloud/memorystore/v1/memorystore.proto

  # echo "Running the gcloud command generator for Parameter Manager..."
  # ./bin/surfer-dev generate parametermanager_ga.yaml --googleapis . --out . --proto-files-include-list google/cloud/parametermanager/v1/service.proto

  echo "Running the gcloud command generator for IAM (Policy Bindings)..."
  ./bin/surfer-dev generate iam_ga.yaml --googleapis . --out . --proto-files-include-list google/iam/v3/policy_bindings_service.proto

  echo "Running the gcloud command generator for Developer Connect..."
  ./bin/surfer-dev generate developer_connect_ga.yaml --googleapis . --out . --proto-files-include-list google/cloud/developerconnect/v1/developer_connect.proto
)

echo "✅ Generation complete."
