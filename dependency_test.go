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
	"bytes"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"sync"
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
// silently weakening the check. The TestLayering* tests below enforce this.
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

// testSupportLayer is the pseudo-layer index used for testSupport packages;
// it sorts below every real layer.
var testSupportLayer = len(layers)

func TestLayeringTableIsSorted(t *testing.T) {
	// Given the layer table above.
	for _, l := range layers {
		// When the patterns of each layer are compared with their sorted order.
		sorted := slices.Sorted(slices.Values(l.pathPatterns))
		// Then they already are in that order, so diffs to the table stay small.
		if !slices.Equal(l.pathPatterns, sorted) {
			t.Errorf("layer %q path patterns are not sorted. Expected order:\n\t%s", l.name, strings.Join(sorted, "\n\t"))
		}
	}
}

func TestLayeringEveryPackageHasExactlyOneLayer(t *testing.T) {
	// Given every package in the module with non-test Go files.
	pkgs := modulePackages(t)
	for _, p := range pkgs {
		id := relative(p.ImportPath)
		if slices.Contains(testSupport, id) {
			continue
		}

		// When it is matched against every layer pattern.
		var matched []string
		for _, l := range layers {
			if slices.ContainsFunc(l.pathPatterns, func(pat string) bool { return matches(pat, id) }) {
				matched = append(matched, l.name)
			}
		}

		// Then exactly one layer claims it.
		switch len(matched) {
		case 1:
		case 0:
			t.Errorf("package %s matches no layer; add it to layers or testSupport in dependency_test.go", id)
		default:
			t.Errorf("package %s matches more than one layer: %v", id, matched)
		}
	}
}

func TestLayeringEveryPatternMatchesAPackage(t *testing.T) {
	// Given every package in the module.
	pkgs := modulePackages(t)
	ids := make([]string, 0, len(pkgs))
	for _, p := range pkgs {
		ids = append(ids, relative(p.ImportPath))
	}

	// When each layer pattern and testSupport entry is applied to them.
	for _, l := range layers {
		for _, pat := range l.pathPatterns {
			// Then it matches at least one package, so renames and deletions
			// cannot leave a stale entry behind.
			if !slices.ContainsFunc(ids, func(id string) bool { return matches(pat, id) }) {
				t.Errorf("layer %q pattern %q matches no package; remove or fix it", l.name, pat)
			}
		}
	}
	for _, id := range testSupport {
		if !slices.Contains(ids, id) {
			t.Errorf("testSupport %q is not a package; remove or fix it", id)
		}
	}
}

func TestLayeringImportsPointDownward(t *testing.T) {
	// Given every package with its layer index.
	pkgs := modulePackages(t)
	layerOf := classify(pkgs)
	for _, p := range pkgs {
		from := relative(p.ImportPath)
		fromLayer, ok := layerOf[from]
		if !ok || fromLayer == testSupportLayer {
			continue // unclassified packages are reported by another test; test support may import anything
		}

		// When each of its non-test imports inside the module is looked up.
		for _, to := range moduleImports(p) {
			toLayer, ok := layerOf[to]
			if !ok || toLayer == testSupportLayer {
				continue // covered by TestLayeringProductionCodeDoesNotImportTestSupport
			}

			// Then the imported package is in the same layer or a lower one.
			if toLayer < fromLayer {
				t.Errorf("%s (%s) imports %s (%s): dependencies must point downward", from, layers[fromLayer].name, to, layers[toLayer].name)
			}
		}
	}
}

func TestLayeringProductionCodeDoesNotImportTestSupport(t *testing.T) {
	// Given every package with its layer index.
	pkgs := modulePackages(t)
	layerOf := classify(pkgs)
	for _, p := range pkgs {
		from := relative(p.ImportPath)
		if layerOf[from] == testSupportLayer {
			continue
		}

		// When its non-test imports are checked.
		for _, to := range moduleImports(p) {
			// Then none of them is a test-support package.
			if slices.Contains(testSupport, to) {
				t.Errorf("%s imports test support package %s from non-test code", from, to)
			}
		}
	}
}

