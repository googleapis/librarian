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

//go:build configdocgen

package main

import (
	"go/ast"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExtractYamlName(t *testing.T) {
	for _, test := range []struct {
		name string
		tag  *ast.BasicLit
		want string
	}{
		{
			name: "valid yaml tag",
			tag:  &ast.BasicLit{Value: "`yaml:\"name\"`"},
			want: "name",
		},
		{
			name: "yaml tag with options",
			tag:  &ast.BasicLit{Value: "`yaml:\"name,omitempty\"`"},
			want: "name",
		},
		{
			name: "no yaml tag",
			tag:  &ast.BasicLit{Value: "`json:\"name\"`"},
			want: "",
		},
		{
			name: "empty tag",
			tag:  &ast.BasicLit{Value: "``"},
			want: "",
		},
		{
			name: "nil tag",
			tag:  nil,
			want: "",
		},
		{
			name: "ignored field",
			tag:  &ast.BasicLit{Value: "`yaml:\"-\"`"},
			want: "-",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := extractYamlName(test.tag)
			if got != test.want {
				t.Errorf("extractYamlName() = %q, want %q", got, test.want)
			}
		})
	}
}

func TestGetTypeName(t *testing.T) {
	for _, test := range []struct {
		name string
		expr ast.Expr
		want string
	}{
		{
			name: "ident",
			expr: &ast.Ident{Name: "string"},
			want: "string",
		},
		{
			name: "pointer",
			expr: &ast.StarExpr{X: &ast.Ident{Name: "Config"}},
			want: "*Config",
		},
		{
			name: "slice",
			expr: &ast.ArrayType{Elt: &ast.Ident{Name: "int"}},
			want: "[]int",
		},
		{
			name: "map",
			expr: &ast.MapType{
				Key:   &ast.Ident{Name: "string"},
				Value: &ast.Ident{Name: "bool"},
			},
			want: "map[string]bool",
		},
		{
			name: "selector",
			expr: &ast.SelectorExpr{
				X:   &ast.Ident{Name: "time"},
				Sel: &ast.Ident{Name: "Time"},
			},
			want: "time.Time",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := getTypeName(test.expr)
			if got != test.want {
				t.Errorf("getTypeName() = %q, want %q", got, test.want)
			}
		})
	}
}

func TestFormatType(t *testing.T) {
	allStructs := map[string]*ast.StructType{
		"Repo": {},
	}

	for _, test := range []struct {
		name     string
		typeName string
		want     string
	}{
		{
			name:     "basic type",
			typeName: "string",
			want:     "string",
		},
		{
			name:     "internal struct",
			typeName: "Repo",
			want:     "[Repo](#repo-configuration)",
		},
		{
			name:     "pointer to basic",
			typeName: "*int",
			want:     "int (optional)",
		},
		{
			name:     "pointer to internal",
			typeName: "*Repo",
			want:     "[Repo](#repo-configuration) (optional)",
		},
		{
			name:     "slice of basic",
			typeName: "[]string",
			want:     "list of string",
		},
		{
			name:     "slice of internal",
			typeName: "[]Repo",
			want:     "list of [Repo](#repo-configuration)",
		},
		{
			name:     "slice of pointers to internal",
			typeName: "[]*Repo",
			want:     "list of [Repo](#repo-configuration) (optional)",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := formatType(test.typeName, allStructs)
			if got != test.want {
				t.Errorf("formatType(%q) = %q, want %q", test.typeName, got, test.want)
			}
		})
	}
}

func TestCleanDoc(t *testing.T) {
	for _, test := range []struct {
		name string
		doc  string
		want string
	}{
		{
			name: "simple doc",
			doc:  "This is a comment.",
			want: "This is a comment.",
		},
		{
			name: "multiline doc",
			doc:  "This is\na multiline\ncomment.",
			want: "This is a multiline comment.",
		},
		{
			name: "doc with extra spaces",
			doc:  " This  is   a    comment. ",
			want: "This is a comment.",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := cleanDoc(test.doc)
			if got != test.want {
				t.Errorf("cleanDoc() = %q, want %q", got, test.want)
			}
		})
	}
}

func TestGenerate(t *testing.T) {
	dir := t.TempDir()

	configContent := `
package config
// Config doc
type Config struct {
	A string ` + "`" + `yaml:"a"` + "`" + `
}
// Second doc
type Second struct {
	B int ` + "`" + `yaml:"b"` + "`" + `
}
`
	otherContent := `
package config
// Other doc
type Other struct {
	C bool ` + "`" + `yaml:"c"` + "`" + `
}
// Alpha doc
type Alpha struct {
	D string ` + "`" + `yaml:"d"` + "`" + `
}
`

	if err := os.WriteFile(filepath.Join(dir, "config.go"), []byte(configContent), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "other.go"), []byte(otherContent), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module config\n"), 0644); err != nil {
		t.Fatal(err)
	}

	pkg, err := loadPackage(dir)
	if err != nil {
		t.Fatal(err)
	}

	var buf strings.Builder
	d, err := newDocData(pkg)
	if err != nil {
		t.Fatal(err)
	}
	if err := d.generate(&buf); err != nil {
		t.Fatal(err)
	}
	got := buf.String()

	configIdx := strings.Index(got, "## Root Configuration")
	secondIdx := strings.Index(got, "## Second Configuration")
	alphaIdx := strings.Index(got, "## Alpha Configuration")
	otherIdx := strings.Index(got, "## Other Configuration")
	if configIdx == -1 || secondIdx == -1 || alphaIdx == -1 || otherIdx == -1 {
		t.Fatalf("missing objects in output: root_config=%d, second=%d, alpha=%d, other=%d", configIdx, secondIdx, alphaIdx, otherIdx)
	}
	if !(configIdx < secondIdx && secondIdx < alphaIdx && alphaIdx < otherIdx) {
		t.Errorf("incorrect order: root_config=%d, second=%d, alpha=%d, other=%d", configIdx, secondIdx, alphaIdx, otherIdx)
	}

	if !strings.Contains(got, "[Link to code](../config.go#L4)") {
		t.Error("output missing expected source link for Config")
	}
}
