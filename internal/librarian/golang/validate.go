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

package golang

import (
	"errors"
	"fmt"
	"path/filepath"

	"github.com/googleapis/librarian/internal/config"
)

// Validate checks the Go-specific configuration of every library. It applies
// lexical checks to internal copies without consulting the filesystem, so
// that "librarian tidy" rejects a copy that is
// null, resolves outside its library, overlaps another generated directory,
// reuses a proto package or lists a malformed plugin.
func Validate(cfg *config.Config) error {
	var defaultOutput string
	if cfg.Default != nil {
		defaultOutput = cfg.Default.Output
	}
	var errs []error
	for _, library := range cfg.Libraries {
		output := library.Output
		if output == "" {
			output = DefaultOutput(library.Name, defaultOutput)
		}
		if err := validateInternalCopies(library, output); err != nil {
			errs = append(errs, fmt.Errorf("library %q: %w", library.Name, err))
		}
		if preview := library.Preview; preview != nil {
			previewOutput := preview.Output
			if previewOutput == "" {
				previewOutput = filepath.Join("preview", "internal", output)
			}
			if err := validateInternalCopies(preview, previewOutput); err != nil {
				errs = append(errs, fmt.Errorf("library %q preview: %w", library.Name, err))
			}
		}
	}
	return errors.Join(errs...)
}
