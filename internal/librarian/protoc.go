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
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/fetch"
)

// install installs the protoc tool.
func install(ctx context.Context, protoc *config.Protoc) error {
	url := downloadURL(protoc.Version)
	dir, err := installDir(protoc.Version)
	if err != nil {
		return err
	}
	tarball := filepath.Join(dir, "protoc.zip")
	if err := fetch.Download(ctx, tarball, url, protoc.SHA256); err != nil {
		return err
	}
	defer os.Remove(tarball)
	filter := func(path string) (string, bool) {
		cleanPath := filepath.Clean(path)
		if cleanPath == "bin/protoc" || cleanPath == "include" {
			return cleanPath, true
		}
		return "", false
	}
	return fetch.ExtractTarball(tarball, dir, filter)
}

// downloadURL returns the download URL for the protoc binary for the given version.
func downloadURL(version string) string {
	return fmt.Sprintf("https://github.com/protocolbuffers/protobuf/releases/download/v%s/protoc-%s-linux-x86_64.zip", version, version)
}

// installDir returns the directory where the protoc binary should be installed.
func installDir(version string) (string, error) {
	binDir, err := cache.BinDirectory()
	if err != nil {
		return "", err
	}
	return filepath.Join(binDir, "protoc", fmt.Sprintf("v%s", version)), nil
}
