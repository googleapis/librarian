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

package cpp

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/testhelper"
)

func TestFormat_EmptyOutput(t *testing.T) {
	if err := Format(t.Context(), nil, &config.Library{Output: ""}); err != nil {
		t.Fatalf("expected nil error for empty output, got: %v", err)
	}
	if err := Format(t.Context(), nil, nil); err != nil {
		t.Fatalf("expected nil error for nil library, got: %v", err)
	}
}

func TestFormat_NoCppFiles(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "CMakeLists.txt"), []byte("# cmake\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Format(t.Context(), nil, &config.Library{Output: dir}); err != nil {
		t.Fatalf("expected nil error when no .h or .cc files present, got: %v", err)
	}
}

func TestFormat_MissingClangFormatFailsLoudly(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "service.h"), []byte("class Service {};\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg := &config.Config{
		Tools: &config.Tools{
			ClangFormat: &config.ClangFormat{
				Path: "/nonexistent/path/to/clang-format",
			},
		},
	}
	err := Format(t.Context(), cfg, &config.Library{Output: dir})
	if err == nil {
		t.Fatal("expected error when clang-format is missing, got nil")
	}
	if !strings.Contains(err.Error(), "clang-format required but not found") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestFormat_FormatsCppFiles(t *testing.T) {
	testhelper.RequireCommand(t, "clang-format")
	dir := t.TempDir()
	unformatted := "int   foo(  int  x ) { return   x ; }\n"
	headerPath := filepath.Join(dir, "foo.h")
	if err := os.WriteFile(headerPath, []byte(unformatted), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := Format(t.Context(), nil, &config.Library{Output: dir}); err != nil {
		t.Fatalf("Format failed: %v", err)
	}

	formatted, err := os.ReadFile(headerPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(formatted) == unformatted {
		t.Errorf("expected file to be formatted by clang-format, but content is unchanged:\n%s", string(formatted))
	}
}
