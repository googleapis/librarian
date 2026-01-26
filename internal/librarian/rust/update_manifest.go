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

package rust

import (
	"context"
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/googleapis/librarian/internal/command"
)

// CrateInfo contains the package information.
type CrateInfo struct {
	Name    string `toml:"name"`
	Version string `toml:"version"`
	Publish bool   `toml:"publish"`
}

// Cargo is a wrapper for CrateInfo for parsing Cargo.toml files.
type Cargo struct {
	Package *CrateInfo `toml:"package"`
}

// updateCargoVersion updates the version in a Cargo.toml file. It uses a
// line-based approach to preserve comments and formatting, which is important
// because some Cargo.toml files are hand-crafted and contain comments that
// must be preserved.
func updateCargoVersion(path, newVersion string) error {
	contents, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	lines := strings.Split(string(contents), "\n")
	idx := slices.IndexFunc(lines, func(a string) bool { return strings.HasPrefix(a, "version ") })
	if idx == -1 {
		return fmt.Errorf("no version field found in %q", path)
	}
	// The number of spaces may seem weird. They match the number of spaces in
	// the mustache template.
	lines[idx] = fmt.Sprintf(`version                = "%s"`, newVersion)
	return os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0644)
}

// shouldBumpManifestVersion checks if the manifest version needs to be bumped.
// It returns false if the version has already been updated since the last tag.
func shouldBumpManifestVersion(ctx context.Context, gitExe, lastTag, manifest string) (bool, error) {
	delta := fmt.Sprintf("%s..HEAD", lastTag)
	contents, err := command.Output(ctx, gitExe, "diff", delta, "--", manifest)
	if err != nil {
		return false, err
	}
	if len(contents) == 0 {
		return true, nil
	}
	lines := strings.Split(contents, "\n")
	has := func(prefix string) bool {
		return slices.ContainsFunc(lines, func(line string) bool { return strings.HasPrefix(line, prefix) })
	}
	return !has("+version "), nil
}
