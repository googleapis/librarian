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
		path, restIdx, noRestIdx := extractRESTNumericEnumsInfo(apiLit)
		if path == "" {
			continue
		}
		numericEnums := readRestNumericEnums(googleapisDir, path)
		yesMap, noMap := splitNumericEnums(numericEnums)

		// Update RESTNumericEnums
		if len(yesMap) == 0 {
			if restIdx != -1 {
				apiLit.Elts = removeElement(apiLit.Elts, restIdx)
				// Re-adjust noRestIdx if it was after restIdx
				if noRestIdx > restIdx {
					noRestIdx--
				}
			}
		} else {
			kv := &ast.KeyValueExpr{
				Key:   ast.NewIdent("RESTNumericEnums"),
				Value: createRestNumericEnumsExpr(yesMap),
			}
			if restIdx != -1 {
				apiLit.Elts[restIdx] = kv
			} else {
				apiLit.Elts = append(apiLit.Elts, kv)
			}
		}

		// Update NoRESTNumericEnums
		if len(noMap) == 0 {
			if noRestIdx != -1 {
				apiLit.Elts = removeElement(apiLit.Elts, noRestIdx)
			}
		} else {
			kv := &ast.KeyValueExpr{
				Key:   ast.NewIdent("NoRESTNumericEnums"),
				Value: createRestNumericEnumsExpr(noMap),
			}
			if noRestIdx != -1 {
				apiLit.Elts[noRestIdx] = kv
			} else {
				apiLit.Elts = append(apiLit.Elts, kv)
			}
		}
	}

	out, err := os.Create(apiGoPath)
	if err != nil {
		return fmt.Errorf("failed to create %s: %w", apiGoPath, err)
	}
	defer out.Close()
	return format.Node(out, fset, astFile)
}

func removeElement(elts []ast.Expr, i int) []ast.Expr {
	return append(elts[:i], elts[i+1:]...)
}

func splitNumericEnums(numericEnums map[string]bool) (map[string]bool, map[string]bool) {
	yesMap := make(map[string]bool)
	noMap := make(map[string]bool)
	for lang, val := range numericEnums {
		if val {
			yesMap[lang] = true
		} else {
			noMap[lang] = true
		}
	}
	return yesMap, noMap
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
	if len(numericEnums) == 0 {
		return nil
	}
	return simplifyRestNumericEnums(numericEnums)
}

func extractRESTNumericEnumsInfo(apiLit *ast.CompositeLit) (string, int, int) {
	var path string
	restIdx := -1
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
		if ident.Name == "RESTNumericEnums" {
			restIdx = i
		}
		if ident.Name == "NoRESTNumericEnums" {
			noRestIdx = i
		}
	}
	return path, restIdx, noRestIdx
}

func simplifyRestNumericEnums(numericEnums map[string]bool) map[string]bool {
	if len(numericEnums) != bazelLangs {
		return numericEnums
	}
	var firstVal bool
	var firstSet bool
	for lang := range langToConstant {
		if lang == "all" {
			continue
		}
		val, ok := numericEnums[lang]
		if !ok {
			return numericEnums
		}
		if !firstSet {
			firstVal = val
			firstSet = true
		} else if val != firstVal {
			return numericEnums
		}
	}
	return map[string]bool{"all": firstVal}
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
