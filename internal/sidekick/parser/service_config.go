// Copyright 2024 Google LLC
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

package parser

import (
	"os"
	"path/filepath"

	"github.com/googleapis/librarian/internal/serviceconfig"
	"github.com/googleapis/librarian/internal/sidekick/config"
)

func loadServiceConfig(cfg *Config) (*serviceconfig.Service, error) {
	if name := cfg.ServiceConfig; name != "" {
		return serviceconfig.Read(findServiceConfigPath(name, cfg.Source))
	}
	return nil, nil
}

// findServiceConfigPath finds the service config path for the current parser configuration.
//
// The service config files are specified as relative to the `googleapis-root`
// path (or `extra-protos-root` when set). This finds the right path given a
// configuration.
func findServiceConfigPath(serviceConfigFile string, options map[string]string) string {
	for _, opt := range config.SourceRoots(options) {
		dir, ok := options[opt]
		if !ok {
			// Ignore options that are not set
			continue
		}
		location := filepath.Join(dir, serviceConfigFile)
		stat, err := os.Stat(location)
		if err == nil && stat.Mode().IsRegular() {
			return location
		}
	}
	// Fallback to the current directory, it may fail but that is detected
	// elsewhere.
	return serviceConfigFile
}
