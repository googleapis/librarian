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

package librarian

import (
	"errors"
	"fmt"

	"github.com/googleapis/librarian/internal/config"
)

var (
	// errUnsupportedPath is returned when a dot-notation path is not supported.
	errUnsupportedPath = errors.New("unsupported config path")
)

// setConfigValue sets a value at a specific path within the configuration.
func setConfigValue(cfg *config.Config, path string, value string) (*config.Config, error) {
	switch path {
	case "version":
		cfg.Version = value
		return cfg, nil
	default:
		return nil, fmt.Errorf("%w: %s", errUnsupportedPath, path)
	}
}

// getConfigValue returns the value at a specific path within the configuration.
func getConfigValue(cfg *config.Config, path string) (string, error) {
	switch path {
	case "version":
		return cfg.Version, nil
	default:
		return "", fmt.Errorf("%w: %s", errUnsupportedPath, path)
	}
}
