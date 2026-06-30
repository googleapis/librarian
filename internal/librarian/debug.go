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
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/googleapis/librarian/internal/cache"
	"github.com/googleapis/librarian/internal/librarian/golang"
	"github.com/googleapis/librarian/internal/librarian/java"
	"github.com/urfave/cli/v3"
)

// debugCommand returns the CLI command for librarian debugging tools.
func debugCommand() *cli.Command {
	return &cli.Command{
		Name:      "debug",
		Usage:     "various debugging commands",
		UsageText: "librarian debug [command]",
		Commands: []*cli.Command{
			envCommand(),
		},
	}
}

// envCommand returns the CLI command for printing the librarian environment.
func envCommand() *cli.Command {
	return &cli.Command{
		Name:      "env",
		Usage:     "print environment variables for the librarian command line interface.",
		UsageText: "librarian debug env",
		Description: `env prints the librarian interpretation of the environment it is run in.
This includes the resolved LIBRARIAN_CACHE and LIBRARIAN_BIN paths,
as well as the language-specific tool installation directories.`,
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return runEnv(cmd.Root().Writer)
		},
	}
}

func runEnv(w io.Writer) error {
	cacheDir := dirOrErr(cache.Directory())
	buildDir := dirOrErr(cache.BinDirectory())
	goToolsDir := dirOrErr(golang.InstallDir())
	javaToolsDir := dirOrErr(java.InstallDir())
	var b strings.Builder
	fmt.Fprintf(&b, "LIBRARIAN_CACHE=%s\n", cacheDir)
	fmt.Fprintf(&b, "LIBRARIAN_BIN=%s\n", buildDir)
	fmt.Fprintln(&b)
	fmt.Fprintln(&b, "Language-specific tool installation directories:")
	fmt.Fprintf(&b, "  golang: %s\n", goToolsDir)
	fmt.Fprintf(&b, "  java: %s\n", javaToolsDir)
	_, err := io.WriteString(w, b.String())
	return err
}

// dirOrErr converts a directory path and potential error into a string. If an error
// occurred, it returns a formatted error string; otherwise, it returns the directory path.
func dirOrErr(dir string, err error) string {
	if err != nil {
		return fmt.Sprintf("<error: %v>", err)
	}
	return dir
}
