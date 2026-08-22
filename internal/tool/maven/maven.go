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

// Package maven provides utilities for installing Maven tool dependencies.
package maven

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/googleapis/librarian/internal/command"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/filesystem"
)

var (
	errReadPOM    = errors.New("failed to read pom.xml")
	errParsePOM   = errors.New("failed to parse pom.xml")
	errInvalidPOM = errors.New("invalid pom.xml metadata")
)

// pomProject represents the target Maven metadata structured from pom.xml.
type pomProject struct {
	XMLName    xml.Name `xml:"project"`
	ArtifactID string   `xml:"artifactId"`
	Version    string   `xml:"version"`
	Parent     struct {
		Version string `xml:"version"`
	} `xml:"parent"`
}

// Install installs Maven tool dependencies.
func Install(ctx context.Context, tools []*config.MavenTool, binDir, libDir string) error {
	for _, mvnTool := range tools {
		var err error
		if mvnTool.LocalPath != "" {
			err = installLocalMavenTool(ctx, mvnTool, binDir, libDir)
		} else {
			err = installExternalMavenTool(ctx, mvnTool, binDir, libDir)
		}
		if err != nil {
			return fmt.Errorf("failed to install maven tool %s: %w", mvnTool.Name, err)
		}
	}
	return nil
}

// installExternalMavenTool downloads a Maven-based external tool, copies its compiled artifact
// (.jar or .exe) to the sibling lib folder, and creates an executable wrapper script
// in the bin folder pointing directly to that library file.
func installExternalMavenTool(ctx context.Context, mvnTool *config.MavenTool, binDir, libDir string) error {
	artifact, ext := getM2ArtifactSpec(mvnTool)
	if err := downloadM2Artifact(ctx, artifact, binDir); err != nil {
		return err
	}
	artifactPath, err := resolveM2ArtifactPath(mvnTool, ext)
	if err != nil {
		return err
	}
	if _, err := os.Stat(artifactPath); err != nil {
		return fmt.Errorf("downloaded artifact not found at %s: %w", artifactPath, err)
	}
	isExe := ext == "exe"
	destPath, err := copyArtifactToLib(artifactPath, libDir, isExe)
	if err != nil {
		return err
	}
	return createBinWrapper(mvnTool.Name, destPath, binDir, isExe, mvnTool.MainClass)
}

// installLocalMavenTool compiles a local Maven project, parses its pom.xml metadata coordinates,
// copies the built target artifact (.jar or .exe) to the sibling lib folder, and creates an executable
// wrapper script in the bin folder.
func installLocalMavenTool(ctx context.Context, mvnTool *config.MavenTool, binDir, libDir string) error {
	absLocalPath, err := filepath.Abs(mvnTool.LocalPath)
	if err != nil {
		return fmt.Errorf("failed to resolve absolute local path for %s: %w", mvnTool.LocalPath, err)
	}
	if err := buildLocalMavenProject(ctx, mvnTool.LocalPath); err != nil {
		return err
	}
	pomPath := filepath.Join(absLocalPath, "pom.xml")
	proj, err := parsePOM(pomPath)
	if err != nil {
		return err
	}
	ext := mvnTool.Packaging
	if ext == "" {
		ext = "jar"
	}
	fileName := fmt.Sprintf("%s-%s.%s", proj.ArtifactID, proj.Version, ext)
	artifactPath := filepath.Join(absLocalPath, "target", fileName)
	if _, err := os.Stat(artifactPath); err != nil {
		return fmt.Errorf("compiled artifact not found at %q: %w", artifactPath, err)
	}
	isExe := ext == "exe"
	destPath, err := copyArtifactToLib(artifactPath, libDir, isExe)
	if err != nil {
		return err
	}
	return createBinWrapper(mvnTool.Name, destPath, binDir, isExe, mvnTool.MainClass)
}

// parsePOM extracts the Maven metadata from the specified pom.xml path.
func parsePOM(pomPath string) (*pomProject, error) {
	data, err := os.ReadFile(pomPath)
	if err != nil {
		return nil, fmt.Errorf("%w %q: %w", errReadPOM, pomPath, err)
	}
	var proj pomProject
	if err := xml.Unmarshal(data, &proj); err != nil {
		return nil, fmt.Errorf("%w %q: %w", errParsePOM, pomPath, err)
	}
	if proj.Version == "" {
		proj.Version = proj.Parent.Version
	}
	if proj.ArtifactID == "" || proj.Version == "" {
		return nil, fmt.Errorf("%w %q: missing artifactId or version", errInvalidPOM, pomPath)
	}
	return &proj, nil
}

