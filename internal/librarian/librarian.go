// Copyright 2024 Google LLC
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

package librarian

import (
	"context"
	"fmt"
	"log/slog"
)

// Run executes the Librarian CLI with the given command line
// arguments.
func Run(ctx context.Context, arg ...string) error {
	cmd := newLibrarianCommand()
	slog.Info("librarian", "arguments", arg)
	if err := cmd.Config.SetDefaults(); err != nil {
		return fmt.Errorf("failed to initialize config: %w", err)
	}
	if _, err := cmd.Config.IsValid(); err != nil {
		return fmt.Errorf("failed to validate config: %s", err)
	}
	return cmd.Run(ctx, arg)
}
