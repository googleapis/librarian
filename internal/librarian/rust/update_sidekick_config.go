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
	"errors"
	"fmt"
	"os"
	"path"
	"slices"
	"strings"
)

// UpdateSidekickConfig updates the version in a .sidekick.toml file.
// It uses a line-based approach to preserve comments and formatting.
func UpdateSidekickConfig(manifest, newVersion string) error {
	dir, _ := path.Split(manifest)
	config := path.Join(dir, ".sidekick.toml")
	_, err := os.Stat(config)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	contents, err := os.ReadFile(config)
	if err != nil {
		return err
	}
	lines := strings.Split(string(contents), "\n")
	iCodec := slices.IndexFunc(lines, func(a string) bool { return strings.HasPrefix(a, "[codec]") })
	if iCodec == -1 {
		return fmt.Errorf("expected a line starting with `[codec]` in the sidekick config, got=%v", lines)
	}
	var result []string
	result = append(result, lines[:iCodec+1]...)
	tail := lines[iCodec+1:]
	iVersion := slices.IndexFunc(tail, func(a string) bool { return strings.HasPrefix(a, "version ") && strings.Contains(a, " = ") })
	verLine := fmt.Sprintf(`version        = '%s'`, newVersion)
	if iVersion == -1 {
		result = append(result, verLine)
		result = append(result, tail...)
	} else {
		tail[iVersion] = verLine
		result = append(result, tail...)
	}
	return os.WriteFile(config, []byte(strings.Join(result, "\n")), 06444)
}
