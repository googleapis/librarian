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

package librarianops

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
				Usage:    "path to googleapis repository",
				Required: true,
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			googleapisDir := cmd.String("googleapis")
			return runUpdateTransports(googleapisDir)
		},
	}
}

func runUpdateTransports(googleapisDir string) error {
	apiGoPath := "internal/serviceconfig/api.go"
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, apiGoPath, nil, parser.ParseComments)
	if err != nil {
		return fmt.Errorf("failed to parse %s: %w", apiGoPath, err)
	}

	var apisSlice *ast.CompositeLit
	ast.Inspect(f, func(n ast.Node) bool {
		v, ok := n.(*ast.ValueSpec)
		if !ok || len(v.Names) == 0 || v.Names[0].Name != "APIs" {
			return true
		}
		if len(v.Values) == 0 {
			return true
		}
		cl, ok := v.Values[0].(*ast.CompositeLit)
		if !ok {
			return true
		}
		apisSlice = cl
		return false
	})

	if apisSlice == nil {
		return fmt.Errorf("could not find APIs variable in %s", apiGoPath)
	}

	for _, elt := range apisSlice.Elts {
		cl, ok := elt.(*ast.CompositeLit)
		if !ok {
			continue
		}
		var path string
		var transportsIdx = -1
		for i, kv := range cl.Elts {
			kve, ok := kv.(*ast.KeyValueExpr)
			if !ok {
				continue
			}
			ident, ok := kve.Key.(*ast.Ident)
			if !ok {
				continue
			}
			if ident.Name == "Path" {
				if lit, ok := kve.Value.(*ast.BasicLit); ok && lit.Kind == token.STRING {
					path = strings.Trim(lit.Value, "\"")
				}
			}
			if ident.Name == "Transports" {
				transportsIdx = i
			}
		}

		if path == "" {
			continue
		}

		buildPath := filepath.Join(googleapisDir, path, "BUILD.bazel")
		if _, err := os.Stat(buildPath); os.IsNotExist(err) {
			continue
		}

		transports, err := bazel.ParseTransports(buildPath)
		if err != nil {
			slog.Warn("failed to parse transports", "path", buildPath, "error", err)
			continue
		}

		if len(transports) == 0 {
			continue
		}

		// Only simplify to "all" if all 8 recognized languages are present and share the same transport.
		allSame := len(transports) == len(allLanguages)
		var firstVal string
		if allSame {
			for _, lang := range allLanguages {
				val, ok := transports[lang]
				if !ok {
					allSame = false
					break
				}
				if firstVal == "" {
					firstVal = val
				} else if val != firstVal {
					allSame = false
					break
				}
			}
		}

		if allSame && len(transports) > 0 {
			transports = map[string]string{"all": firstVal}
		}

		// Create Transports map literal
		keys := make([]string, 0, len(transports))
		for k := range transports {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		mapElt := &ast.CompositeLit{
			Type: &ast.MapType{
				Key:   ast.NewIdent("string"),
				Value: ast.NewIdent("string"),
			},
			Elts: []ast.Expr{},
		}
		for _, lang := range keys {
			val := transports[lang]
			var langKey ast.Expr = ast.NewIdent("lang" + strings.ToUpper(lang[:1]) + lang[1:])
			if !langConstantExists(lang) {
				langKey = &ast.BasicLit{Kind: token.STRING, Value: "\"" + lang + "\""}
			}

			mapElt.Elts = append(mapElt.Elts, &ast.KeyValueExpr{
				Key:   langKey,
				Value: &ast.BasicLit{Kind: token.STRING, Value: "\"" + val + "\""},
			})
		}

		newKV := &ast.KeyValueExpr{
			Key:   ast.NewIdent("Transports"),
			Value: mapElt,
		}

		if transportsIdx != -1 {
			cl.Elts[transportsIdx] = newKV
		} else {
			cl.Elts = append(cl.Elts, newKV)
		}
	}

	out, err := os.Create(apiGoPath)
	if err != nil {
		return fmt.Errorf("failed to create %s: %w", apiGoPath, err)
	}
	defer out.Close()

	if err := format.Node(out, fset, f); err != nil {
		return fmt.Errorf("failed to format %s: %w", apiGoPath, err)
	}

	return nil
}

func langConstantExists(lang string) bool {
	switch lang {
	case "all", "csharp", "go", "java", "nodejs", "php", "python", "ruby", "rust":
		return true
	}
	return false
}
