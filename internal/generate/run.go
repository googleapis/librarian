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

package generate

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"strings"
)

func Run(ctx context.Context, arg ...string) error {
	cmd := generatorCommand()

	if err := cmd.flags.Parse(arg); err != nil {
		return err
	}
	if len(cmd.flags.Args()) == 0 {
		cmd.flags.Usage()
		return fmt.Errorf("missing command")
	}

	c := cmd.flags.Args()[0]
	sub := cmd.lookup(c)
	if sub == nil {
		return fmt.Errorf("invalid command: %q", c)
	}
	if err := sub.flags.Parse(arg[1:]); err != nil {
		return err
	}
	return sub.run(ctx)
}

func runCommand(c string, args ...string) error {
	cmd := exec.Command(c, args...)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	slog.Info(strings.Repeat("-", 80))
	slog.Info(cmd.String())
	slog.Info(strings.Repeat("-", 80))
	return cmd.Run()
}
