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

package librarianops

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/yaml"
)

// GenerateLibrarianConfigPath given a repository directory, returns the path to the librarian config file.
// For example if the repoDir is "/path/to/repo", the config file path will be "/path/to/repo/librarian.yaml".
func GenerateLibrarianConfigPath(repoDir string) string {
	return filepath.Join(repoDir, "librarian.yaml")
}

// Reads the librarian.yaml config file and returns a Config struct. If the file does not exist, returns an error.
func getConfigFile(configPath string) (*config.Config, error) {
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("config file does not exist at path: %s", configPath)
	}
	return yaml.Read[config.Config](configPath)
}

// UpdateLibrarianVersionInConfigFile updates the version field in the librarian.yaml config file with the provided version.
// If the file does not exist, returns an error.
func UpdateLibrarianVersionInConfigFile(version, configPath string) error {
	configFile, err := getConfigFile(configPath)
	if err != nil {
		return err
	}
	configFile.Version = version
	return yaml.Write(configPath, configFile)
}
