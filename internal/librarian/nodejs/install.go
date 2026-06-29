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
	"bytes"
	"context"
	_ "embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

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
	for _, cmd := range []string{"node", "pnpm"} {
		if _, err := exec.LookPath(cmd); err != nil {
			return fmt.Errorf("%s is not installed or not in PATH, which is required for Node.js tool installation: %w", cmd, err)
		}
	}

	cfg, err := yaml.Unmarshal[config.Config](librarianYAML)
	if err != nil {
		return fmt.Errorf("parsing embedded librarian.yaml: %w", err)
	}
	env, err := getPNPMEnv(ctx)
	if err != nil {
		return err
	}

	for _, tool := range cfg.Tools.PNPM {
		if len(tool.Build) > 0 {
			if err := installPNPMToolFromSource(ctx, env, tool); err != nil {
				return err
			}
			continue
		}

		pkg := tool.Package
		if pkg == "" {
			pkg = fmt.Sprintf("%s@%s", tool.Name, tool.Version)
		}
		if err := runPNPM(ctx, "", env, "add", "-g", pkg); err != nil {
			return err
		}
	}
	return nil
}

// getPNPMEnv resolves Node's global installation bin prefix path dynamically
// and constructs a transient environment variable block to configure pnpm.
//
// This redirects all globally-installed pnpm binaries, virtual stores, and
// content-addressable storage caches to be nested under the Node prefix folder.
// This enables complete environment caching and restore on CI runners,
// while permanently avoiding persistent side-effects on the host machine
// (it does not modify the user's personal ~/.config/pnpm/rc files).
func getPNPMEnv(ctx context.Context) ([]string, error) {
	binOut, err := commandOutput(ctx, "node", "-e", "console.log(require('path').dirname(process.execPath))")
	if err != nil {
		return nil, fmt.Errorf("failed to resolve node bin directory: %w", err)
	}
	globalBin := strings.TrimSpace(binOut)

	// In pnpm v11+, globally installed binaries are stored in PNPM_HOME/bin.
	// We want them to be stored directly in globalBin (node's bin directory).
	// See https://pnpm.io/blog/releases/11.0#isolated-global-virtual-store-global-installs
	pnpmHome := filepath.Dir(globalBin)

	env := os.Environ()
	env = append(env, "PNPM_HOME="+pnpmHome)
	env = append(env, "PNPM_CONFIG_GLOBAL_BIN_DIR="+globalBin)
	env = append(env, "PNPM_CONFIG_GLOBAL_DIR="+filepath.Join(globalBin, "pnpm-global"))
	env = append(env, "PNPM_CONFIG_STORE_DIR="+filepath.Join(globalBin, "pnpm-store"))
	env = append(env, "PNPM_CONFIG_DANGEROUSLY_ALLOW_ALL_BUILDS=true")
	return env, nil
}

func runPNPM(ctx context.Context, dir string, env []string, args ...string) error {
	cmd := exec.CommandContext(ctx, "pnpm", args...)
	cmd.Dir = dir
	cmd.Env = env
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func runPNPMBuildCmd(ctx context.Context, dir string, env []string, cmdStr string) error {
	cmd := exec.CommandContext(ctx, "sh", "-c", cmdStr)
	cmd.Dir = dir
	cmd.Env = env
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func commandOutput(ctx context.Context, name string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	return out.String(), err
}

func installPNPMToolFromSource(ctx context.Context, env []string, tool *config.PNPMTool) error {
	if tool.Package == "" {
		return fmt.Errorf("pnpm tool %s has build steps but no package URL", tool.Name)
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
		if err := runPNPMBuildCmd(ctx, genDir, env, cmd); err != nil {
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
