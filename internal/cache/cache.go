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

// EnvLibrarianCache is the environment variable used to override the default cache directory.
const EnvLibrarianCache = "LIBRARIAN_CACHE"

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
