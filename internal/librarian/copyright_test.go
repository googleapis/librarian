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
	"testing"
	"time"
)

func TestExtractCopyrightYear(t *testing.T) {
	currentYear := fmt.Sprintf("%d", time.Now().Year())

	for _, test := range []struct {
		name     string
		filename string
		language string
		content  string
		wantYear string
	}{
		{
			name:     "rust file with copyright",
			filename: "Cargo.toml",
			language: languageRust,
			content:  "# Copyright 2024 Google LLC\n[package]\nname = \"test\"\n",
			wantYear: "2024",
		},
		{
			name:     "python file with copyright",
			filename: "setup.py",
			language: languagePython,
			content:  "# -*- coding: utf-8 -*-\n# Copyright 2025 Google LLC\n\nimport setuptools\n",
			wantYear: "2025",
		},
		{
			name:     "rust file does not exist",
			filename: "Cargo.toml",
			language: languageRust,
			wantYear: currentYear,
		},
		{
			name:     "rust file without copyright line",
			filename: "Cargo.toml",
			language: languageRust,
			content:  "[package]\nname = \"test\"\nversion = \"0.1.0\"\n",
			wantYear: currentYear,
		},
		{
			name:     "python file without copyright line",
			filename: "setup.py",
			language: languagePython,
			content:  "# -*- coding: utf-8 -*-\n\nimport setuptools\n",
			wantYear: currentYear,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			dir := t.TempDir()
			if err := os.WriteFile(filepath.Join(dir, test.filename), []byte(test.content), 0644); err != nil {
				t.Fatal(err)
			}
			got, err := ExtractCopyrightYear(dir, test.language)
			if err != nil {
				t.Fatal(err)
			}
			if got != test.wantYear {
				t.Errorf("got year %q, want %q", got, test.wantYear)
			}
		})
	}
}
