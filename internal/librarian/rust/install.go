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

package rust

import (
	"context"
	"errors"
	"fmt"

	"github.com/googleapis/librarian/internal/command"
	"github.com/googleapis/librarian/internal/config"
)

var ErrMissingToolVersion = errors.New("cargo tool missing version")

// Install installs cargo tool dependencies defined in the tools configuration.
func Install(ctx context.Context, tools *config.Tools) error {
	if len(tools.Cargo) == 0 {
		return nil
	}
	for _, tool := range tools.Cargo {
		if tool.Version == "" {
			return fmt.Errorf("%w: %s", ErrMissingToolVersion, tool.Name)
		}
		t := fmt.Sprintf("%s@%s", tool.Name, tool.Version)
		if err := command.Run(ctx, "cargo", "install", "--locked", t); err != nil {
			return err
		}
	}
	return nil
}
