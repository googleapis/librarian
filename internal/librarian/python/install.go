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

package python

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

// Install installs Python pip tool dependencies.
func Install(ctx context.Context) error {
	cfg, err := yaml.Unmarshal[config.Config](librarianYAML)
	if err != nil {
		return fmt.Errorf("parsing embedded librarian.yaml: %w", err)
	}
	for _, tool := range cfg.Tools.Pip {
		pkg := tool.Package
		if pkg == "" {
			pkg = fmt.Sprintf("%s==%s", tool.Name, tool.Version)
		}
		if err := command.RunStreaming(ctx, "pip", "install", pkg); err != nil {
			return err
		}
	}
	return nil
}
