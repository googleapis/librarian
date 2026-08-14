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
	"crypto/sha256"
	"encoding/hex"
	"encoding/xml"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
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

// markerPath returns the path of the hidden file recording a completed install
// of the named tool, alongside its wrapper in binDir.
func markerPath(binDir, name string) string {
	return filepath.Join(binDir, "."+name+".fingerprint")
}

// upToDate reports whether the named tool is already installed with the given
// fingerprint: its bin wrapper exists and the recorded fingerprint matches.
// This lets a repeated install skip the expensive mvn build/download.
func upToDate(binDir, name, fingerprint string) bool {
	if fingerprint == "" {
		return false
	}
	if _, err := os.Stat(filepath.Join(binDir, name)); err != nil {
		return false
	}
	recorded, err := os.ReadFile(markerPath(binDir, name))
	return err == nil && string(recorded) == fingerprint
}

// writeMarker records fingerprint so a later install can skip redoing the work.
func writeMarker(binDir, name, fingerprint string) error {
	if fingerprint == "" {
		return nil
	}
	return os.WriteFile(markerPath(binDir, name), []byte(fingerprint), 0o644)
}

// externalFingerprint identifies a prebuilt tool by its immutable Maven
// coordinate, so a re-install of the same version is skipped.
func externalFingerprint(mvnTool *config.MavenTool) string {
	artifact, _ := getM2ArtifactSpec(mvnTool)
	return "external:" + artifact
}

// localFingerprint hashes the source tree of a locally-built tool (path, size,
// and modtime of every file), excluding build output and VCS directories, so an
// unchanged source tree is not rebuilt.
func localFingerprint(localPath string) (string, error) {
	h := sha256.New()
	err := filepath.WalkDir(localPath, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if name := d.Name(); name == "target" || name == ".git" {
				return fs.SkipDir
			}
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		fmt.Fprintf(h, "%s\x00%d\x00%d\n", p, info.Size(), info.ModTime().UnixNano())
		return nil
	})
	if err != nil {
		return "", fmt.Errorf("failed to fingerprint %q: %w", localPath, err)
	}
	return "local:" + hex.EncodeToString(h.Sum(nil)), nil
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
	fingerprint := externalFingerprint(mvnTool)
	if upToDate(binDir, mvnTool.Name, fingerprint) {
		slog.Debug("maven tool already installed, skipping download", "tool", mvnTool.Name)
		return nil
	}
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
	if err := createBinWrapper(mvnTool.Name, destPath, binDir, isExe, mvnTool.MainClass); err != nil {
		return err
	}
	return writeMarker(binDir, mvnTool.Name, fingerprint)
}

// installLocalMavenTool compiles a local Maven project, parses its pom.xml metadata coordinates,
// copies the built target artifact (.jar or .exe) to the sibling lib folder, and creates an executable
// wrapper script in the bin folder.
func installLocalMavenTool(ctx context.Context, mvnTool *config.MavenTool, binDir, libDir string) error {
	fingerprint, err := localFingerprint(mvnTool.LocalPath)
	if err != nil {
		return err
	}
	if upToDate(binDir, mvnTool.Name, fingerprint) {
		slog.Debug("maven tool source unchanged, skipping build", "tool", mvnTool.Name)
		return nil
	}
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
	if err := createBinWrapper(mvnTool.Name, destPath, binDir, isExe, mvnTool.MainClass); err != nil {
		return err
	}
	return writeMarker(binDir, mvnTool.Name, fingerprint)
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

// mavenOffline reports whether mvn should run in offline mode (-o). Enabled by
// setting LIBRARIAN_MAVEN_OFFLINE to a truthy value once ~/.m2 is warm, which
// skips all remote metadata lookups. Default is online (current behavior).
func mavenOffline() bool {
	switch strings.ToLower(os.Getenv("LIBRARIAN_MAVEN_OFFLINE")) {
	case "1", "true", "yes":
		return true
	}
	return false
}

// downloadArgs builds the mvn dependency:get arguments. -B (batch) and -ntp
// (no transfer progress) keep output non-interactive and quiet, matching the
// package build.
func downloadArgs(artifact string, offline bool) []string {
	args := []string{"dependency:get", "-B", "-ntp"}
	if offline {
		args = append(args, "-o")
	}
	return append(args, "-Dartifact="+artifact)
}

// downloadM2Artifact executes mvn dependency:get to download the target artifact.
func downloadM2Artifact(ctx context.Context, artifact, workDir string) error {
	args := downloadArgs(artifact, mavenOffline())
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
		content = fmt.Sprintf("#!/bin/sh\nexec java -cp %q %q \"$@\"\n", destPath, mainClass)
	default:
		content = fmt.Sprintf("#!/bin/sh\nexec java -jar %q \"$@\"\n", destPath)
	}
	return os.WriteFile(wrapperPath, []byte(content), 0o755)
}

// packageArgs builds the mvn package arguments for a local project build:
// batch mode, no transfer progress, parallel, and skipping tests and
// static-analysis plugins the generator build does not need.
func packageArgs(localPath string, offline bool) []string {
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
	}
	if offline {
		args = append(args, "-o")
	}
	return append(args, "-pl", localPath, "--also-make")
}

// buildLocalMavenProject builds the local Maven project at the target relative path under the monorepo root.
func buildLocalMavenProject(ctx context.Context, localPath string) error {
	args := packageArgs(localPath, mavenOffline())
	if err := command.RunStreaming(ctx, "mvn", args...); err != nil {
		return fmt.Errorf("failed to build local Maven project %q: %w", localPath, err)
	}
	return nil
}
