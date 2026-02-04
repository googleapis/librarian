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

// Package librarianops provides orchestration for running librarian across
// multiple repositories.
package librarianops

import (
	"context"

	"github.com/urfave/cli/v3"
)

const (
	repoRust = "google-cloud-rust"
	repoFake = "fake-repo" // used for testing
)

var supportedRepositories = map[string]bool{
	repoFake: true, // used for testing
	repoRust: true,
}

// Run executes the librarianops command with the given arguments.
func Run(ctx context.Context, args ...string) error {
	cmd := &cli.Command{
		Name:      "librarianops",
		Usage:     "orchestrate librarian operations across multiple repositories",
		UsageText: "librarianops [command]",
		Commands: []*cli.Command{
			generateCommand(),
			updateTransportsCommand(),
		},
	}
	return cmd.Run(ctx, args)
}
