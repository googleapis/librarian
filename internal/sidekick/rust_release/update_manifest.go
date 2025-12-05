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

package rustrelease

import (
	"fmt"
	"os"
	"os/exec"
	"slices"
	"strings"

	"github.com/googleapis/librarian/internal/semver"
	"github.com/googleapis/librarian/internal/sidekick/config"
	"github.com/pelletier/go-toml/v2"
)

// crateInfo contains the package information.
type crateInfo struct {
	Name    string `toml:"name"`
	Version string `toml:"version"`
	Publish bool   `toml:"publish"`
}

type cargo struct {
	Package *crateInfo `toml:"package"`
}

func updateManifest(config *config.Release, lastTag, manifest string) ([]string, error) {
	needsBump, err := manifestVersionNeedsBump(config, lastTag, manifest)
	if err != nil {
		return nil, err
	}
	if !needsBump {
		return nil, nil
	}
	contents, err := os.ReadFile(manifest)
	if err != nil {
		return nil, err
	}
	info := cargo{
		Package: &crateInfo{
			Publish: true,
		},
	}
	if err := toml.Unmarshal(contents, &info); err != nil {
		return nil, err
	}
	if !info.Package.Publish {
		return nil, nil
	}
	// Only ever take a minor version bump.
	// TODO(https://github.com/googleapis/librarian/issues/3182): Implement desired pre-1.0.0 semantics.
	newVersion, err := semver.DeriveNextOptions{BumpVersionCore: true}.DeriveNext(semver.Minor, info.Package.Version)
	if err != nil {
		return nil, err
	}
	if err := UpdateCargoVersion(manifest, newVersion); err != nil {
		return nil, err
	}
	if err := updateSidekickConfig(manifest, newVersion); err != nil {
		return nil, err
	}
	return []string{info.Package.Name}, nil
}

// UpdateCargoVersion updates the version in a Cargo.toml file. It uses a
// line-based approach to preserve comments and formatting, which is important
// because some Cargo.toml files are hand-crafted and contain comments that
// must be preserved.
func UpdateCargoVersion(path, newVersion string) error {
	contents, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	lines := strings.Split(string(contents), "\n")
	idx := slices.IndexFunc(lines, func(a string) bool { return strings.HasPrefix(a, "version ") })
	if idx == -1 {
		return fmt.Errorf("expected a line starting with `version ` in %v", lines)
	}
	// The number of spaces may seem weird. They match the number of spaces in
	// the mustache template.
	lines[idx] = fmt.Sprintf(`version                = "%s"`, newVersion)
	return os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0644)
}

func manifestVersionNeedsBump(config *config.Release, lastTag, manifest string) (bool, error) {
	delta := fmt.Sprintf("%s..HEAD", lastTag)
	cmd := exec.Command(gitExe(config), "diff", delta, "--", manifest)
	cmd.Dir = "."
	contents, err := cmd.CombinedOutput()
	if err != nil {
		return false, err
	}
	if len(contents) == 0 {
		return true, nil
	}
	lines := strings.Split(string(contents), "\n")
	has := func(prefix string) bool {
		return slices.ContainsFunc(lines, func(line string) bool { return strings.HasPrefix(line, prefix) })
	}
	return !has("+version "), nil
}
