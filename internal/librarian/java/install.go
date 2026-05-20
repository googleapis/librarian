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
	"encoding/xml"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/googleapis/librarian/internal/command"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/yaml"
)

//go:embed librarian.yaml
var librarianYAML []byte

type javaToolsConfig struct {
	Tools struct {
		Maven []*config.MavenTool `yaml:"maven"`
		Pip   []*config.PipTool   `yaml:"pip"`
	} `yaml:"tools"`
}

// Install installs Java tool dependencies.
func Install(ctx context.Context) error {
	// Task 1: Prerequisite check: must run in google-cloud-java root
	if _, err := os.Stat("sdk-platform-java/gapic-generator-java/pom.xml"); err != nil {
		return fmt.Errorf("librarian install java must be run from the root of a google-cloud-java repository clone: %w", err)
	}

	// Check other required tools in PATH
	for _, cmd := range []string{"java", "mvn", "pip"} {
		if _, err := exec.LookPath(cmd); err != nil {
			return fmt.Errorf("%s is not installed or not in PATH, which is required for Java tool installation: %w", cmd, err)
		}
	}

	cfg, err := yaml.Unmarshal[javaToolsConfig](librarianYAML)
	if err != nil {
		return fmt.Errorf("parsing embedded librarian.yaml: %w", err)
	}

	installDir, err := getInstallDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(installDir, 0755); err != nil {
		return fmt.Errorf("failed to create install directory %q: %w", installDir, err)
	}

	// Install Maven tools
	for _, t := range cfg.Tools.Maven {
		if t.LocalPath != "" {
			// Task 2: Local Maven artifact (gapic-generator-java)
			if err := installLocalMavenTool(ctx, t, installDir); err != nil {
				return fmt.Errorf("failed to install local maven tool %s: %w", t.Name, err)
			}
		} else {
			// Task 3: External Maven artifact (google-java-format, protoc-gen-grpc-java)
			if err := installExternalMavenTool(ctx, t, installDir); err != nil {
				return fmt.Errorf("failed to install external maven tool %s: %w", t.Name, err)
			}
		}
	}

	// Task 4: Install Pip tools
	if len(cfg.Tools.Pip) > 0 {
		if err := config.InstallPipTools(ctx, cfg.Tools.Pip); err != nil {
			return fmt.Errorf("failed to install pip tools: %w", err)
		}
	}

	fmt.Printf("--------------------------------------------------\n")
	fmt.Printf("All Java tools installed in %s\n", installDir)
	fmt.Printf("Please ensure this directory is in your PATH.\n")
	fmt.Printf("--------------------------------------------------\n")

	return nil
}

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

type pomProject struct {
	Version string `xml:"version"`
}

func getLocalToolVersion(pomPath string) (string, error) {
	data, err := os.ReadFile(pomPath)
	if err != nil {
		return "", err
	}
	var proj pomProject
	if err := xml.Unmarshal(data, &proj); err != nil {
		return "", err
	}
	if proj.Version == "" {
		return "", fmt.Errorf("version not found in %s", pomPath)
	}
	return proj.Version, nil
}

