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

// Package librarianops provides operations related to the Librarian tool.
package librarianops

import (
	"context"
	"fmt"
	"strings"

	"github.com/googleapis/librarian/internal/command"
)

// GetLatestLibrarianVersion run `go list -m -f '{{.Version}}' github.com/googleapis/librarian@main` to
// get librarian version. The command is essentially a "deep dive" request for information about a
// specific Go module version.
func GetLatestLibrarianVersion(ctx context.Context) (string, error) {
	version, err := command.Output(ctx, "go", "list", "-m", "-f", "{{.Version}}", "github.com/googleapis/librarian@main")
	if err != nil {
		return "", fmt.Errorf("getting librarian version failed: %w", err)
	}
	return strings.TrimSpace(version), nil
}
