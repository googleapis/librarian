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
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/google/yamlfmt/pkg/yaml"
	"github.com/googleapis/librarian/internal/config"
)

func updatePubspecDependencyVersions(lib *config.Library, defaults *config.Default, newDeps map[string]string) error {
	packageDir := libraryOutput(lib, defaults)
	pubspecPath := filepath.Join(packageDir, "pubspec.yaml")
	return modifyPubspecYaml(pubspecPath, func(root *yaml.Node) {
		for i := 0; i < len(root.Content); i += 2 {
			keyNode := root.Content[i]
			valNode := root.Content[i+1]

			if keyNode.Value == "dependencies" && valNode.Kind == yaml.MappingNode {
				for j := 0; j < len(valNode.Content); j += 2 {
					depKeyNode := valNode.Content[j]
					depValNode := valNode.Content[j+1]

					if constraint, ok := newDeps["package:"+depKeyNode.Value]; ok {
						depValNode.Kind = yaml.ScalarNode
						depValNode.Tag = "!!str"
						depValNode.Value = constraint
						depValNode.Content = nil
					}
				}
				return
			}
		}
	})
}

func updatePubspecVersion(path string, newVersion string) error {
	return modifyPubspecYaml(path, func(root *yaml.Node) {
		for i := 0; i < len(root.Content); i += 2 {
			keyNode := root.Content[i]
			valNode := root.Content[i+1]

			if keyNode.Value == "version" {
				valNode.Kind = yaml.ScalarNode
				valNode.Tag = "!!str"
				valNode.Value = newVersion
				valNode.Content = nil
				return
			}
		}
	})
}

func modifyPubspecYaml(path string, fn func(*yaml.Node)) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var node yaml.Node
	if err := yaml.Unmarshal(content, &node); err != nil {
		return fmt.Errorf("failed to parse pubspec.yaml: %w", err)
	}

	if node.Kind != yaml.DocumentNode || len(node.Content) == 0 {
		return errors.New("pubspec.yaml is empty")
	}
	root := node.Content[0]
	if root.Kind != yaml.MappingNode {
		return errors.New("pubspec.yaml is not a mapping")
	}

	fn(root)

	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(&node); err != nil {
		return fmt.Errorf("failed to encode pubspec.yaml: %w", err)
	}
	if err := enc.Close(); err != nil {
		return err
	}

	return os.WriteFile(path, buf.Bytes(), 0644)
}
