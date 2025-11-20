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
	"strconv"
	"strings"

	"github.com/googleapis/librarian/internal/command"
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

	newVersion, err := BumpPackageVersion(info.Package.Version)
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

// UpdateCargoVersion updates the version in a Cargo.toml file and formats it with taplo.
func UpdateCargoVersion(path, newVersion string) error {
	contents, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	info := cargo{
		Package: &crateInfo{
			Publish: true,
		},
	}
	if err := toml.Unmarshal(contents, &info); err != nil {
		return err
	}
	if info.Package == nil {
		return nil
	}
	if info.Package.Version == "" {
		return fmt.Errorf("no version found in %s", path)
	}

	info.Package.Version = newVersion
	updated, err := toml.Marshal(&info)
	if err != nil {
		return err
	}
	if err := os.WriteFile(path, updated, 0644); err != nil {
		return err
	}
	if err := command.Run("taplo", "fmt", path); err != nil {
		return fmt.Errorf("failed to format %s: %w", path, err)
	}
	return nil
}

// BumpPackageVersion increments the minor version and resets the patch version.
func BumpPackageVersion(version string) (string, error) {
	components := strings.SplitN(version, ".", 3)
	if len(components) != 3 {
		return version, nil
	}
	n, err := strconv.Atoi(components[1])
	if err != nil {
		return "", err
	}
	components[1] = fmt.Sprintf("%d", n+1)
	v := strings.Split(components[2], "-")
	v[0] = "0"
	components[2] = strings.Join(v, "-")
	return strings.Join(components, "."), nil
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
