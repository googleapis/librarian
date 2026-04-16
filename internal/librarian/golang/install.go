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
	_ "embed"
	"fmt"

	"github.com/googleapis/librarian/internal/command"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/yaml"
)

//go:embed librarian.yaml
var librarianYAML []byte

// Install installs the tools required for Go library generation.
func Install(ctx context.Context, tools *config.Tools) error {
	if tools == nil || len(tools.Go) == 0 {
		return installFallbackTools(ctx)
	}

	for _, tool := range tools.Go {
		version := tool.Version
		if version == "" {
			version = "latest"
		}
		toolStr := fmt.Sprintf("%s@%s", tool.Name, version)
		if err := command.Run(ctx, command.Go, "install", toolStr); err != nil {
			return fmt.Errorf("install %s: %w", toolStr, err)
		}
	}
	return nil
}

func installFallbackTools(ctx context.Context) error {
	cfg, err := yaml.Unmarshal[config.Config](librarianYAML)
	if err != nil {
		return fmt.Errorf("parsing embedded librarian.yaml: %w", err)
	}
	if cfg.Tools == nil || len(cfg.Tools.Go) == 0 {
		return fmt.Errorf("no go tools defined in embedded librarian.yaml")
	}
	for _, tool := range cfg.Tools.Go {
		version := tool.Version
		if version == "" {
			version = "latest"
		}
		toolStr := fmt.Sprintf("%s@%s", tool.Name, version)
		if err := command.Run(ctx, command.Go, "install", toolStr); err != nil {
			return fmt.Errorf("install %s: %w", toolStr, err)
		}
	}
	return nil
}
