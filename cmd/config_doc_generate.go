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
	"go/parser"
	"go/token"
	"log"
	"os"
	"reflect"
	"regexp"
	"sort"
	"strings"
)

var (
	inputDir     = flag.String("input", "internal/config", "Input directory containing config structs")
	outputFile   = flag.String("output", "doc/config-schema.md", "Output file for documentation")
	reWhitespace = regexp.MustCompile(`\s+`)
)

func main() {
	flag.Parse()
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, *inputDir, nil, parser.ParseComments)
	if err != nil {
		return err
	}

	pkg, ok := pkgs["config"]
	if !ok {
		return fmt.Errorf("package config not found in %s", *inputDir)
	}

	structs := make(map[string]*ast.StructType)
	docs := make(map[string]string)

	for _, file := range pkg.Files {
		ast.Inspect(file, func(n ast.Node) bool {
			ts, ok := n.(*ast.TypeSpec)
			if !ok {
				return true
			}
			st, ok := ts.Type.(*ast.StructType)
			if !ok {
				return true
			}
			structs[ts.Name.Name] = st
			if ts.Doc != nil {
				docs[ts.Name.Name] = cleanDoc(ts.Doc.Text())
			}
			return true
		})
	}

	out, err := os.Create(*outputFile)
	if err != nil {
		return err
	}
	defer out.Close()

	fmt.Fprintln(out, "# librarian.yaml Schema")
	fmt.Fprintln(out)
	fmt.Fprintln(out, "This document describes the schema for the `librarian.yaml` file.")

	// Start with Config struct
	if st, ok := structs["Config"]; ok {
		writeStruct(out, "Config", st, structs, docs)
	}

	// Write other referenced structs in alphabetical order
	keys := make([]string, 0, len(structs))
	for k := range structs {
		if k != "Config" {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)

	for _, k := range keys {
		writeStruct(out, k, structs[k], structs, docs)
	}

	return nil
}

func writeStruct(out *os.File, name string, st *ast.StructType, allStructs map[string]*ast.StructType, docs map[string]string) {
	fmt.Fprintln(out)
	fmt.Fprintf(out, "## %s Object\n", name)
	fmt.Fprintln(out)
	if doc := docs[name]; doc != "" {
		fmt.Fprintf(out, "%s\n", doc)
		fmt.Fprintln(out)
	}

	fmt.Fprintln(out, "| Field | Type | Description |")
	fmt.Fprintln(out, "| :--- | :--- | :--- |")

	for _, field := range st.Fields.List {
		if len(field.Names) == 0 {
			// Embedded struct
			if ident, ok := field.Type.(*ast.Ident); ok {
				fmt.Fprintf(out, "| (embedded) | [%s](#%s-object) | |\n", ident.Name, strings.ToLower(ident.Name))
			}
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
		res = fmt.Sprintf("[%s](#%s-object)", cleanType, strings.ToLower(cleanType))
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
	doc = strings.TrimSpace(doc)
	doc = strings.ReplaceAll(doc, "\n", " ")
	return reWhitespace.ReplaceAllString(doc, " ")
}
