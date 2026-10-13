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

package librarian

import (
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

const modulePath = "github.com/googleapis/librarian"

// layers lists the package layers of this module from top to bottom. A
// package may import packages in its own layer or any layer below it, never
// above. A path pattern ending in "/" matches every package under that
// directory; any other pattern matches one package exactly.
//
// Every package in the module must match exactly one pattern (or be listed in
// testSupport), and every pattern must match at least one package, so adding,
// moving or deleting a package forces this table to be updated rather than
// silently weakening the check.
var layers = []struct {
	name         string
	pathPatterns []string
}{
	{"entry points", []string{"cmd/", "tool/"}},
	{"command orchestration", []string{"internal/librarian"}},
	{"language integrations", []string{"internal/librarian/"}},
	{"generation engine and tool installers", []string{"internal/sidekick/", "internal/tool/"}},
	{"domain", []string{
		"internal/config",
		"internal/postprocessing",
		"internal/proto",
		"internal/repometadata",
		"internal/semver",
		"internal/serviceconfig",
		"internal/snippetmetadata",
		"internal/sources",
	}},
	{"infrastructure", []string{
		"internal/cache",
		"internal/command",
		"internal/fetch",
		"internal/filesystem",
		"internal/git",
		"internal/license",
		"internal/yaml",
	}},
}

// testSupport packages may import anything and are imported only by tests.
var testSupport = []string{
	"internal/sample",
	"internal/sidekick/api/apitest",
	"internal/testhelper",
}

// thirdPartyGates names the one package allowed to import a dependency, so
// the rest of the module goes through that package's wrapper.
var thirdPartyGates = []struct {
	prefix string
	only   string
}{
	{"go.yaml.in/yaml", "internal/yaml"},
	{"github.com/go-git/", "internal/git"},
}

type modulePackage struct {
	ImportPath string
	Dir        string
	GoFiles    []string
	Imports    []string
}

// TestDependencyRules enforces the layering above against the non-test
// imports of every package in the module.
func TestDependencyRules(t *testing.T) {
	pkgs := modulePackages(t)
	for _, l := range layers {
		if !slices.IsSorted(l.pathPatterns) {
			t.Errorf("layer %q path patterns are not sorted; keep the table sorted so diffs stay small. Expected order:\n\t%s",
				l.name, strings.Join(slices.Sorted(slices.Values(l.pathPatterns)), "\n\t"))
		}
	}
	layerOf := map[string]int{}
	for _, p := range pkgs {
		id := relative(p.ImportPath)
		if slices.Contains(testSupport, id) {
			layerOf[id] = len(layers)
			continue
		}
		var matched []int
		for i, l := range layers {
			if slices.ContainsFunc(l.pathPatterns, func(pat string) bool { return matches(pat, id) }) {
				matched = append(matched, i)
			}
		}
		switch len(matched) {
		case 1:
			layerOf[id] = matched[0]
		case 0:
			t.Errorf("package %s matches no layer; add it to layers or testSupport in dependency_test.go", id)
		default:
			t.Errorf("package %s matches more than one layer: %v", id, matched)
		}
	}
	for _, l := range layers {
		for _, pat := range l.pathPatterns {
			if !slices.ContainsFunc(pkgs, func(p modulePackage) bool { return matches(pat, relative(p.ImportPath)) }) {
				t.Errorf("layer %q pattern %q matches no package; remove or fix it", l.name, pat)
			}
		}
	}
	for _, id := range testSupport {
		if !slices.ContainsFunc(pkgs, func(p modulePackage) bool { return relative(p.ImportPath) == id }) {
			t.Errorf("testSupport %q is not a package; remove or fix it", id)
		}
	}
	if t.Failed() {
		return
	}
	for _, p := range pkgs {
		from := relative(p.ImportPath)
		fromLayer := layerOf[from]
		if fromLayer == len(layers) {
			continue // test support may import anything
		}
		for _, imp := range p.Imports {
			for _, gate := range thirdPartyGates {
				if strings.HasPrefix(imp, gate.prefix) && from != gate.only {
					t.Errorf("%s imports %s; only %s may (use its wrapper)", from, imp, gate.only)
				}
			}
			to, ok := strings.CutPrefix(imp, modulePath+"/")
			if !ok {
				continue
			}
			toLayer := layerOf[to]
			switch {
			case toLayer == len(layers):
				t.Errorf("%s imports test support package %s from non-test code", from, to)
			case toLayer < fromLayer:
				t.Errorf("%s (%s) imports %s (%s): dependencies must point downward", from, layers[fromLayer].name, to, layers[toLayer].name)
			}
		}
	}
}

// TestConfigIsPureData enforces the internal/config rule in AGENTS.md: the
// package mirrors librarian.yaml and declares no functions or methods.
func TestConfigIsPureData(t *testing.T) {
	for _, p := range modulePackages(t) {
		if relative(p.ImportPath) != "internal/config" {
			continue
		}
		for _, name := range p.GoFiles {
			f, err := parser.ParseFile(token.NewFileSet(), filepath.Join(p.Dir, name), nil, parser.SkipObjectResolution)
			if err != nil {
				t.Fatal(err)
			}
			for _, d := range f.Decls {
				if fn, ok := d.(*ast.FuncDecl); ok {
					t.Errorf("internal/config/%s declares func %s; internal/config holds data types only", name, fn.Name.Name)
				}
			}
		}
	}
}

// modulePackages returns every package in the module that has non-test Go
// files, with its non-test imports.
func modulePackages(t *testing.T) []modulePackage {
	t.Helper()
	dec := json.NewDecoder(strings.NewReader(rungo(t, "list", "-json=ImportPath,Dir,GoFiles,Imports", "./...")))
	var pkgs []modulePackage
	for {
		var p modulePackage
		if err := dec.Decode(&p); err == io.EOF {
			break
		} else if err != nil {
			t.Fatalf("parsing go list output: %v", err)
		}
		if len(p.GoFiles) > 0 && strings.HasPrefix(p.ImportPath, modulePath) {
			pkgs = append(pkgs, p)
		}
	}
	if len(pkgs) == 0 {
		t.Fatal("go list found no packages")
	}
	return pkgs
}

func relative(importPath string) string {
	if importPath == modulePath {
		return "."
	}
	return strings.TrimPrefix(importPath, modulePath+"/")
}

func matches(pattern, id string) bool {
	if dir, ok := strings.CutSuffix(pattern, "/"); ok {
		return strings.HasPrefix(id, dir+"/")
	}
	return pattern == id
}
