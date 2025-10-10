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
	"bytes"
	"errors"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"
)

var noHeaderRequiredFiles = []string{
	".gcloudignore",
	".github/CODEOWNERS",
	".gitignore",
	"Dockerfile",
	"LICENSE",
	"languagecontainer/go.mod",
	"languagecontainer/go.sum",
	"coverage.out",
	"go.mod",
	"go.sum",
	"librarian",
	"renovate.json",
}

var ignoredExts = map[string]bool{
	".excalidraw": true,
	".md":         true,
	".yml":        true,
	".yaml":       true,
	".txt":        true,
	".webp":       true,
}

var ignoredDirs = []string{
	".git",
	".idea",
	".vscode",
	"infra",
	"testdata",
}

// expectedHeader defines the regex for the required copyright header.
const expectedHeader = `// Copyright 202\d Google LLC
//
// Licensed under the Apache License, Version 2.0 \(the "License"\);
// you may not use this file except in compliance with the License\.
// You may obtain a copy of the License at`

var headerRegex = regexp.MustCompile("(?s)" + expectedHeader)

func TestHeaders(t *testing.T) {
	sfs := os.DirFS(".")
	err := fs.WalkDir(sfs, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip ignored files and directories.
		if d.IsDir() {
			if slices.Contains(ignoredDirs, d.Name()) {
				return fs.SkipDir
			}
			return nil
		}
		if slices.Contains(noHeaderRequiredFiles, path) || ignoredExts[filepath.Ext(path)] {
			return nil
		}

		f, err := sfs.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()

		ok, err := hasValidHeader(path, f)
		if err != nil {
			return err
		}
		if !ok {
			t.Errorf("%q: invalid header", path)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func hasValidHeader(path string, r io.Reader) (bool, error) {
	allBytes, err := io.ReadAll(r)
	if err != nil {
		return false, err
	}

	// If the file is a shell script and starts with a shebang, skip that line.
	if strings.HasSuffix(path, ".sh") && bytes.HasPrefix(allBytes, []byte("#!")) {
		// Find the index of the first newline to get the rest of the content.
		if i := bytes.IndexByte(allBytes, '\n'); i > -1 {
			allBytes = allBytes[i+1:]
		}
	}

	// If the file is a mustache template, the license is expected to be
	// wrapped as:
	// {{!
	// Copyright 2024 Google LLC
	// ...
	// }}
	if strings.HasSuffix(path, ".mustache") {
		if !bytes.HasPrefix(allBytes, []byte("{{!")) {
			return false, nil
		}
		end := bytes.Index(allBytes, []byte("}}"))
		if end == -1 {
			return false, nil
		}
		headerContent := allBytes[len("{{!"):end]
		headerContent = bytes.TrimPrefix(headerContent, []byte("\n"))
		var builder strings.Builder
		lines := strings.Split(string(headerContent), "\n")
		for i, line := range lines {
			builder.WriteString("//")
			if len(line) > 0 {
				builder.WriteString(" ")
			}
			builder.WriteString(line)
			if i < len(lines)-1 {
				builder.WriteString("\n")
			}
		}
		return headerRegex.MatchString(builder.String()), nil
	}

	return headerRegex.Match(allBytes), nil
}

func TestGolangCILint(t *testing.T) {
	rungo(t, "run", "github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest", "run")
}

func TestGoImports(t *testing.T) {
	cmd := exec.Command("go", "run", "golang.org/x/tools/cmd/goimports@latest", "-d", ".")
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	if err := cmd.Run(); err != nil {
		t.Fatalf("goimports failed to run: %v\nOutput:\n%s", err, out.String())
	}
	if out.Len() > 0 {
		t.Errorf("goimports found unformatted files:\n%s", out.String())
	}
}

func TestGoModTidy(t *testing.T) {
	rungo(t, "mod", "tidy", "-diff")
}

func TestGovulncheck(t *testing.T) {
	rungo(t, "run", "golang.org/x/vuln/cmd/govulncheck@latest", "./...")
}

func TestGodocLint(t *testing.T) {
	rungo(t, "run", "github.com/godoc-lint/godoc-lint/cmd/godoclint@v0.3.0",
		// TODO(https://github.com/googleapis/librarian/issues/1510): fix test
		"-exclude", "internal/sidekick",
		"./...")
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