func installLocalMavenTool(ctx context.Context, mavenTool *config.MavenTool, installDir string) error {
	// 1. Build the tool
	args := []string{
		"package", "-B", "-ntp", "-T", "1.5C",
		"-DskipTests", "-Dcheckstyle.skip", "-Dclirr.skip", "-Denforcer.skip", "-Dfmt.skip",
		"-pl", mavenTool.LocalPath, "--also-make",
	}
	fmt.Printf("Building local tool %s...\n", mavenTool.Name)
	if err := command.RunStreaming(ctx, "mvn", args...); err != nil {
		return fmt.Errorf("failed to build local tool: %w", err)
	}
	// 2. Get version
	pomPath := filepath.Join(mavenTool.LocalPath, "pom.xml")
	version, err := getLocalToolVersion(pomPath)
	if err != nil {
		return fmt.Errorf("failed to get local tool version: %w", err)
	}
	// 3. Resolve jar path
	// Replace the "{{Version}}" placeholder in LocalArtifact with the actual version
	// parsed from the local pom.xml.
	jarPath := strings.ReplaceAll(mavenTool.LocalArtifact, "{{Version}}", version)
	if _, err := os.Stat(jarPath); err != nil {
		return fmt.Errorf("built JAR not found at %s: %w", jarPath, err)
	}
	absJarPath, err := filepath.Abs(jarPath)
	if err != nil {
		return fmt.Errorf("failed to resolve absolute path for %s: %w", jarPath, err)
	}
	// 4. Create wrapper
	wrapperPath := filepath.Join(installDir, mavenTool.Name)
	var wrapperContent string
	if mavenTool.MainClass != "" {
		wrapperContent = fmt.Sprintf("#!/bin/bash\nexec java -cp %q %s \"$@\"\n", absJarPath, mavenTool.MainClass)
	} else {
		wrapperContent = fmt.Sprintf("#!/bin/bash\nexec java -jar %q \"$@\"\n", absJarPath)
	}
	if err := os.WriteFile(wrapperPath, []byte(wrapperContent), 0755); err != nil {
		return fmt.Errorf("failed to write wrapper script: %w", err)
	}
	return nil
}

func installExternalMavenTool(ctx context.Context, t *config.MavenTool, installDir string) error {
	// 1. Construct artifact string for maven
	ext := strings.ToLower(t.Packaging)
	if ext == "" {
		ext = "jar"
	}
	artifact := fmt.Sprintf("%s:%s:%s:%s", t.GroupID, t.ArtifactID, t.Version, ext)
	if t.Classifier != "" {
		artifact = fmt.Sprintf("%s:%s", artifact, t.Classifier)
	}

	// 2. Download via mvn
	args := []string{
		"dependency:get",
		"-Dartifact=" + artifact,
	}
	fmt.Printf("Downloading external tool %s...\n", t.Name)
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}
	if err := os.Chdir(installDir); err != nil {
		return fmt.Errorf("failed to change directory to %s: %w", installDir, err)
	}
	defer os.Chdir(cwd)
	if err := command.RunStreaming(ctx, "mvn", args...); err != nil {
		return fmt.Errorf("failed to download artifact %s: %w", artifact, err)
	}

	// 3. Resolve path in .m2/repository
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get user home directory: %w", err)
	}
	m2Repo := filepath.Join(home, ".m2", "repository")
	groupIDPath := strings.ReplaceAll(t.GroupID, ".", "/")

	fileName := fmt.Sprintf("%s-%s", t.ArtifactID, t.Version)
	if t.Classifier != "" {
		fileName = fmt.Sprintf("%s-%s", fileName, t.Classifier)
	}
	fileName = fmt.Sprintf("%s.%s", fileName, ext)

	artifactPath := filepath.Join(m2Repo, groupIDPath, t.ArtifactID, t.Version, fileName)
	if _, err := os.Stat(artifactPath); err != nil {
		return fmt.Errorf("downloaded artifact not found at %s: %w", artifactPath, err)
	}

	// 4. Create wrapper
	wrapperPath := filepath.Join(installDir, t.Name)
	var wrapperContent string
	if strings.ToLower(t.Packaging) == "exe" {
		// Make the downloaded binary executable
		if err := os.Chmod(artifactPath, 0755); err != nil {
			return fmt.Errorf("failed to make %s executable: %w", artifactPath, err)
		}
		wrapperContent = fmt.Sprintf("#!/bin/bash\nexec %q \"$@\"\n", artifactPath)
	} else {
		wrapperContent = fmt.Sprintf("#!/bin/bash\nexec java -jar %q \"$@\"\n", artifactPath)
	}

	if err := os.WriteFile(wrapperPath, []byte(wrapperContent), 0755); err != nil {
		return fmt.Errorf("failed to write wrapper script: %w", err)
	}

	return nil
}
