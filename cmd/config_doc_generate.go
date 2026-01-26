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
	"flag"
	"fmt"
	"go/ast"
	"io"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"

	"golang.org/x/tools/go/packages"
)

var (
	inputDir   = flag.String("input", "internal/config", "Input directory containing config structs")
	outputFile = flag.String("output", "doc/config-schema.md", "Output file for documentation")
)

func main() {
	flag.Parse()
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	out, err := os.Create(*outputFile)
	if err != nil {
		return err
	}
	defer out.Close()

	return generate(out, *inputDir)
}

func generate(out io.Writer, dir string) error {
	cfg := &packages.Config{
		Mode:  packages.NeedSyntax | packages.NeedTypes | packages.NeedName | packages.NeedFiles | packages.NeedModule,
		Dir:   dir,
		Tests: false,
	}
	pkgs, err := packages.Load(cfg, ".")
	if err != nil {
		return err
	}
	if len(pkgs) == 0 {
		return fmt.Errorf("no packages found in %s", dir)
	}
	pkg := pkgs[0]
	if len(pkg.Errors) > 0 {
		return pkg.Errors[0]
	}

	structs := make(map[string]*ast.StructType)
	docs := make(map[string]string)
	sources := make(map[string]string)
	var configKeys []string
	var otherKeys []string

	root := "."
	if pkg.Module != nil {
		root = pkg.Module.Dir
	}

	for _, file := range pkg.Syntax {
		fileName := pkg.Fset.File(file.Pos()).Name()
		relPath, _ := filepath.Rel(root, fileName)
		isConfig := filepath.Base(fileName) == "config.go"

		ast.Inspect(file, func(n ast.Node) bool {
			ts, ok := n.(*ast.TypeSpec)
			if !ok {
				return true
			}
			st, ok := ts.Type.(*ast.StructType)
			if !ok {
				return true
			}

			name := ts.Name.Name
			if structs[name] != nil {
				return true // Already seen
			}

			structs[name] = st
			if ts.Doc != nil {
				docs[name] = cleanDoc(ts.Doc.Text())
			}

			line := pkg.Fset.Position(ts.Pos()).Line
			sources[name] = fmt.Sprintf("../%s#L%d", relPath, line)

			if isConfig {
				configKeys = append(configKeys, name)
			} else {
				otherKeys = append(otherKeys, name)
			}
			return true
		})
	}

	sort.Strings(otherKeys)

	fmt.Fprintln(out, "# librarian.yaml Schema")
	fmt.Fprintln(out)
	fmt.Fprintln(out, "This document describes the schema for the `librarian.yaml` file.")

	// Write Config objects first, then others.
	for _, k := range append(configKeys, otherKeys...) {
		writeStruct(out, k, structs[k], structs, docs, sources[k])
	}

	return nil
}

// writeStruct writes a Markdown representation of a Go struct to the provided writer.
// It generates a table of fields, including their YAML names, types, and descriptions.
func writeStruct(out io.Writer, name string, st *ast.StructType, allStructs map[string]*ast.StructType, docs map[string]string, sourceLink string) {
	title := name + " Configuration"
	if name == "Config" {
		title = "Root Configuration"
	}

	fmt.Fprintln(out)
	fmt.Fprintf(out, "## %s\n", title)
	fmt.Fprintln(out)
	if sourceLink != "" {
		fmt.Fprintf(out, "[Source](%s)\n\n", sourceLink)
	}
	if doc := docs[name]; doc != "" {
		fmt.Fprintf(out, "%s\n", doc)
		fmt.Fprintln(out)
	}

	fmt.Fprintln(out, "| Field | Type | Description |")
	fmt.Fprintln(out, "| :--- | :--- | :--- |")

	for _, field := range st.Fields.List {
		if len(field.Names) == 0 {
			// Embedded struct
			typeName := getTypeName(field.Type)
			fmt.Fprintf(out, "| (embedded) | %s | |\n", formatType(typeName, allStructs))
			continue
		}

		yamlName := extractYamlName(field.Tag)
		if yamlName == "" || yamlName == "-" {
			continue
		}

		typeName := getTypeName(field.Type)
		description := ""
		if field.Doc != nil {
			description = cleanDoc(field.Doc.Text())
		}

		fmt.Fprintf(out, "| `%s` | %s | %s |\n", yamlName, formatType(typeName, allStructs), description)
	}
}

func extractYamlName(tag *ast.BasicLit) string {
	if tag == nil {
		return ""
	}
	t := reflect.StructTag(strings.Trim(tag.Value, "`"))
	val := t.Get("yaml")
	if val == "" {
		return ""
	}
	return strings.Split(val, ",")[0]
}

func getTypeName(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		return "*" + getTypeName(t.X)
	case *ast.ArrayType:
		return "[]" + getTypeName(t.Elt)
	case *ast.MapType:
		return fmt.Sprintf("map[%s]%s", getTypeName(t.Key), getTypeName(t.Value))
	case *ast.SelectorExpr:
		return fmt.Sprintf("%s.%s", getTypeName(t.X), t.Sel.Name)
	default:
		return fmt.Sprintf("%T", expr)
	}
}

func formatType(typeName string, allStructs map[string]*ast.StructType) string {
	isSlice := strings.HasPrefix(typeName, "[]")
	cleanType := strings.TrimPrefix(typeName, "[]")
	isPointer := strings.HasPrefix(cleanType, "*")
	cleanType = strings.TrimPrefix(cleanType, "*")

	res := cleanType
	// If it's one of our structs, link it
	if _, ok := allStructs[cleanType]; ok {
		anchor := strings.ToLower(cleanType) + "-configuration"
		if cleanType == "Config" {
			anchor = "root-configuration"
		}
		res = fmt.Sprintf("[%s](#%s)", cleanType, anchor)
	}

	if isPointer {
		res = res + " (optional)"
	}
	if isSlice {
		res = "list of " + res
	}

	return res
}

func cleanDoc(doc string) string {
	return strings.Join(strings.Fields(doc), " ")
}
