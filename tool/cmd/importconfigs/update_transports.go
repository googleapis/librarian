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
	"context"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/googleapis/librarian/internal/config/bazel"
	"github.com/urfave/cli/v3"
)

var allLanguages = []string{"csharp", "go", "java", "nodejs", "php", "python", "ruby"}

func updateTransportsCommand() *cli.Command {
	return &cli.Command{
		Name:  "update-transports",
		Usage: "update transport values in internal/serviceconfig/api.go from BUILD.bazel files",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "googleapis",
				Usage:    "path to googleapis dir",
				Required: true,
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			googleapisDir := cmd.String("googleapis")
			return runUpdateTransports("internal/serviceconfig/api.go", googleapisDir)
		},
	}
}

func runUpdateTransports(apiGoPath, googleapisDir string) error {
	fset := token.NewFileSet()
	astFile, err := parser.ParseFile(fset, apiGoPath, nil, parser.ParseComments)
	if err != nil {
		return fmt.Errorf("failed to parse %s: %w", apiGoPath, err)
	}

	apisSlice := findAPIsSlice(astFile)
	if apisSlice == nil {
		return fmt.Errorf("could not find APIs variable in %s", apiGoPath)
	}

	for _, expr := range apisSlice.Elts {
		apiLit, ok := expr.(*ast.CompositeLit)
		if !ok {
			continue
		}
		path, transportsIdx := extractAPIInfo(apiLit)
		if path == "" {
			continue
		}
		transports := readTransports(googleapisDir, path)
		if len(transports) == 0 || (len(transports) == 1 && transports["all"] == "grpc+rest") {
			if transportsIdx != -1 {
				// Remove the Transports field if it exists and is now the default.
				apiLit.Elts = append(apiLit.Elts[:transportsIdx], apiLit.Elts[transportsIdx+1:]...)
			}
			continue
		}
		transportsKV := &ast.KeyValueExpr{
			Key:   ast.NewIdent("Transports"),
			Value: createTransportsExpr(transports),
		}
		if transportsIdx != -1 {
			apiLit.Elts[transportsIdx] = transportsKV
		} else {
			apiLit.Elts = append(apiLit.Elts, transportsKV)
		}
	}

	out, err := os.Create(apiGoPath)
	if err != nil {
		return fmt.Errorf("failed to create %s: %w", apiGoPath, err)
	}
	defer out.Close()
	return format.Node(out, fset, astFile)
}

func readTransports(googleapisDir, path string) map[string]string {
	buildPath := filepath.Join(googleapisDir, path, "BUILD.bazel")
	if _, err := os.Stat(buildPath); os.IsNotExist(err) {
		return nil
	}
	transports, err := bazel.ParseTransports(buildPath)
	if err != nil {
		slog.Warn("failed to parse transports", "path", buildPath, "error", err)
		return nil
	}
	if len(transports) == 0 {
		return nil
	}
	return simplifyTransports(transports)
}

// findAPIsSlice locates the APIs variable in the provided AST file and returns its composite literal value.
func findAPIsSlice(astFile *ast.File) *ast.CompositeLit {
	var apisSlice *ast.CompositeLit
	ast.Inspect(astFile, func(n ast.Node) bool {
		v, ok := n.(*ast.ValueSpec)
		if !ok || len(v.Names) == 0 || v.Names[0].Name != "APIs" {
			return true
		}
		if len(v.Values) == 0 {
			return true
		}
		compLit, ok := v.Values[0].(*ast.CompositeLit)
		if !ok {
			return true
		}
		apisSlice = compLit
		return false
	})
	return apisSlice
}

// extractAPIInfo parses an API composite literal to find the Path value and the index of the Transports field.
// If not found, it returns an empty string for path, and -1 for transportsIdx.
func extractAPIInfo(apiLit *ast.CompositeLit) (string, int) {
	var path string
	transportsIdx := -1
	for i, expr := range apiLit.Elts {
		kvExpr, ok := expr.(*ast.KeyValueExpr)
		if !ok {
			continue
		}
		ident, ok := kvExpr.Key.(*ast.Ident)
		if !ok {
			continue
		}
		if ident.Name == "Path" {
			if lit, ok := kvExpr.Value.(*ast.BasicLit); ok && lit.Kind == token.STRING {
				path = strings.Trim(lit.Value, "\"")
			}
		}
		if ident.Name == "Transports" {
			transportsIdx = i
		}
	}
	return path, transportsIdx
}

// simplifyTransports attempts to condense the transports map to a single "all" entry
// if all supported languages share the same transport value.
func simplifyTransports(transports map[string]string) map[string]string {
	if len(transports) != len(allLanguages) {
		return transports
	}
	var firstVal string
	for _, lang := range allLanguages {
		val, ok := transports[lang]
		if !ok {
			return transports
		}
		if firstVal == "" {
			firstVal = val
		} else if val != firstVal {
			return transports
		}
	}
	return map[string]string{"all": firstVal}
}

func createTransportsExpr(transports map[string]string) *ast.CompositeLit {
	keys := make([]string, 0, len(transports))
	for k := range transports {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	mapElt := &ast.CompositeLit{
		Type: &ast.MapType{
			Key:   ast.NewIdent("string"),
			Value: ast.NewIdent("Transport"),
		},
		Elts: []ast.Expr{},
	}
	for _, lang := range keys {
		val := transports[lang]
		var langKey ast.Expr = ast.NewIdent("lang" + strings.ToUpper(lang[:1]) + lang[1:])
		if !langConstantExists(lang) {
			langKey = &ast.BasicLit{Kind: token.STRING, Value: "\"" + lang + "\""}
		}

		var valExpr ast.Expr
		switch val {
		case "grpc":
			valExpr = ast.NewIdent("GRPC")
		case "rest":
			valExpr = ast.NewIdent("Rest")
		case "grpc+rest":
			valExpr = ast.NewIdent("GRPCRest")
		default:
			valExpr = &ast.BasicLit{Kind: token.STRING, Value: "\"" + val + "\""}
		}

		mapElt.Elts = append(mapElt.Elts, &ast.KeyValueExpr{
			Key:   langKey,
			Value: valExpr,
		})
	}
	return mapElt
}

func langConstantExists(lang string) bool {
	switch lang {
	case "all", "csharp", "go", "java", "nodejs", "php", "python", "ruby", "rust":
		return true
	}
	return false
}
