// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package config

import (
	"fmt"
)

// GlobalConfig defines the contract for the config.yaml file.
type GlobalConfig struct {
	GlobalFilesAllowlist []*GlobalFile `yaml:"global_files_allowlist"`
}

// GlobalFile defines the global files in language repositories.
type GlobalFile struct {
	Path        string     `yaml:"path"`
	Permissions Permission `yaml:"permissions"`
}

type Permission int

const (
	ReadOnly Permission = iota
	WriteOnly
	ReadWrite
	Unknown
)

func (permission Permission) String() string {
	switch permission {
	case ReadOnly:
		return "ReadOnly"
	case WriteOnly:
		return "WriteOnly"
	case ReadWrite:
		return "ReadWrite"
	default:
		return "Unknown"
	}
}

func (g *GlobalConfig) Validate() error {
	for i, globalFile := range g.GlobalFilesAllowlist {
		path, permissions := globalFile.Path, globalFile.Permissions
		if !isValidDirPath(path) {
			return fmt.Errorf("invalid global file path at index %d: %q", i, path)
		}
		if permissions == Unknown {
			return fmt.Errorf("invalid global file permission at index %d: %q", i, permissions)
		}
	}

	return nil
}
