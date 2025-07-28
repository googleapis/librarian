// Copyright 2024 Google LLC
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
	"bufio"
	"bytes"
	"errors"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"
)

var doubleSlashHeader = regexp.MustCompile(`^// Copyright 20\d\d Google LLC
//
// Licensed under the Apache License, Version 2\.0 \(the "License"\);`)

var shellHeader = regexp.MustCompile(`^#!/.*

# Copyright 20\d\d Google LLC
#
# Licensed under the Apache License, Version 2\.0 \(the "License"\);`)

var hashHeader = regexp.MustCompile(`^# Copyright 20\d\d Google LLC
#
# Licensed under the Apache License, Version 2\.0 \(the "License"\);`)

var noHeaderRequiredFiles = []string{".github/CODEOWNERS", "go.sum", "go.mod", ".gitignore", "LICENSE", "renovate.json", "coverage.out", "librarian"}

func TestHeaders(t *testing.T) {
	sfs := os.DirFS(".")
	err := fs.WalkDir(sfs, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if d.Name() == "testdata" || d.Name() == ".git" || d.Name() == ".vscode" || d.Name() == ".idea" {
				return fs.SkipDir
			}
			return nil
		}

		var requiredHeader *regexp.Regexp
		switch {
		case strings.HasSuffix(path, ".go") || strings.HasSuffix(path, ".proto"):
			requiredHeader = doubleSlashHeader
		case strings.HasSuffix(path, ".sh"):
			requiredHeader = shellHeader
		case strings.HasSuffix(path, ".yaml") || strings.HasSuffix(path, ".yml") || strings.HasPrefix(path, "Dockerfile"):
			requiredHeader = hashHeader
		case strings.HasSuffix(path, ".md"):
			return nil
		case slices.Contains(noHeaderRequiredFiles, path):
			return nil
		default:
			// Given the mixture of allow-lists and requirements, if there's a file which
			// isn't covered, we report an error.
			t.Errorf("%q: unknown header requirements", path)
			return nil
		}
		f, err := sfs.Open(path)
		if err != nil {
			return err
		}
		defer func() {
			if cerr := f.Close(); cerr != nil {
				t.Fatal(err)
			}
		}()
		if !requiredHeader.MatchReader(bufio.NewReader(f)) {
			t.Errorf("%q: incorrect header", path)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestGolangCILint(t *testing.T) {
	rungo(t, "run", "github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest", "run")
}

func TestGoFmt(t *testing.T) {
	cmd := exec.Command("gofmt", "-l", ".")
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	if err := cmd.Run(); err != nil {
		t.Fatalf("gofmt failed to run: %v\nOutput:\n%s", err, out.String())
	}
	if out.Len() > 0 {
		t.Errorf("gofmt found unformatted files:\n%s", out.String())
	}
}

func TestGoModTidy(t *testing.T) {
	rungo(t, "mod", "tidy", "-diff")
}

func TestGovulncheck(t *testing.T) {
	rungo(t, "run", "golang.org/x/vuln/cmd/govulncheck@latest", "./...")
}

func TestGodocLint(t *testing.T) {
	rungo(t, "run", "github.com/godoc-lint/godoc-lint/cmd/godoclint@latest",
		"-exclude", "cmd/librarian/main.go",
		"-exclude", "internal/statepb",
		"./...")
}

func TestCoverage(t *testing.T) {
	rungo(t, "test", "-coverprofile=coverage.out", "./internal/...", "./cmd/...")
}

func rungo(t *testing.T, args ...string) {
	t.Helper()

	cmd := exec.Command("go", args...)
	if output, err := cmd.CombinedOutput(); err != nil {
		if ee := (*exec.ExitError)(nil); errors.As(err, &ee) && len(ee.Stderr) > 0 {
			t.Fatalf("%v: %v\n%s", cmd, err, ee.Stderr)
		}
		t.Fatalf("%v: %v\n%s", cmd, err, output)
	}
}

func TestExportedSymbolsHaveDocs(t *testing.T) {
	err := filepath.WalkDir(".", func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") ||
			strings.HasSuffix(path, "_test.go") || strings.HasSuffix(path, ".pb.go") {
			return nil
		}

		fset := token.NewFileSet()
		node, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
		if err != nil {
			t.Errorf("failed to parse file %q: %v", path, err)
			return nil
		}

		// Visit every top-level declaration in the file.
		for _, decl := range node.Decls {
			gen, ok := decl.(*ast.GenDecl)
			if ok && (gen.Tok == token.TYPE || gen.Tok == token.VAR) {
				for _, spec := range gen.Specs {
					switch s := spec.(type) {
					case *ast.TypeSpec:
						checkDoc(t, s.Name, gen.Doc, path)
					case *ast.ValueSpec:
						for _, name := range s.Names {
							checkDoc(t, name, gen.Doc, path)
						}
					}
				}
			}
			if fn, ok := decl.(*ast.FuncDecl); ok {
				checkDoc(t, fn.Name, fn.Doc, path)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func checkDoc(t *testing.T, name *ast.Ident, doc *ast.CommentGroup, path string) {
	t.Helper()
	if !name.IsExported() {
		return
	}
	if doc == nil {
		t.Errorf("%s: %q is missing doc comment",
			path, name.Name)
	}
}
