// Copyright 2024 Google LLC
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

package sidekick

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path"
	"path/filepath"
	"strings"

	cmd "github.com/googleapis/librarian/internal/command"
	"github.com/googleapis/librarian/internal/fetch"
	"github.com/googleapis/librarian/internal/sidekick/config"
)

var (
	downloadTarball = fetch.DownloadTarball
)

func makeSourceRoot(ctx context.Context, rootConfig *config.Config, configPrefix string) (string, error) {
	sourceRoot, ok := rootConfig.Source[fmt.Sprintf("%s-root", configPrefix)]
	if !ok {
		return "", nil
	}
	if ok := isDirectory(sourceRoot); ok {
		return sourceRoot, nil
	}
	if !requiresDownload(sourceRoot) {
		return "", fmt.Errorf("only directories and https URLs are supported for googleapis-root")
	}
	// Treat `googleapis-root` as a URL to download. We want to avoid downloads
	// if possible, so we will first try to use a cache directory in $HOME.
	// Only if that fails we try a new download.
	source, ok := rootConfig.Source[fmt.Sprintf("%s-sha256", configPrefix)]
	if !ok {
		return "", fmt.Errorf("using an https:// URL for googleapis-root requires setting googleapis-sha256")
	}
	cacheDir, err := getCacheDir(rootConfig)
	if err != nil {
		return "", err
	}
	target := path.Join(cacheDir, source)
	if isDirectory(target) {
		return target, nil
	}
	tgz := target + ".tar.gz"
	if err := downloadTarball(ctx, tgz, sourceRoot, source); err != nil {
		return "", err
	}

	if err := extractTarball(tgz, cacheDir); err != nil {
		slog.Error("error extracting .tar.gz file", "file", tgz, "cacheDir", cacheDir, "error", err)
		return "", err
	}
	dirname := extractedName(rootConfig, sourceRoot, configPrefix)
	if err := os.Rename(path.Join(cacheDir, dirname), target); err != nil {
		return "", err
	}
	return target, nil
}

func extractTarball(source, destination string) error {
	return cmd.Run("tar", "-zxf", source, "-C", destination)
}

func extractedName(rootConfig *config.Config, googleapisRoot, configPrefix string) string {
	name, ok := rootConfig.Source[fmt.Sprintf("%s-extracted-name", configPrefix)]
	if ok {
		return name
	}
	return "googleapis-" + filepath.Base(strings.TrimSuffix(googleapisRoot, ".tar.gz"))
}

func isDirectory(name string) bool {
	stat, err := os.Stat(name)
	if err != nil {
		return false
	}
	if !stat.IsDir() {
		return false
	}
	return true
}

func getCacheDir(rootConfig *config.Config) (string, error) {
	cacheDir, ok := rootConfig.Source["cachedir"]
	if !ok {
		var err error
		if cacheDir, err = os.UserCacheDir(); err != nil {
			return "", err
		}
	}
	return path.Join(cacheDir, "sidekick", "downloads"), nil
}

func requiresDownload(googleapisRoot string) bool {
	return strings.HasPrefix(googleapisRoot, "https://") || strings.HasPrefix(googleapisRoot, "http://")
}
