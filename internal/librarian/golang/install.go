// Copyright 2025 Google LLC
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

package golang

import (
	"context"
	"os"
	"path/filepath"

	"github.com/googleapis/librarian"
	"github.com/googleapis/librarian/internal/command"
)

// Install installs Go tool dependencies defined in the librarian module's go.mod tool directive.
func Install(ctx context.Context) error {
	dir, err := os.MkdirTemp("", "librarian-install-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), librarian.GoMod, 0644); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dir, "go.sum"), librarian.GoSum, 0644); err != nil {
		return err
	}
	return command.RunInDir(ctx, dir, "go", "install", "tool")
}
