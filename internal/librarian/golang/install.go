// Copyright 2025 Google LLC
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

package golang

import (
	"context"
	"fmt"

	"github.com/googleapis/librarian/internal/command"
	"github.com/googleapis/librarian/internal/config"
)

var fallbackTools = []string{
	"github.com/googleapis/gapic-generator-go/cmd/protoc-gen-go_gapic@v0.58.0",
	"golang.org/x/tools/cmd/goimports@latest",
	"google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.3.0",
	"google.golang.org/protobuf/cmd/protoc-gen-go@v1.36.11",
}

// Install installs the tools required for Go library generation.
func Install(ctx context.Context, tools *config.Tools) error {
	if tools == nil || len(tools.Go) == 0 {
		for _, tool := range fallbackTools {
			if err := command.Run(ctx, command.Go, "install", tool); err != nil {
				return fmt.Errorf("install %s: %w", tool, err)
			}
		}
		return nil
	}

	for _, tool := range tools.Go {
		version := tool.Version
		if version == "" {
			version = "latest"
		}
		t := fmt.Sprintf("%s@%s", tool.Name, version)
		if err := command.Run(ctx, command.Go, "install", t); err != nil {
			return fmt.Errorf("install %s: %w", t, err)
		}
	}
	return nil
}
