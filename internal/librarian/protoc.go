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

package librarian

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/googleapis/librarian/internal/cache"
	"github.com/googleapis/librarian/internal/fetch"
	"github.com/googleapis/librarian/internal/filesystem"
)

const githubURLBase = "https://github.com"

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

// installProtoc downloads and installs the protoc binary from the given URL to the given directory.
func installProtoc(ctx context.Context, url, dir, sha256 string) error {
	tarball := filepath.Join(dir, "protoc.zip")
	if err := fetch.Download(ctx, tarball, url, sha256); err != nil {
		return err
	}
	defer os.Remove(tarball)
	return filesystem.Unzip(ctx, tarball, dir)
}

// protocDownloadURL returns the download URL for the protoc binary for the given version, OS, and arch.
func protocDownloadURL(version, os, arch string) string {
	suffix := platformSuffix(os, arch)
	return fmt.Sprintf("%s/protocolbuffers/protobuf/releases/download/v%s/protoc-%s-%s.zip", githubURLBase, version, version, suffix)
}

// protocInstallDir returns the directory where the protoc binary should be installed.
func protocInstallDir(version string) (string, error) {
	binDir, err := cache.BinDirectory()
	if err != nil {
		return "", err
	}
	return filepath.Join(binDir, "protoc", fmt.Sprintf("v%s", version)), nil
}

// platformSuffix returns the platform suffix for the given OS and architecture.
func platformSuffix(os, arch string) string {
	if os == "windows" {
		return "win64"
	}

	return osMap[os] + "-" + archMap[arch]
}