// getM2ArtifactSpec constructs the Maven coordinate string and returns it along with the file extension.
func getM2ArtifactSpec(mvnTool *config.MavenTool) (string, string) {
	ext := mvnTool.Packaging
	if ext == "" {
		ext = "jar"
	}
	artifact := fmt.Sprintf("%s:%s:%s:%s", mvnTool.GroupID, mvnTool.ArtifactID, mvnTool.Version, ext)
	if mvnTool.Classifier != "" {
		artifact = fmt.Sprintf("%s:%s", artifact, mvnTool.Classifier)
	}
	return artifact, ext
}

// downloadM2Artifact executes mvn dependency:get to download the target artifact.
func downloadM2Artifact(ctx context.Context, artifact, workDir string) error {
	args := []string{
		"dependency:get",
		"-Dartifact=" + artifact,
	}
	if err := command.RunStreamingInDir(ctx, workDir, "mvn", args...); err != nil {
		return fmt.Errorf("failed to download artifact %s: %w", artifact, err)
	}
	return nil
}

// resolveM2ArtifactPath returns the absolute path to the downloaded artifact in the local .m2 repository.
func resolveM2ArtifactPath(mvnTool *config.MavenTool, ext string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user home directory: %w", err)
	}
	m2Repo := filepath.Join(home, ".m2", "repository")
	groupIDPath := strings.ReplaceAll(mvnTool.GroupID, ".", "/")
	fileName := fmt.Sprintf("%s-%s", mvnTool.ArtifactID, mvnTool.Version)
	if mvnTool.Classifier != "" {
		fileName = fmt.Sprintf("%s-%s", fileName, mvnTool.Classifier)
	}
	fileName = fmt.Sprintf("%s.%s", fileName, ext)
	return filepath.Join(m2Repo, groupIDPath, mvnTool.ArtifactID, mvnTool.Version, fileName), nil
}

// copyArtifactToLib copies the artifact file into the isolated sibling lib directory,
// applying execution permission bits if needed.
func copyArtifactToLib(srcPath, libDir string, makeExecutable bool) (string, error) {
	fileName := filepath.Base(srcPath)
	destPath := filepath.Join(libDir, fileName)
	if err := filesystem.CopyFile(srcPath, destPath); err != nil {
		return "", fmt.Errorf("failed to copy artifact to lib folder: %w", err)
	}
	if makeExecutable {
		if err := os.Chmod(destPath, 0o755); err != nil {
			return "", fmt.Errorf("failed to make copied exe executable: %w", err)
		}
	}
	return destPath, nil
}

// createBinWrapper creates a shell wrapper script in the bin directory that forwards executions to the library file.
func createBinWrapper(wrapperName, destPath, binDir string, isExecutable bool, mainClass string) error {
	wrapperPath := filepath.Join(binDir, wrapperName)
	var content string
	switch {
	case isExecutable:
		content = fmt.Sprintf("#!/bin/sh\nexec %q \"$@\"\n", destPath)
	case mainClass != "":
		content = fmt.Sprintf("#!/bin/sh\nexec java --add-exports jdk.compiler/com.sun.tools.javac.api=ALL-UNNAMED --add-exports jdk.compiler/com.sun.tools.javac.file=ALL-UNNAMED --add-exports jdk.compiler/com.sun.tools.javac.parser=ALL-UNNAMED --add-exports jdk.compiler/com.sun.tools.javac.tree=ALL-UNNAMED --add-exports jdk.compiler/com.sun.tools.javac.util=ALL-UNNAMED -cp %q %q \"$@\"\n", destPath, mainClass)
	default:
		content = fmt.Sprintf("#!/bin/sh\nexec java --add-exports jdk.compiler/com.sun.tools.javac.api=ALL-UNNAMED --add-exports jdk.compiler/com.sun.tools.javac.file=ALL-UNNAMED --add-exports jdk.compiler/com.sun.tools.javac.parser=ALL-UNNAMED --add-exports jdk.compiler/com.sun.tools.javac.tree=ALL-UNNAMED --add-exports jdk.compiler/com.sun.tools.javac.util=ALL-UNNAMED -jar %q \"$@\"\n", destPath)
	}
	return os.WriteFile(wrapperPath, []byte(content), 0o755)
}

// buildLocalMavenProject builds the local Maven project at the target relative path under the monorepo root.
func buildLocalMavenProject(ctx context.Context, localPath string) error {
	args := []string{
		"package",
		"-B",
		"-ntp",
		"-T", "1.5C",
		"-DskipTests",
		"-Dcheckstyle.skip",
		"-Dclirr.skip",
		"-Denforcer.skip",
		"-Dfmt.skip",
		"-pl", localPath,
		"--also-make",
	}
	if err := command.RunStreaming(ctx, "mvn", args...); err != nil {
		return fmt.Errorf("failed to build local Maven project %q: %w", localPath, err)
	}
	return nil
}