func TestLayeringThirdPartyLibrariesAreImportedOnlyThroughTheirWrapper(t *testing.T) {
	// Given every package and the gated third-party prefixes.
	pkgs := modulePackages(t)
	for _, gate := range thirdPartyGates {
		t.Run(gate.prefix, func(t *testing.T) {
			for _, p := range pkgs {
				from := relative(p.ImportPath)
				// When a non-test import starts with the gated prefix.
				for _, imp := range p.Imports {
					if !strings.HasPrefix(imp, gate.prefix) {
						continue
					}
					// Then the importer is the one wrapper package allowed to use it.
					if from != gate.only {
						t.Errorf("%s imports %s; only %s may (use its wrapper)", from, imp, gate.only)
					}
				}
			}
		})
	}
}

func TestLayeringInternalConfigDeclaresNoFunctions(t *testing.T) {
	// Given the non-test files of internal/config, which AGENTS.md defines as
	// a 1:1 mirror of librarian.yaml holding data types only.
	pkgs := modulePackages(t)
	i := slices.IndexFunc(pkgs, func(p modulePackage) bool { return relative(p.ImportPath) == "internal/config" })
	if i < 0 {
		t.Fatal("internal/config not found")
	}
	cfg := pkgs[i]

	for _, name := range cfg.GoFiles {
		// When each file is parsed.
		f, err := parser.ParseFile(token.NewFileSet(), filepath.Join(cfg.Dir, name), nil, parser.SkipObjectResolution)
		if err != nil {
			t.Fatal(err)
		}

		// Then it declares no functions or methods.
		for _, d := range f.Decls {
			if fn, ok := d.(*ast.FuncDecl); ok {
				t.Errorf("internal/config/%s declares func %s; internal/config holds data types only", name, fn.Name.Name)
			}
		}
	}
}

type modulePackage struct {
	ImportPath string
	Dir        string
	GoFiles    []string
	Imports    []string
}

// listPackages runs go list once for the whole file; every test shares the
// result.
var listPackages = sync.OnceValues(func() ([]modulePackage, error) {
	cmd := exec.Command("go", "list", "-json=ImportPath,Dir,GoFiles,Imports", "./...")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("go list ./...: %w\n%s", err, stderr.String())
	}
	dec := json.NewDecoder(bytes.NewReader(out))
	var pkgs []modulePackage
	for {
		var p modulePackage
		if err := dec.Decode(&p); err == io.EOF {
			break
		} else if err != nil {
			return nil, err
		}
		if len(p.GoFiles) > 0 && strings.HasPrefix(p.ImportPath, modulePath) {
			pkgs = append(pkgs, p)
		}
	}
	return pkgs, nil
})

// modulePackages returns every package in the module that has non-test Go
// files, with its non-test imports.
func modulePackages(t *testing.T) []modulePackage {
	t.Helper()
	pkgs, err := listPackages()
	if err != nil {
		t.Fatal(err)
	}
	if len(pkgs) == 0 {
		t.Fatal("go list found no packages")
	}
	return pkgs
}

// classify maps each package to its layer index, or testSupportLayer.
// Packages matching no layer are omitted; TestLayeringEveryPackageHasExactlyOneLayer
// reports those.
func classify(pkgs []modulePackage) map[string]int {
	layerOf := map[string]int{}
	for _, p := range pkgs {
		id := relative(p.ImportPath)
		if slices.Contains(testSupport, id) {
			layerOf[id] = testSupportLayer
			continue
		}
		for i, l := range layers {
			if slices.ContainsFunc(l.pathPatterns, func(pat string) bool { return matches(pat, id) }) {
				layerOf[id] = i
				break
			}
		}
	}
	return layerOf
}

// moduleImports returns the module-relative IDs of p's imports that are inside
// this module.
func moduleImports(p modulePackage) []string {
	var out []string
	for _, imp := range p.Imports {
		if to, ok := strings.CutPrefix(imp, modulePath+"/"); ok {
			out = append(out, to)
		}
	}
	return out
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
