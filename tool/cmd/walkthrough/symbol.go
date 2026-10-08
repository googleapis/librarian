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

package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
)

// findSymbol returns the 0-based first and last line of a top-level Go
// declaration. symbol is a function, type, var or const name, or
// "Type.Method" for a method; a pointer receiver matches too.
func findSymbol(path, symbol string) (start, end int, err error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, nil, 0)
	if err != nil {
		return 0, 0, err
	}
	recv, name, _ := strings.Cut(symbol, ".")
	if name == "" {
		recv, name = "", recv
	}
	var found ast.Node
	for _, decl := range f.Decls {
		switch d := decl.(type) {
		case *ast.FuncDecl:
			if d.Name.Name == name && receiverName(d) == recv {
				found = d
			}
		case *ast.GenDecl:
			if recv != "" {
				continue
			}
			for _, spec := range d.Specs {
				if specNamed(spec, name) {
					found = spec
					if len(d.Specs) == 1 {
						found = d
					}
				}
			}
		}
		if found != nil {
			return fset.Position(found.Pos()).Line - 1, fset.Position(found.End()).Line - 1, nil
		}
	}
	return 0, 0, fmt.Errorf("symbol %q not found", symbol)
}

func receiverName(d *ast.FuncDecl) string {
	if d.Recv == nil || len(d.Recv.List) == 0 {
		return ""
	}
	t := d.Recv.List[0].Type
	if star, ok := t.(*ast.StarExpr); ok {
		t = star.X
	}
	if idx, ok := t.(*ast.IndexExpr); ok {
		t = idx.X
	}
	if id, ok := t.(*ast.Ident); ok {
		return id.Name
	}
	return ""
}

func specNamed(spec ast.Spec, name string) bool {
	switch s := spec.(type) {
	case *ast.TypeSpec:
		return s.Name.Name == name
	case *ast.ValueSpec:
		for _, n := range s.Names {
			if n.Name == name {
				return true
			}
		}
	}
	return false
}
