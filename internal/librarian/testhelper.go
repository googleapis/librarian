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

package librarian

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/googleapis/librarian/internal/config"
)

// testReleaseVersion is the version that libraries are always released at when
// using the testhelper language implementation.
const testReleaseVersion = "1.2.3"

func testReleaseLibrary(cfg *config.Config, lib *config.Library) (*config.Config, error) {
	lib.Version = testReleaseVersion
	return cfg, nil
}

func testGenerate(library *config.Library) error {
	if err := os.MkdirAll(library.Output, 0755); err != nil {
		return err
	}
	content := fmt.Sprintf("# %s\n\nGenerated library\n", library.Name)
	readmePath := filepath.Join(library.Output, "README.md")
	return os.WriteFile(readmePath, []byte(content), 0644)
}
