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

package postprocessing

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
)

func TestApply(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	createFiles(t, dir, map[string]string{
		"Foo.java":      "package com.example;\n\npublic class Foo {\n\tpublic void oldFunc() {}\n}\n",
		"to_delete.txt": "delete me",
	})
	cfg := &config.Postprocess{
		CopyFile: []config.CopyConfig{
			{Src: "Foo.java", Dst: "CopiedFoo.java"},
		},
		RemoveFile: []string{"to_delete.txt"},
		Replace: []config.ReplaceConfig{
			{Path: "Foo.java", Original: "oldFunc", Replacement: "newFunc"},
		},
		ReplaceRegex: []config.ReplaceRegexConfig{
			{Path: "Foo.java", Pattern: `public class (\w+)`, Replacement: "public class Bar"},
		},
		MethodOperations: []config.MethodOperation{
			{Path: "Foo.java", Action: "delete", FuncName: "public void newFunc()"},
		},
	}
	if err := Apply(dir, cfg); err != nil {
		t.Fatal(err)
	}
	wantFiles := map[string]string{
		"CopiedFoo.java": "package com.example;\n\npublic class Foo {\n\tpublic void oldFunc() {}\n}\n",
		"Foo.java":       "package com.example;\n\npublic class Bar {\n}\n",
	}
	if diff := cmp.Diff(wantFiles, readDirFiles(t, dir)); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestApply_NilOrEmptyConfig(t *testing.T) {
	for _, test := range []struct {
		name string
		cfg  *config.Postprocess
	}{
		{
			name: "nil config",
			cfg:  nil,
		},
		{
			name: "empty config",
			cfg:  &config.Postprocess{},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			dir := t.TempDir()
			if err := Apply(dir, test.cfg); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestApply_Error(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name    string
		files   map[string]string
		cfg     *config.Postprocess
		wantErr error
	}{
		{
			name: "copy fails - nonexistent source",
			cfg: &config.Postprocess{
				CopyFile: []config.CopyConfig{
					{Src: "nonexistent.txt", Dst: "dst.txt"},
				},
			},
			wantErr: fs.ErrNotExist,
		},
		{
			name: "remove fails - pattern matches no files",
			cfg: &config.Postprocess{
				RemoveFile: []string{"nonexistent/*.java"},
			},
			wantErr: fs.ErrNotExist,
		},
		{
			name:  "replace fails - text not found",
			files: map[string]string{"file.txt": "hello"},
			cfg: &config.Postprocess{
				Replace: []config.ReplaceConfig{
					{Path: "file.txt", Original: "missing", Replacement: "world"},
				},
			},
			wantErr: errTextNotFound,
		},
		{
			name:  "replace regex fails - pattern not matched",
			files: map[string]string{"file.txt": "hello"},
			cfg: &config.Postprocess{
				ReplaceRegex: []config.ReplaceRegexConfig{
					{Path: "file.txt", Pattern: `\d+`, Replacement: "123"},
				},
			},
			wantErr: errTextNotFound,
		},
		{
			name:  "method operation fails - unsupported action",
			files: map[string]string{"Test.java": "class Test {}"},
			cfg: &config.Postprocess{
				MethodOperations: []config.MethodOperation{
					{Path: "Test.java", Action: "invalid_action", FuncName: "void foo()"},
				},
			},
			wantErr: errUnsupportedMethodAction,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			createFiles(t, dir, test.files)
			gotErr := Apply(dir, test.cfg)
			if !errors.Is(gotErr, test.wantErr) {
				t.Errorf("Apply() error = %v, wantErr %v", gotErr, test.wantErr)
			}
		})
	}
}

func TestReplace(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	content := "Hello World"
	original := "World"
	replacement := "Go"
	want := "Hello Go"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Replace(path, original, replacement); err != nil {
		t.Fatal(err)
	}
	gotBytes, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	got := string(gotBytes)
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestReplaceRegex(t *testing.T) {
	for _, test := range []struct {
		name        string
		content     string
		pattern     string
		replacement string
		want        string
	}{
		{
			name:        "simple replacement",
			content:     "Hello World",
			pattern:     "World",
			replacement: "Go",
			want:        "Hello Go",
		},
		{
			name:        "regex replacement",
			content:     "Hello 123 World",
			pattern:     `\d+`,
			replacement: "Number",
			want:        "Hello Number World",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "test.txt")
			if err := os.WriteFile(path, []byte(test.content), 0o644); err != nil {
				t.Fatal(err)
			}
			if err := ReplaceRegex(path, test.pattern, test.replacement); err != nil {
				t.Fatal(err)
			}
			gotBytes, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			got := string(gotBytes)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestReplace_Error(t *testing.T) {
	for _, test := range []struct {
		name        string
		content     string
		original    string
		replacement string
		wantErr     error
	}{
		{
			name:        "file does not exist",
			original:    "old",
			replacement: "new",
			wantErr:     fs.ErrNotExist,
		},
		{
			name:        "text not found",
			content:     "Hello World",
			original:    "Apple",
			replacement: "Go",
			wantErr:     errTextNotFound,
		},
		{
			name:        "empty original string",
			content:     "Hello World",
			original:    "",
			replacement: "Go",
			wantErr:     errEmptyOriginal,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "nonexistent.txt")
			if test.content != "" {
				path = filepath.Join(dir, "test.txt")
				if err := os.WriteFile(path, []byte(test.content), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			err := Replace(path, test.original, test.replacement)
			if !errors.Is(err, test.wantErr) {
				t.Errorf("Replace() returned unexpected error: got %v, want %v", err, test.wantErr)
			}
		})
	}
}

func TestReplaceRegex_Error(t *testing.T) {
	for _, test := range []struct {
		name        string
		content     string
		pattern     string
		replacement string
		wantErr     error
	}{
		{
			name:        "file does not exist",
			pattern:     "old",
			replacement: "new",
			wantErr:     fs.ErrNotExist,
		},
		{
			name:        "pattern not found",
			content:     "Hello World",
			pattern:     `\d+`,
			replacement: "Number",
			wantErr:     errTextNotFound,
		},
		{
			name:        "empty pattern string",
			content:     "Hello World",
			pattern:     "",
			replacement: "Number",
			wantErr:     errEmptyPattern,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "nonexistent.txt")
			if test.content != "" {
				path = filepath.Join(dir, "test.txt")
				if err := os.WriteFile(path, []byte(test.content), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			err := ReplaceRegex(path, test.pattern, test.replacement)
			if !errors.Is(err, test.wantErr) {
				t.Errorf("ReplaceRegex() returned unexpected error: got %v, want %v", err, test.wantErr)
			}
		})
	}
}

func TestApplyToFiles(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name    string
		files   map[string]string
		pattern string
	}{
		{
			name:    "exact file success",
			files:   map[string]string{"foo.txt": "hello"},
			pattern: "foo.txt",
		},
		{
			name:    "glob pattern success",
			files:   map[string]string{"a.java": "match", "b.java": "match"},
			pattern: "*.java",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			createFiles(t, dir, test.files)
			if err := applyToFiles(dir, test.pattern, func(string) error { return nil }); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestApplyToFiles_Error(t *testing.T) {
	for _, test := range []struct {
		name    string
		files   map[string]string
		pattern string
		action  func(string) error
		wantErr error
	}{
		{
			name:    "action fails on glob match",
			files:   map[string]string{"a.java": "match", "b.java": "nomatch"},
			pattern: "*.java",
			action: func(p string) error {
				if strings.HasSuffix(p, "b.java") {
					return errTextNotFound
				}
				return nil
			},
			wantErr: errTextNotFound,
		},
		{
			name:    "action fails on exact file",
			files:   map[string]string{"foo.txt": "nomatch"},
			pattern: "foo.txt",
			action:  func(string) error { return errTextNotFound },
			wantErr: errTextNotFound,
		},
		{
			name:    "zero files match pattern",
			files:   map[string]string{"other.txt": "hello"},
			pattern: "*.java",
			action:  func(string) error { return nil },
			wantErr: fs.ErrNotExist,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			dir := t.TempDir()
			createFiles(t, dir, test.files)
			err := applyToFiles(dir, test.pattern, test.action)
			if !errors.Is(err, test.wantErr) {
				t.Errorf("applyToFiles() error = %v, want %v", err, test.wantErr)
			}
		})
	}
}

func TestRemoveFiles(t *testing.T) {
	for _, test := range []struct {
		name      string
		files     map[string]string
		patterns  []string
		wantFiles map[string]string
	}{
		{
			name:      "single glob pattern",
			files:     map[string]string{"A.java": "java content", "B.txt": "txt content"},
			patterns:  []string{"*.java"},
			wantFiles: map[string]string{"B.txt": "txt content"},
		},
		{
			name:      "exact filename",
			files:     map[string]string{"A.java": "java content", "B.txt": "txt content"},
			patterns:  []string{"A.java"},
			wantFiles: map[string]string{"B.txt": "txt content"},
		},
		{
			name:      "multiple glob patterns",
			files:     map[string]string{"A.java": "java content", "B.txt": "txt content", "C.md": "md content"},
			patterns:  []string{"*.java", "*.txt"},
			wantFiles: map[string]string{"C.md": "md content"},
		},
		{
			name:      "nested directory file removal",
			files:     map[string]string{"src/A.java": "java", "src/B.txt": "txt", "docs/C.html": "html"},
			patterns:  []string{"src/*.java"},
			wantFiles: map[string]string{"src/B.txt": "txt", "docs/C.html": "html"},
		},
		{
			name:      "directory file deletion",
			files:     map[string]string{"src/A.java": "java", "src/B.txt": "txt"},
			patterns:  []string{"src/*"},
			wantFiles: map[string]string{},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			dir := t.TempDir()
			createFiles(t, dir, test.files)
			if err := RemoveFiles(dir, test.patterns); err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(test.wantFiles, readDirFiles(t, dir)); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestRemoveFiles_Error(t *testing.T) {
	for _, test := range []struct {
		name     string
		files    map[string]string
		patterns []string
		wantErr  error
	}{
		{
			name:     "zero files match pattern",
			patterns: []string{"nonexistent/*.java"},
			wantErr:  fs.ErrNotExist,
		},
		{
			name:     "remove non-empty directory",
			files:    map[string]string{"targetDir/file.txt": "data"},
			patterns: []string{"targetDir"},
			wantErr:  syscall.ENOTEMPTY,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			dir := t.TempDir()
			createFiles(t, dir, test.files)
			err := RemoveFiles(dir, test.patterns)
			if !errors.Is(err, test.wantErr) {
				t.Errorf("RemoveFiles() error = %v, want %v", err, test.wantErr)
			}
		})
	}
}

func createFiles(t *testing.T, dir string, files map[string]string) {
	t.Helper()
	for relPath, content := range files {
		absPath := filepath.Join(dir, relPath)
		if err := os.MkdirAll(filepath.Dir(absPath), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(absPath, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func readDirFiles(t *testing.T, dir string) map[string]string {
	t.Helper()
	gotFiles := make(map[string]string)
	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			t.Fatal(err)
		}
		b, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		gotFiles[filepath.ToSlash(rel)] = string(b)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return gotFiles
}

func TestCopyFiles(t *testing.T) {
	for _, test := range []struct {
		name      string
		files     map[string]string
		configs   []config.CopyConfig
		wantFiles map[string]string
	}{
		{
			name:  "single file copy",
			files: map[string]string{"src.txt": "hello"},
			configs: []config.CopyConfig{
				{Src: "src.txt", Dst: "dst.txt"},
			},
			wantFiles: map[string]string{"src.txt": "hello", "dst.txt": "hello"},
		},
		{
			name:  "multiple copies of same source",
			files: map[string]string{"src.txt": "hello"},
			configs: []config.CopyConfig{
				{Src: "src.txt", Dst: "copied1.txt"},
				{Src: "src.txt", Dst: "copied2.txt"},
			},
			wantFiles: map[string]string{"src.txt": "hello", "copied1.txt": "hello", "copied2.txt": "hello"},
		},
		{
			name:  "nested directory copy",
			files: map[string]string{"sub/src.txt": "nested content"},
			configs: []config.CopyConfig{
				{Src: "sub/src.txt", Dst: "out/dst.txt"},
			},
			wantFiles: map[string]string{"sub/src.txt": "nested content", "out/dst.txt": "nested content"},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			dir := t.TempDir()
			createFiles(t, dir, test.files)
			if err := CopyFiles(dir, test.configs); err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(test.wantFiles, readDirFiles(t, dir)); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestCopyFiles_Error(t *testing.T) {
	for _, test := range []struct {
		name    string
		files   map[string]string
		configs []config.CopyConfig
		wantErr error
	}{
		{
			name:  "nonexistent source file",
			files: map[string]string{},
			configs: []config.CopyConfig{
				{Src: "nonexistent.txt", Dst: "dst.txt"},
			},
			wantErr: fs.ErrNotExist,
		},
		{
			name:  "same source and destination",
			files: map[string]string{"foo.txt": "hello"},
			configs: []config.CopyConfig{
				{Src: "foo.txt", Dst: "foo.txt"},
			},
			wantErr: errSameSourceAndDestination,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			dir := t.TempDir()
			createFiles(t, dir, test.files)
			err := CopyFiles(dir, test.configs)
			if !errors.Is(err, test.wantErr) {
				t.Errorf("CopyFiles() error = %v, want %v", err, test.wantErr)
			}
		})
	}
}

func TestReplaceAll(t *testing.T) {
	for _, test := range []struct {
		name      string
		files     map[string]string
		replaces  []config.ReplaceConfig
		wantFiles map[string]string
	}{
		{
			name:  "single file replacement",
			files: map[string]string{"Test.java": "old text"},
			replaces: []config.ReplaceConfig{
				{Path: "*.java", Original: "old", Replacement: "new"},
			},
			wantFiles: map[string]string{"Test.java": "new text"},
		},
		{
			name: "multiple files across subdirectories and untouched files",
			files: map[string]string{
				"src/A.java":     "package foo; class A {}",
				"sub/B.java":     "package foo; class B {}",
				"doc/readme.txt": "package foo description",
			},
			replaces: []config.ReplaceConfig{
				{Path: "*/*.java", Original: "package foo;", Replacement: "package bar;"},
			},
			wantFiles: map[string]string{
				"src/A.java":     "package bar; class A {}",
				"sub/B.java":     "package bar; class B {}",
				"doc/readme.txt": "package foo description",
			},
		},
		{
			name:  "sequential replacement configs",
			files: map[string]string{"Test.java": "alpha beta"},
			replaces: []config.ReplaceConfig{
				{Path: "*.java", Original: "alpha", Replacement: "gamma"},
				{Path: "*.java", Original: "beta", Replacement: "delta"},
			},
			wantFiles: map[string]string{"Test.java": "gamma delta"},
		},
		{
			name: "skips directories matched by glob",
			files: map[string]string{
				"subdir/A.java": "old text",
				"B.java":        "old text",
			},
			replaces: []config.ReplaceConfig{
				{Path: "*", Original: "old", Replacement: "new"},
			},
			wantFiles: map[string]string{
				"subdir/A.java": "old text",
				"B.java":        "new text",
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			dir := t.TempDir()
			createFiles(t, dir, test.files)
			if err := ReplaceAll(dir, test.replaces); err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(test.wantFiles, readDirFiles(t, dir)); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestReplaceAll_Error(t *testing.T) {
	for _, test := range []struct {
		name     string
		files    map[string]string
		replaces []config.ReplaceConfig
		wantErr  error
	}{
		{
			name:     "no files match pattern",
			files:    map[string]string{"foo.txt": "hello"},
			replaces: []config.ReplaceConfig{{Path: "*.java", Original: "old", Replacement: "new"}},
			wantErr:  fs.ErrNotExist,
		},
		{
			name:     "text not found in file",
			files:    map[string]string{"Test.java": "hello world"},
			replaces: []config.ReplaceConfig{{Path: "*.java", Original: "missing", Replacement: "new"}},
			wantErr:  errTextNotFound,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			dir := t.TempDir()
			createFiles(t, dir, test.files)
			err := ReplaceAll(dir, test.replaces)
			if !errors.Is(err, test.wantErr) {
				t.Errorf("ReplaceAll() error = %v, want %v", err, test.wantErr)
			}
		})
	}
}

func TestReplaceRegexAll(t *testing.T) {
	for _, test := range []struct {
		name      string
		files     map[string]string
		regexes   []config.ReplaceRegexConfig
		wantFiles map[string]string
	}{
		{
			name:  "single file regex replacement",
			files: map[string]string{"Test.java": "version 1.2.3"},
			regexes: []config.ReplaceRegexConfig{
				{Path: "*.java", Pattern: `version \d+\.\d+\.\d+`, Replacement: "version 2.0.0"},
			},
			wantFiles: map[string]string{"Test.java": "version 2.0.0"},
		},
		{
			name: "multiline replacement and capture groups",
			files: map[string]string{
				"src/A.java": "import com.old.Foo;\nimport com.old.Bar;",
				"sub/B.txt":  "import com.old.Baz;",
			},
			regexes: []config.ReplaceRegexConfig{
				{Path: "*/*.java", Pattern: `import com\.old\.(\w+);`, Replacement: "import com.new.${1};"},
			},
			wantFiles: map[string]string{
				"src/A.java": "import com.new.Foo;\nimport com.new.Bar;",
				"sub/B.txt":  "import com.old.Baz;",
			},
		},
		{
			name: "skips directories matched by glob",
			files: map[string]string{
				"subdir/A.java": "version 1.2.3",
				"B.java":        "version 1.2.3",
			},
			regexes: []config.ReplaceRegexConfig{
				{Path: "*", Pattern: `version \d+\.\d+\.\d+`, Replacement: "version 2.0.0"},
			},
			wantFiles: map[string]string{
				"subdir/A.java": "version 1.2.3",
				"B.java":        "version 2.0.0",
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			dir := t.TempDir()
			createFiles(t, dir, test.files)
			if err := ReplaceRegexAll(dir, test.regexes); err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(test.wantFiles, readDirFiles(t, dir)); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestReplaceRegexAll_Error(t *testing.T) {
	for _, test := range []struct {
		name    string
		files   map[string]string
		regexes []config.ReplaceRegexConfig
		wantErr error
	}{
		{
			name:    "no files match pattern",
			files:   map[string]string{"foo.txt": "hello"},
			regexes: []config.ReplaceRegexConfig{{Path: "*.java", Pattern: "old", Replacement: "new"}},
			wantErr: fs.ErrNotExist,
		},
		{
			name:    "pattern not found in file",
			files:   map[string]string{"Test.java": "version 1.0.0"},
			regexes: []config.ReplaceRegexConfig{{Path: "*.java", Pattern: `\d{3}`, Replacement: "2.0.0"}},
			wantErr: errTextNotFound,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			dir := t.TempDir()
			createFiles(t, dir, test.files)
			err := ReplaceRegexAll(dir, test.regexes)
			if !errors.Is(err, test.wantErr) {
				t.Errorf("ReplaceRegexAll() error = %v, want %v", err, test.wantErr)
			}
		})
	}
}

func TestApplyMethodOperations(t *testing.T) {
	for _, test := range []struct {
		name      string
		files     map[string]string
		ops       []config.MethodOperation
		wantFiles map[string]string
	}{
		{
			name: "single file sequential operations",
			files: map[string]string{
				"Test.java": "package com.example;\n\npublic class Test {\n\tpublic void toDelete() {}\n\tpublic void newFunc() {}\n}",
			},
			ops: []config.MethodOperation{
				{Path: "*.java", Action: "delete", FuncName: "public void toDelete()"},
				{Path: "*.java", Action: "duplicate", FuncName: "public void newFunc()", NewName: "newFuncCopy"},
				{Path: "*.java", Action: "deprecate", FuncName: "public void newFuncCopy()", DeprecationMessage: "Use newFunc instead."},
			},
			wantFiles: map[string]string{
				"Test.java": "package com.example;\n\npublic class Test {\n\tpublic void newFunc() {}\n\n\t/**\n\t * @deprecated Use newFunc instead.\n\t */\n\t@Deprecated\n\tpublic void newFuncCopy() {}\n}",
			},
		},
		{
			name: "batch execution across subdirectories ignoring non-matching files",
			files: map[string]string{
				"src/A.java":     "public class A {\n\tpublic void removeMe() {}\n}",
				"sub/B.java":     "public class B {\n\tpublic void removeMe() {}\n}",
				"doc/readme.txt": "public class C {\n\tpublic void removeMe() {}\n}",
			},
			ops: []config.MethodOperation{
				{Path: "*/*.java", Action: "delete", FuncName: "public void removeMe()"},
			},
			wantFiles: map[string]string{
				"src/A.java":     "public class A {\n}",
				"sub/B.java":     "public class B {\n}",
				"doc/readme.txt": "public class C {\n\tpublic void removeMe() {}\n}",
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			dir := t.TempDir()
			createFiles(t, dir, test.files)
			if err := ApplyMethodOperations(dir, test.ops); err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(test.wantFiles, readDirFiles(t, dir)); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestApplyMethodOperations_Error(t *testing.T) {
	for _, test := range []struct {
		name    string
		files   map[string]string
		ops     []config.MethodOperation
		wantErr error
	}{
		{
			name:    "no files match pattern",
			files:   map[string]string{"foo.txt": "hello"},
			ops:     []config.MethodOperation{{Path: "*.java", Action: "delete", FuncName: "public void foo()"}},
			wantErr: fs.ErrNotExist,
		},
		{
			name:    "method not found in file",
			files:   map[string]string{"Test.java": "public class Test {}"},
			ops:     []config.MethodOperation{{Path: "*.java", Action: "delete", FuncName: "public void missing()"}},
			wantErr: errMethodNotFound,
		},
		{
			name:    "unsupported action",
			files:   map[string]string{"Test.java": "public class Test {}"},
			ops:     []config.MethodOperation{{Path: "*.java", Action: "invalid_action", FuncName: "public void foo()"}},
			wantErr: errUnsupportedMethodAction,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			dir := t.TempDir()
			createFiles(t, dir, test.files)
			err := ApplyMethodOperations(dir, test.ops)
			if !errors.Is(err, test.wantErr) {
				t.Errorf("ApplyMethodOperations() error = %v, want %v", err, test.wantErr)
			}
		})
	}
}
