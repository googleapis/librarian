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

package python

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/googleapis/librarian/internal/command"
	"github.com/googleapis/librarian/internal/config"
)

// Format formats and lints a generated Python library using ruff.
func Format(ctx context.Context, library *config.Library) error {
	if library == nil {
		return ErrNilLibrary
	}
	if library.Output == "" {
		return ErrEmptyOutput
	}
	outdir, err := filepath.Abs(library.Output)
	if err != nil {
		return fmt.Errorf("failed to resolve output directory: %w", err)
	}
	if err := command.RunInDir(ctx, outdir, "ruff", "check", "--select", "I", "--fix", outdir); err != nil {
		return err
	}
	return command.RunInDir(ctx, outdir, "ruff", "format", outdir)
}
