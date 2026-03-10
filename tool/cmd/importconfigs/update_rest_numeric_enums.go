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
	"maps"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/googleapis/librarian/tool/cmd/importconfigs/bazel"
	"github.com/urfave/cli/v3"
)

func updateRestNumericEnumsCommand() *cli.Command {
	return &cli.Command{
		Name:  "update-rest-numeric-enums",
		Usage: "update rest numeric enums values in internal/serviceconfig/api.go from BUILD.bazel files",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "googleapis",
				Usage:    "path to googleapis dir",
				Required: true,
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			googleapisDir := cmd.String("googleapis")
			return runUpdateRestNumericEnums("internal/serviceconfig/api.go", googleapisDir)
		},
	}
}

func runUpdateRestNumericEnums(apiGoPath, googleapisDir string) error {
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
		path, index := extractRESTNumericEnumsInfo(apiLit)
		if path == "" {
			continue
		}
		noRestNumericEnums := readRestNumericEnums(googleapisDir, path)
		if len(noRestNumericEnums) == 0 {
			if index != -1 {
				// Remove the NoRESTNumericEnums field if it exists and is now the default.
				apiLit.Elts = append(apiLit.Elts[:index], apiLit.Elts[index+1:]...)
			}
			continue
		}
		restKV := &ast.KeyValueExpr{
			Key:   ast.NewIdent("NoRESTNumericEnums"),
			Value: createRestNumericEnumsExpr(noRestNumericEnums),
		}
		if index != -1 {
			apiLit.Elts[index] = restKV
		} else {
			apiLit.Elts = append(apiLit.Elts, restKV)
		}
	}

	out, err := os.Create(apiGoPath)
	if err != nil {
		return fmt.Errorf("failed to create %s: %w", apiGoPath, err)
	}
	defer out.Close()
	return format.Node(out, fset, astFile)
}

func readRestNumericEnums(googleapisDir, path string) map[string]bool {
	buildPath := filepath.Join(googleapisDir, path, "BUILD.bazel")
	if _, err := os.Stat(buildPath); os.IsNotExist(err) {
		return nil
	}
	numericEnums, err := bazel.ParseRESTNumericEnums(buildPath)
	if err != nil {
		slog.Warn("failed to parse rest numeric enums", "path", buildPath, "error", err)
		return nil
	}
	return simplifyRestNumericEnums(numericEnums)
}

func extractRESTNumericEnumsInfo(apiLit *ast.CompositeLit) (string, int) {
	var path string
	noRestIdx := -1
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
		if ident.Name == "NoRESTNumericEnums" {
			noRestIdx = i
		}
	}
	return path, noRestIdx
}

func simplifyRestNumericEnums(restNumericEnums map[string]bool) map[string]bool {
	var (
		firstVal bool
		firstSet bool
	)
	for lang := range langToConstant {
		if lang == "all" {
			continue
		}
		val, ok := restNumericEnums[lang]
		if !ok {
			return removeDefaults(restNumericEnums)
		}
		if !firstSet {
			firstVal = val
			firstSet = true
		} else if val != firstVal {
			return removeDefaults(restNumericEnums)
		}
	}
	if !firstVal {
		// All languages need rest_numeric_enums.
		return make(map[string]bool)
	}
	// All languages do not need rest_numeric_enums.
	// Return all: true.
	return map[string]bool{"all": firstVal}
}

func removeDefaults(restNumericEnums map[string]bool) map[string]bool {
	maps.DeleteFunc(restNumericEnums, func(k string, v bool) bool {
		return !v
	})
	return restNumericEnums
}

func createRestNumericEnumsExpr(numericEnums map[string]bool) *ast.CompositeLit {
	keys := make([]string, 0, len(numericEnums))
	for k := range numericEnums {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	mapElt := &ast.CompositeLit{
		Type: &ast.MapType{
			Key:   ast.NewIdent("string"),
			Value: ast.NewIdent("bool"),
		},
		Elts: []ast.Expr{},
	}
	for _, lang := range keys {
		var langKey ast.Expr
		if constName, ok := langToConstant[lang]; ok {
			langKey = ast.NewIdent(constName)
		} else {
			langKey = &ast.BasicLit{Kind: token.STRING, Value: "\"" + lang + "\""}
		}

		mapElt.Elts = append(mapElt.Elts, &ast.KeyValueExpr{
			Key:   langKey,
			Value: ast.NewIdent("true"),
		})
	}
	return mapElt
}
