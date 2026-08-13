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

// Package protoc provides utilities for installing the protoc tool.
package protoc

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/googleapis/librarian/internal/cache"
	"github.com/googleapis/librarian/internal/command"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/fetch"
	"github.com/googleapis/librarian/internal/filesystem"
)

const (
	githubURLBase = "https://github.com"
	osWindows     = "windows"
	protocDir     = "protoc"
)

var (
	osMap = map[string]string{
		"darwin": "osx",
		"linux":  "linux",
	}
	archMap = map[string]string{
		"arm64": "aarch_64",
		"amd64": "x86_64",
	}
)

// Install installs the protoc tool.
func Install(ctx context.Context, protoc *config.Protoc) error {
	binaryPath, err := BinaryPath(protoc.Version)
	if err != nil {
		return err
	}
	if _, err := os.Stat(binaryPath); err == nil {
		return nil
	}
	url := downloadURL(protoc.Version, runtime.GOOS, runtime.GOARCH)
	dir, err := InstallDir(protoc.Version)
	if err != nil {
		return err
	}
	return downloadAndExtract(ctx, url, dir, protoc.SHA256)
}

// BinaryPath returns the absolute path to the protoc binary for the given version.
func BinaryPath(version string) (string, error) {
	if version == "" {
		return "", errors.New("protoc version cannot be empty")
	}
	dir, err := InstallDir(version)
	if err != nil {
		return "", err
	}
	protocPath := filepath.Join(dir, "bin", protocDir)
	if runtime.GOOS == osWindows {
		protocPath += ".exe"
	}
	return protocPath, nil
}

// BinaryPathOrSystem returns the path to the configured protoc binary if pc is non-nil
// and pc.Version is not empty, or falls back to looking up "protoc" in PATH.
func BinaryPathOrSystem(pc *config.Protoc) (string, error) {
	if pc != nil && pc.Version != "" {
		return BinaryPath(pc.Version)
	}
	return exec.LookPath("protoc")
}

// Run executes protoc with the given version and arguments.
func Run(ctx context.Context, env map[string]string, protoc *config.Protoc, args ...string) error {
	protocPath, err := BinaryPath(protoc.Version)
	if err != nil {
		return err
	}
	return command.RunWithEnv(ctx, env, protocPath, args...)
}

// RunOrSystem executes the configured protoc tool if pc is non-nil,
// or falls back to executing system "protoc".
func RunOrSystem(ctx context.Context, env map[string]string, pc *config.Protoc, args ...string) error {
	if pc != nil {
		return Run(ctx, env, pc, args...)
	}
	return command.RunWithEnv(ctx, env, "protoc", args...)
}

// InstallDir returns the directory where the protoc binary should be installed.
func InstallDir(version string) (string, error) {
	binDir, err := cache.BinDirectory()
	if err != nil {
		return "", err
	}
	return filepath.Join(binDir, protocDir, fmt.Sprintf("v%s", version)), nil
}

// downloadAndExtract downloads and installs the protoc binary from the given URL to the given directory.
func downloadAndExtract(ctx context.Context, url, dir, sha256 string) error {
	tarball := filepath.Join(dir, "protoc.zip")
	if err := fetch.Download(ctx, tarball, url, sha256); err != nil {
		return err
	}
	defer os.Remove(tarball)
	return filesystem.Unzip(ctx, tarball, dir)
}

// downloadURL returns the download URL for the protoc binary for the given version, OS, and arch.
func downloadURL(version, os, arch string) string {
	suffix := platformSuffix(os, arch)
	return fmt.Sprintf("%s/protocolbuffers/protobuf/releases/download/v%s/protoc-%s-%s.zip", githubURLBase, version, version, suffix)
}

// platformSuffix returns the platform suffix for the given OS and architecture.
func platformSuffix(os, arch string) string {
	if os == osWindows {
		return "win64"
	}

	return osMap[os] + "-" + archMap[arch]
}
