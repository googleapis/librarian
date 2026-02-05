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

// Package importconfig provides orchestration for importing configurations.
package importconfig

import (
	"context"

	"github.com/urfave/cli/v3"
)

// Run executes the importconfig command with the given arguments.
func Run(ctx context.Context, args ...string) error {
	cmd := &cli.Command{
		Name:      "import-configs",
		Usage:     "orchestrate import-config operations",
		UsageText: "import-configs [command]",
		Commands: []*cli.Command{
			updateTransportsCommand(),
		},
	}
	return cmd.Run(ctx, args)
}
