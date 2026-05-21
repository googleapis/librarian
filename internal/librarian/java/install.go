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

package java

import (
	"context"
	_ "embed"
	"fmt"
	"os/exec"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/pip"
	"github.com/googleapis/librarian/internal/yaml"
)

//go:embed librarian.yaml
var librarianYAML []byte

// Install installs Java tool dependencies.
func Install(ctx context.Context) error {
	for _, cmd := range []string{"pip"} {
		if _, err := exec.LookPath(cmd); err != nil {
			return fmt.Errorf("%s is not installed or not in PATH, which is required for Java tool installation: %w", cmd, err)
		}
	}
	cfg, err := yaml.Unmarshal[config.Config](librarianYAML)
	if err != nil {
		return fmt.Errorf("parsing embedded librarian.yaml: %w", err)
	}
	if len(cfg.Tools.Pip) > 0 {
		if err := pip.Install(ctx, cfg.Tools.Pip); err != nil {
			return fmt.Errorf("failed to install pip tools: %w", err)
		}
	}
	return nil
}
