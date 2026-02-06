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

package python

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

const gapicVersionLinePrefix = "__version__ = "
const gapicVersionFile = "gapic_version.py"

var (
	errMultipleVersions   = errors.New("multiple versions found")
	errNoVersionFound     = errors.New("no version found")
	errSymLinkVersionFile = errors.New("version file is a symlink")
)

// Bump updates the version number in the library with the given output
// directory.
func Bump(output, version string) error {
	return bumpGapicVersions(output, version)
}

// bumpGapicVersion finds all gapic_version.py files under output. For each
// file, it expect to find a line starting "__version__ = ", and changes that
// line to end with the specified version (in quotes, with no escaping applied -
// we don't expect version numbers to require quoting). Any existing text after
// "__version__ = " (on the same line) is omitted. All other lines are
// preserved.
func bumpGapicVersions(output, version string) error {
	return filepath.WalkDir(output, func(path string, d fs.DirEntry, err error) error {
		if d.Name() == gapicVersionFile {
			if !d.Type().IsRegular() {
				return fmt.Errorf("failed to bump %s: %w", path, errSymLinkVersionFile)
			}
			return bumpSingleGapicVersionFile(path, version)
		}
		return nil
	})
}

// Bumps a single gapic_version.py file as described in bumpGapicVersions.
func bumpSingleGapicVersionFile(path, version string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to bump '%s': %w", path, err)
	}
	lines := strings.Split(string(data), "\n")
	var foundLine bool
	for i, line := range lines {
		if strings.HasPrefix(line, gapicVersionLinePrefix) {
			if foundLine {
				return fmt.Errorf("failed to bump '%s': %w", path, errMultipleVersions)
			}
			lines[i] = fmt.Sprintf("%s\"%s\"", gapicVersionLinePrefix, version)
			foundLine = true
		}
	}
	if !foundLine {
		return fmt.Errorf("failed to bump '%s': %w", path, errNoVersionFound)
	}
	if err := os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0644); err != nil {
		return fmt.Errorf("failed to bump '%s': %w", path, err)
	}
	return nil
}
