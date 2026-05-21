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
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/googleapis/librarian/internal/command"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/filesystem"
	"github.com/googleapis/librarian/internal/pip"
	"github.com/googleapis/librarian/internal/yaml"
)

//go:embed librarian.yaml
var librarianYAML []byte

// Install installs Java tool dependencies.
func Install(ctx context.Context) error {
	for _, cmd := range []string{"java", "mvn", "pip"} {
		if _, err := exec.LookPath(cmd); err != nil {
			return fmt.Errorf("%s is not installed or not in PATH, which is required for Java tool installation: %w", cmd, err)
		}
	}
	cfg, err := yaml.Unmarshal[config.Config](librarianYAML)
	if err != nil {
		return fmt.Errorf("parsing embedded librarian.yaml: %w", err)
	}
	binDir, err := getInstallDir()
	if err != nil {
		return err
	}
	libDir := filepath.Join(filepath.Dir(binDir), "lib")
	if err := os.MkdirAll(binDir, 0755); err != nil {
		return fmt.Errorf("failed to create bin directory %q: %w", binDir, err)
	}
	if err := os.MkdirAll(libDir, 0755); err != nil {
		return fmt.Errorf("failed to create lib directory %q: %w", libDir, err)
	}
	for _, mvnTool := range cfg.Tools.Maven {
		if err := installExternalMavenTool(ctx, mvnTool, binDir, libDir); err != nil {
			return fmt.Errorf("failed to install external maven tool %s: %w", mvnTool.Name, err)
		}
	}
	if len(cfg.Tools.Pip) > 0 {
		if err := pip.Install(ctx, cfg.Tools.Pip); err != nil {
			return fmt.Errorf("failed to install pip tools: %w", err)
		}
	}
	return nil
}

// getInstallDir returns the absolute path of the installation directory for Java tools.
// It resolves LIBRARIAN_INSTALL_DIR if set, otherwise defaults to "$HOME/java_tools/bin".
// TODO(https://github.com/googleapis/librarian/issues/5850): Refactor this once Librarian-wide variable is ready.
func getInstallDir() (string, error) {
	dir := os.Getenv("LIBRARIAN_INSTALL_DIR")
	if dir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to get user home directory: %w", err)
		}
		dir = filepath.Join(home, "java_tools", "bin")
	}
	return filepath.Abs(dir)
}

// installExternalMavenTool downloads a Maven-based external tool, copies its compiled artifact
// (.jar or .exe) hermetically to the sibling lib folder, and creates an executable wrapper script
// in the bin folder pointing directly to that library file.
func installExternalMavenTool(ctx context.Context, mvnTool *config.MavenTool, binDir, libDir string) error {
	// 1. Construct artifact string for maven
	ext := strings.ToLower(mvnTool.Packaging)
	if ext == "" {
		ext = "jar"
	}
	artifact := fmt.Sprintf("%s:%s:%s:%s", mvnTool.GroupID, mvnTool.ArtifactID, mvnTool.Version, ext)
	if mvnTool.Classifier != "" {
		artifact = fmt.Sprintf("%s:%s", artifact, mvnTool.Classifier)
	}
	// 2. Download via mvn inside binDir (local .m2 populate)
	args := []string{
		"dependency:get",
		"-Dartifact=" + artifact,
	}
	if err := command.RunStreamingInDir(ctx, binDir, "mvn", args...); err != nil {
		return fmt.Errorf("failed to download artifact %s: %w", artifact, err)
	}
	// 3. Resolve path in .m2/repository
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get user home directory: %w", err)
	}
	m2Repo := filepath.Join(home, ".m2", "repository")
	groupIDPath := strings.ReplaceAll(mvnTool.GroupID, ".", "/")
	fileName := fmt.Sprintf("%s-%s", mvnTool.ArtifactID, mvnTool.Version)
	if mvnTool.Classifier != "" {
		fileName = fmt.Sprintf("%s-%s", fileName, mvnTool.Classifier)
	}
	fileName = fmt.Sprintf("%s.%s", fileName, ext)
	artifactPath := filepath.Join(m2Repo, groupIDPath, mvnTool.ArtifactID, mvnTool.Version, fileName)
	if _, err := os.Stat(artifactPath); err != nil {
		return fmt.Errorf("downloaded artifact not found at %s: %w", artifactPath, err)
	}
	// 4. Copy artifact file to hermetic libDir
	destPath := filepath.Join(libDir, fileName)
	if err := filesystem.CopyFile(artifactPath, destPath); err != nil {
		return fmt.Errorf("failed to copy artifact to lib folder: %w", err)
	}
	if ext == "exe" {
		if err := os.Chmod(destPath, 0755); err != nil {
			return fmt.Errorf("failed to make copied exe executable: %w", err)
		}
	}
	// 5. Create wrapper script in binDir pointing to libDir
	wrapperPath := filepath.Join(binDir, mvnTool.Name)
	var wrapperContent string
	if ext == "exe" {
		wrapperContent = fmt.Sprintf("#!/bin/sh\nexec %q \"$@\"\n", destPath)
	} else {
		wrapperContent = fmt.Sprintf("#!/bin/sh\nexec java -jar %q \"$@\"\n", destPath)
	}
	return os.WriteFile(wrapperPath, []byte(wrapperContent), 0755)
}
