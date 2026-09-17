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

package swift

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/googleapis/librarian/internal/command"
	"github.com/googleapis/librarian/internal/config"
)

var (
	errMissingVersion = errors.New("must provide version")
)

const (
	clientsManifest        = "Clients.swift"
	packageVersionManifest = "PackageVersion.swift"
)

// Bump checks if a version bump is required and performs it.
// It returns without error if no bump is needed (version already updated since lastTag)
// or if no version manifest exists for the library (e.g., data-only packages).
func Bump(ctx context.Context, library *config.Library, output, version, gitExe, lastTag string) error {
	if version == "" {
		return errMissingVersion
	}
	packageDir := PackageDirectory(output)
	sourcesDir := filepath.Join(packageDir, "Sources")
	var versionFile string
	if _, err := os.Stat(sourcesDir); err == nil {
		err := filepath.WalkDir(sourcesDir, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				return nil
			}
			switch filepath.Base(path) {
			case clientsManifest:
				versionFile = path
			case packageVersionManifest:
				if versionFile == "" {
					versionFile = path
				}
			}
			return nil
		})
		if err != nil {
			return err
		}
	} else if !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	if versionFile == "" {
		// Some packages (e.g. data-only or protobuf packages) have no Clients.swift or PackageVersion.swift.
		return nil
	}
	skip, err := versionAlreadyBumped(ctx, gitExe, lastTag, versionFile)
	if err != nil {
		return err
	}
	if skip {
		return nil
	}
	library.Version = version
	return nil
}

// versionAlreadyBumped checks if the version was bumped since the last release.
//
// `librarian bump` should be idempotent until the next release. That makes it
// safe to run the command multiple times until a release is out, and allows
// manual tweaks of the version if needed.
func versionAlreadyBumped(ctx context.Context, gitExe, lastTag, versionFile string) (bool, error) {
	delta := fmt.Sprintf("%s..HEAD", lastTag)
	contents, err := command.Output(ctx, gitExe, "diff", delta, "--", versionFile)
	if err != nil {
		return false, err
	}
	lines := strings.Split(contents, "\n")
	has := func(prefix string) bool {
		return slices.ContainsFunc(lines, func(line string) bool {
			return strings.HasPrefix(line, prefix) && (strings.Contains(line, "packageVersion:") || strings.Contains(line, "static let version:"))
		})
	}
	return has("-") && has("+"), nil
}
