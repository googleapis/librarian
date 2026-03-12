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
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// checkAndClean removes all files in dir except those in keep. The keep list
// should contain paths relative to dir. It returns an error if any file
// in keep does not exist.
func checkAndClean(dir string, keep []string) error {
	keepSet := make(map[string]bool)
	for _, k := range keep {
		keepSet[filepath.Clean(k)] = true
	}
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		if keepSet[rel] {
			keepSet[rel] = false
			return nil
		}
		return os.Remove(path)
	})
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			// The top-level directory was not found. This happens when
			// calling `librarian generate` on new libraries and it is not
			// an error.
			return nil
		}
		return err
	}
	var missing []string
	for relative, v := range keepSet {
		if v {
			missing = append(missing, relative)
		}
	}
	if len(missing) != 0 {
		return fmt.Errorf("some keep files %q do not exist", keep)
	}
	return nil
}
