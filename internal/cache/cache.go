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

// Package cache manages the local cache directory for librarian operations.
package cache

import (
	"fmt"
	"os"
	"path/filepath"
)

const (
	// EnvLibrarianBin is the environment variable used to override the default bin directory.
	EnvLibrarianBin = "LIBRARIAN_BIN"
	// EnvLibrarianCache is the environment variable used to override the default cache directory.
	EnvLibrarianCache = "LIBRARIAN_CACHE"
)

// Directory returns the root cache directory for librarian operations. It
// checks the $LIBRARIAN_CACHE environment variable, falling back to
// $HOME/.cache/librarian if not set.
func Directory() (string, error) {
	if cache := os.Getenv(EnvLibrarianCache); cache != "" {
		return cache, nil
	}

	home, err := os.UserCacheDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user cache directory: %w", err)
	}
	return filepath.Join(home, "librarian"), nil
}

// BinDirectory returns the directory where librarian bin files are stored.
// It checks the $LIBRARIAN_BIN environment variable, falling back to the "bin" subdirectory
// under the cache directory if not set.
func BinDirectory() (string, error) {
	if binDir := os.Getenv(EnvLibrarianBin); binDir != "" {
		return binDir, nil
	}
	cacheDir, err := Directory()
	if err != nil {
		return "", err
	}
	return filepath.Join(cacheDir, "bin"), nil
}
