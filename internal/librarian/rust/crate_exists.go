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
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
)

// crateExists determines if a crate (the Rust mapping for "library") exists.
//
// Once a crate is generated the following things are true:
// - The output directory exists
// - The output directory is listed in the top-level Cargo.toml file
//
// If none of those things are true, it is safe to assume the crate is a new
// crate and to execute the commands to add the crate to the repository. If
// both are true, then we can skip the commands to add the crate.
//
// If one of them is true, but not the other, then there is an error in the
// repository configuration that requires human intervention.
func crateExists(output string) (bool, error) {
	hasDirectory := true
	if _, err := os.Stat(output); err != nil {
		hasDirectory = false
		if !errors.Is(err, fs.ErrNotExist) {
			return false, fmt.Errorf("cannot access output directory %q: %w", output, err)
		}
	}
	// The librarian commands are always executed at the top-level directory of
	// the monorepo.
	contents, err := os.ReadFile("Cargo.toml")
	if err != nil {
		return false, err
	}
	hasCargoEntry := bytes.Contains(contents, fmt.Appendf(nil, "\n  \"%s\",\n", output))
	if !hasCargoEntry && hasDirectory {
		return false, fmt.Errorf("inconsistent repository state, crate missing in Cargo.toml, but the directory exists: %s", output)
	}
	if hasCargoEntry && !hasDirectory {
		return false, fmt.Errorf("inconsistent repository state, crate listed in Cargo.toml, but the directory is missing: %s", output)
	}
	return hasDirectory && hasCargoEntry, nil
}
