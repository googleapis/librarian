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

package dart

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/googleapis/librarian/internal/config"
)

func updatePubspecDependencyVersions(lib *config.Library, defaults *config.Default, newDeps map[string]string) error {
	packageDir := libraryOutput(lib, defaults)
	pubspecPath := filepath.Join(packageDir, "pubspec.yaml")
	content, err := os.ReadFile(pubspecPath)
	if err != nil {
		return err
	}
	lines := strings.Split(string(content), "\n")
	var newLines []string
	inDeps := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(line, "dependencies:") {
			inDeps = true
			newLines = append(newLines, line)
			continue
		} else if inDeps && len(line) > 0 && !strings.HasPrefix(line, "  ") && !strings.HasPrefix(line, "#") {
			inDeps = false
		}

		if inDeps && strings.HasPrefix(line, "  ") {
			parts := strings.SplitN(trimmed, ":", 2)
			if len(parts) == 2 {
				depName := strings.TrimSpace(parts[0])
				if constraint, ok := newDeps[depName]; ok {
					indent := line[:len(line)-len(strings.TrimLeft(line, " \t"))]
					newLines = append(newLines, fmt.Sprintf("%s%s: %s", indent, depName, constraint))
					continue
				}
			}
		}
		newLines = append(newLines, line)
	}
	return os.WriteFile(pubspecPath, []byte(strings.Join(newLines, "\n")), 0644)
}

func updatePubspecVersion(path string, newVersion string) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	lines := strings.Split(string(content), "\n")
	var newLines []string
	for _, line := range lines {
		if strings.HasPrefix(line, "version:") {
			newLines = append(newLines, fmt.Sprintf("version: %s", newVersion))
			continue
		}
		newLines = append(newLines, line)
	}
	return os.WriteFile(path, []byte(strings.Join(newLines, "\n")), 0644)
}
