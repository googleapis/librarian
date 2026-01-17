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

func fakeReleaseLibrary(lib *config.Library, nextVersion string) error {
	lib.Version = nextVersion
	return nil
}

func fakeGenerate(library *config.Library) error {
	if _, err := os.Stat(library.Output); err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("failed to stat output directory %q: %w", library.Output, err)
		}
		if err := fakeCreateSkeleton(library); err != nil {
			return err
		}
	}
	content := fmt.Sprintf("# %s\n\nGenerated library\n", library.Name)
	readmePath := filepath.Join(library.Output, "README.md")
	if err := os.WriteFile(readmePath, []byte(content), 0644); err != nil {
		return err
	}
	versionPath := filepath.Join(library.Output, "VERSION")
	if _, err := os.Stat(versionPath); os.IsNotExist(err) {
		return os.WriteFile(versionPath, []byte("0.0.0"), 0644)
	}
	return nil
}

func fakeFormat(library *config.Library) error {
	readmePath := filepath.Join(library.Output, "README.md")
	content, err := os.ReadFile(readmePath)
	if err != nil {
		return err
	}
	formatted := string(content) + "\n---\nFormatted\n"
	return os.WriteFile(readmePath, []byte(formatted), 0644)
}

func fakePublish() error {
	return os.WriteFile("PUBLISHED", []byte("published\n"), 0644)
}

func fakeCreateSkeleton(library *config.Library) error {
	if err := os.MkdirAll(library.Output, 0755); err != nil {
		return err
	}
	content := fmt.Sprintf("# %s\n\nThis is a starter file.\n", library.Name)
	starterPath := filepath.Join(library.Output, "STARTER.md")
	return os.WriteFile(starterPath, []byte(content), 0644)
}
