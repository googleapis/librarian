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

package nodejs

import (
	"context"
	_ "embed"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/googleapis/librarian/internal/command"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/fetch"
	"github.com/googleapis/librarian/internal/yaml"
)

// gapicGeneratorSubdir is the sub-directory within the
// google-cloud-node repo that contains the gapic-generator-typescript
// source.
const gapicGeneratorSubdir = "core/generator/gapic-generator-typescript"

//go:embed librarian.yaml
var librarianYAML []byte

// Install installs Node.js tool dependencies.
func Install(ctx context.Context) error {
	cfg, err := yaml.Unmarshal[config.Config](librarianYAML)
	if err != nil {
		return fmt.Errorf("parsing embedded librarian.yaml: %w", err)
	}
	for _, tool := range cfg.Tools.NPM {
		if len(tool.Build) > 0 {
			if err := installNPMToolFromSource(ctx, tool); err != nil {
				return err
			}
			continue
		}
		pkg := tool.Package
		if pkg == "" {
			pkg = fmt.Sprintf("%s@%s", tool.Name, tool.Version)
		}
		if err := command.RunStreaming(ctx, "npm", "install", "-g", pkg); err != nil {
			return err
		}
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

func installNPMToolFromSource(ctx context.Context, tool *config.NPMTool) error {
	if tool.Package == "" {
		return fmt.Errorf("npm tool %s has build steps but no package URL", tool.Name)
	}
	repo, err := repoFromPackageURL(tool.Package)
	if err != nil {
		return err
	}
	dir, err := fetch.Repo(ctx, repo, tool.Version, tool.Checksum)
	if err != nil {
		return fmt.Errorf("fetching %s: %w", tool.Name, err)
	}

	// Run build steps.
	genDir := filepath.Join(dir, gapicGeneratorSubdir)
	for _, cmd := range tool.Build {
		if err := command.RunInDir(ctx, genDir, "sh", "-c", cmd); err != nil {
			return err
		}
	}
	return nil
}

// repoFromPackageURL extracts the repository path (e.g.,
// "github.com/googleapis/google-cloud-node") from a GitHub archive URL
// like "https://github.com/googleapis/google-cloud-node/archive/<sha>.tar.gz".
func repoFromPackageURL(packageURL string) (string, error) {
	parts := strings.SplitN(packageURL, "/archive/", 2)
	if len(parts) != 2 {
		return "", fmt.Errorf("cannot extract repo from package URL %q", packageURL)
	}
	return strings.TrimPrefix(parts[0], "https://"), nil
}
