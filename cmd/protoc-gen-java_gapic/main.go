// Copyright 2026 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package main provides the protoc-gen-java_gapic plugin binary for generating Java GAPIC clients.
package main

import (
	"fmt"
	"io"
	"os"

	"github.com/googleapis/librarian/internal/gapic/java"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/pluginpb"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "protoc-gen-java_gapic: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	reqBytes, err := io.ReadAll(os.Stdin)
	if err != nil {
		return fmt.Errorf("failed to read request from stdin: %w", err)
	}

	var req pluginpb.CodeGeneratorRequest
	if err := proto.Unmarshal(reqBytes, &req); err != nil {
		return fmt.Errorf("failed to unmarshal CodeGeneratorRequest: %w", err)
	}

	resp, err := java.GenerateGapic(&req)
	if err != nil {
		return fmt.Errorf("generation failed: %w", err)
	}

	respBytes, err := proto.Marshal(resp)
	if err != nil {
		return fmt.Errorf("failed to marshal CodeGeneratorResponse: %w", err)
	}

	if _, err := os.Stdout.Write(respBytes); err != nil {
		return fmt.Errorf("failed to write response to stdout: %w", err)
	}
	return nil
}
