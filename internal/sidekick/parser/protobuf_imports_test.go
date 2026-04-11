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

package parser

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"sort"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestProtobufImportsSync(t *testing.T) {
	ossNames := collectNames(t, "protobuf_imports_oss.go")
	g3Names := collectNames(t, "protobuf_imports_google3.go")

	sort.Strings(ossNames)
	sort.Strings(g3Names)

	if diff := cmp.Diff(ossNames, g3Names); diff != "" {
		t.Errorf("protobuf_imports_oss.go and protobuf_imports_google3.go are out of sync (-oss +g3):\n%s", diff)
	}
}

func collectNames(t *testing.T, filename string) []string {
	t.Helper()
	src, err := os.ReadFile(filename)
	if err != nil {
		t.Fatal(err)
	}

	fset := token.NewFileSet()
	// Use parser.ParseFile to get the AST. We don't need to specify the environment
	// build tags because we are reading the file content directly.
	file, err := parser.ParseFile(fset, filename, src, 0)
	if err != nil {
		t.Fatal(err)
	}

	var names []string
	for _, decl := range file.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok {
			continue
		}
		for _, spec := range genDecl.Specs {
			switch s := spec.(type) {
			case *ast.TypeSpec:
				names = append(names, s.Name.Name)
			case *ast.ValueSpec:
				for _, id := range s.Names {
					names = append(names, id.Name)
				}
			}
		}
	}
	return names
}
