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
	"encoding/json"
	"fmt"
	"go/ast"
	"go/doc"
	"go/parser"
	"go/token"
	"html"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strconv"
	"strings"

	"github.com/googleapis/librarian/internal/yaml"
)

// generatedSpec is the YAML body of a ```generated block. Each kind derives
// a markdown table from the repository at build time, so the page never
// carries a copy that can go stale.
type generatedSpec struct {
	Kind string `yaml:"kind"`
	// Dir scopes kinds that read one directory (dispatch, commands,
	// workflows). Each kind has a default.
	Dir string `yaml:"dir"`
	// Prefix filters the packages kind to import paths under a prefix.
	Prefix string `yaml:"prefix"`
	// Layers buckets packages for the imports kind. Every package must
	// match one layer; a match entry ending in "/" is a prefix, any other
	// is an exact import path.
	Layers []importsLayer `yaml:"layers"`
}

type importsLayer struct {
	ID    string   `yaml:"id"`
	Label string   `yaml:"label"`
	Kind  string   `yaml:"kind"`
	Match []string `yaml:"match"`
}

func (s *site) renderGenerated(body string) (string, error) {
	spec, err := yaml.Unmarshal[generatedSpec]([]byte(body))
	if err != nil {
		return "", fmt.Errorf("generated: %w", err)
	}
	var table string
	switch spec.Kind {
	case "packages":
		table, err = packagesTable(s.rootDir, spec.Prefix)
	case "dispatch":
		table, err = dispatchTable(s.rootDir, orDefault(spec.Dir, "internal/librarian"))
	case "commands":
		table, err = commandsTable(s.rootDir, orDefault(spec.Dir, "internal/librarian"))
	case "workflows":
		table, err = workflowsTable(s.rootDir, orDefault(spec.Dir, ".github/workflows"))
	case "imports":
		// The import map is interactive HTML rather than a table.
		out, err := importsMap(s.rootDir, spec.Layers)
		if err != nil {
			return "", fmt.Errorf("generated imports: %w", err)
		}
		return "\n" + out + "\n", nil
	default:
		return "", fmt.Errorf("generated: unknown kind %q", spec.Kind)
	}
	if err != nil {
		return "", fmt.Errorf("generated %s: %w", spec.Kind, err)
	}
	// A blank line on each side lets goldmark parse the table inside the div.
	return "\n<div class=\"generated\">\n\n" + table + "\n</div>\n", nil
}

func orDefault(v, def string) string {
	if v == "" {
		return def
	}
	return v
}

// markdownTable renders rows as a GFM table. Pipes in cells are escaped.
func markdownTable(header []string, rows [][]string) string {
	var b strings.Builder
	writeRow := func(cells []string) {
		b.WriteString("|")
		for _, c := range cells {
			b.WriteString(" " + strings.ReplaceAll(c, "|", `\|`) + " |")
		}
		b.WriteString("\n")
	}
	writeRow(header)
	b.WriteString("|" + strings.Repeat("---|", len(header)) + "\n")
	for _, r := range rows {
		writeRow(r)
	}
	return b.String()
}

// goPackage is one package of the module as read from its source files.
type goPackage struct {
	Path     string   `json:"id"`
	Name     string   `json:"-"`
	Synopsis string   `json:"desc"`
	Imports  []string `json:"imports"`
}

// listPackages returns every Go package under root, sorted by path, with
// the first sentence of its package comment and its module-internal
// imports from non-test files.
func listPackages(root string) ([]goPackage, error) {
	module, err := modulePath(root)
	if err != nil {
		return nil, err
	}
	var pkgs []goPackage
	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			return nil
		}
		name := d.Name()
		if path != root && (strings.HasPrefix(name, ".") || strings.HasPrefix(name, "_") || name == "testdata" || name == "node_modules" || name == "vendor") {
			return filepath.SkipDir
		}
		pkg, ok, err := readPackage(path, module)
		if err != nil || !ok {
			return err
		}
		rel, _ := filepath.Rel(root, path)
		pkg.Path = filepath.ToSlash(rel)
		if pkg.Path == "." {
			pkg.Path = module
		}
		pkgs = append(pkgs, pkg)
		return nil
	})
	return pkgs, err
}

// readPackage parses the non-test Go files in dir. ok is false when dir
// holds no such files.
func readPackage(dir, module string) (pkg goPackage, ok bool, err error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return pkg, false, err
	}
	fset := token.NewFileSet()
	imports := map[string]bool{}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") || strings.HasSuffix(e.Name(), "_test.go") {
			continue
		}
		f, err := parser.ParseFile(fset, filepath.Join(dir, e.Name()), nil, parser.ImportsOnly|parser.ParseComments)
		if err != nil {
			return pkg, false, err
		}
		ok = true
		pkg.Name = f.Name.Name
		if f.Doc != nil && pkg.Synopsis == "" {
			pkg.Synopsis = new(doc.Package).Synopsis(f.Doc.Text())
		}
		for _, imp := range f.Imports {
			if p, err := strconv.Unquote(imp.Path.Value); err == nil && strings.HasPrefix(p, module+"/") {
				imports[strings.TrimPrefix(p, module+"/")] = true
			}
		}
	}
	// Never nil: the map's JSON must carry [] rather than null.
	pkg.Imports = append([]string{}, slices.Sorted(maps.Keys(imports))...)
	return pkg, ok, nil
}

// packagesTable lists every Go package under root with the first sentence of
// its package comment.
func packagesTable(root, prefix string) (string, error) {
	pkgs, err := listPackages(root)
	if err != nil {
		return "", err
	}
	var rows [][]string
	for _, pkg := range pkgs {
		if !strings.HasPrefix(pkg.Path, strings.TrimSuffix(prefix, "/")) {
			continue
		}
		synopsis := pkg.Synopsis
		if pkg.Name == "main" {
			synopsis = strings.TrimSpace("Command. " + synopsis)
		}
		rows = append(rows, []string{"`" + pkg.Path + "`", synopsis})
	}
	return markdownTable([]string{"Package", "Description"}, rows), nil
}

// importsMap renders the interactive package map: packages grouped into the
// given layers, with their imports embedded as JSON for app.js to draw.
func importsMap(root string, layers []importsLayer) (string, error) {
	if len(layers) == 0 {
		return "", fmt.Errorf("layers are required")
	}
	pkgs, err := listPackages(root)
	if err != nil {
		return "", err
	}
	type layerData struct {
		ID       string      `json:"id"`
		Label    string      `json:"label"`
		Kind     string      `json:"kind"`
		Packages []goPackage `json:"packages"`
	}
	data := make([]layerData, len(layers))
	for i, l := range layers {
		data[i] = layerData{ID: l.ID, Label: l.Label, Kind: kindOr(l.Kind), Packages: []goPackage{}}
	}
	var unassigned []string
	for _, pkg := range pkgs {
		i := layerIndex(layers, pkg.Path)
		if i < 0 {
			unassigned = append(unassigned, pkg.Path)
			continue
		}
		data[i].Packages = append(data[i].Packages, pkg)
	}
	if len(unassigned) > 0 {
		return "", fmt.Errorf("packages not in any layer: %s", strings.Join(unassigned, ", "))
	}
	js, err := json.Marshal(map[string]any{"layers": data})
	if err != nil {
		return "", err
	}
	// No blank lines: goldmark ends an HTML block at the first one.
	var b strings.Builder
	b.WriteString(`<div class="pkgmap">` + "\n")
	b.WriteString(`<p class="pkgmap-hint">Click a package to see what it imports <span class="key imp">imports</span> and who imports it <span class="key dep">imported by</span>. Computed from the source tree at build time.</p>` + "\n")
	b.WriteString(`<div class="pkgmap-body">` + "\n")
	b.WriteString(`<div class="pkgmap-layers">` + "\n")
	for _, l := range data {
		fmt.Fprintf(&b, `<section class="pkgmap-layer kind-%s"><h4>%s <span>%d</span></h4><div class="pkgmap-pkgs">`, html.EscapeString(l.Kind), html.EscapeString(l.Label), len(l.Packages))
		for _, p := range l.Packages {
			fmt.Fprintf(&b, `<button type="button" class="pkg" data-id="%s" title="%s">%s</button>`, html.EscapeString(p.Path), html.EscapeString(p.Synopsis), html.EscapeString(strings.TrimPrefix(p.Path, "internal/")))
		}
		b.WriteString("</div></section>\n")
	}
	b.WriteString(`<svg class="pkgmap-wires" aria-hidden="true"></svg>` + "\n")
	b.WriteString("</div>\n")
	b.WriteString(`<aside class="pkgmap-panel"><p class="muted">Nothing selected.</p></aside>` + "\n")
	b.WriteString("</div>\n")
	b.WriteString(`<script type="application/json" class="pkgmap-data">` + string(js) + "</script>\n")
	b.WriteString("</div>")
	return b.String(), nil
}

func layerIndex(layers []importsLayer, path string) int {
	for i, l := range layers {
		for _, m := range l.Match {
			if strings.HasSuffix(m, "/") && strings.HasPrefix(path, m) || m == path {
				return i
			}
		}
	}
	return -1
}

func modulePath(root string) (string, error) {
	data, err := os.ReadFile(filepath.Join(root, "go.mod"))
	if err != nil {
		return "", err
	}
	for line := range strings.Lines(string(data)) {
		if after, ok := strings.CutPrefix(line, "module "); ok {
			return strings.TrimSpace(after), nil
		}
	}
	return "", fmt.Errorf("no module line in go.mod")
}

// dispatchTable is the language dispatch matrix: every top-level declaration
// in dir that names a config.Language* constant, against the languages it
// names. It is the implicit contract a language package must fulfil.
func dispatchTable(root, dir string) (string, error) {
	type row struct {
		decl, file string
		line       int
		langs      map[string]bool
	}
	var rows []*row
	langSet := map[string]bool{}
	fset := token.NewFileSet()
	files, err := goFiles(filepath.Join(root, dir))
	if err != nil {
		return "", err
	}
	for _, path := range files {
		f, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			return "", err
		}
		for _, decl := range f.Decls {
			for _, named := range namedDecls(decl) {
				r := &row{decl: named.name, file: filepath.Base(path), line: fset.Position(named.node.Pos()).Line, langs: map[string]bool{}}
				ast.Inspect(named.node, func(n ast.Node) bool {
					if lang, ok := languageConstant(n); ok {
						r.langs[lang] = true
						langSet[lang] = true
					}
					return true
				})
				if len(r.langs) > 0 {
					rows = append(rows, r)
				}
			}
		}
	}
	if len(rows) == 0 {
		return "", fmt.Errorf("no config.Language* references under %s", dir)
	}
	langs := slices.Sorted(maps.Keys(langSet))
	sort.SliceStable(rows, func(i, j int) bool {
		if rows[i].file != rows[j].file {
			return rows[i].file < rows[j].file
		}
		return rows[i].line < rows[j].line
	})
	header := append([]string{"Declaration", "File"}, langs...)
	var out [][]string
	for _, r := range rows {
		cells := []string{"`" + r.decl + "`", "`" + r.file + "`"}
		for _, l := range langs {
			if r.langs[l] {
				cells = append(cells, "✓")
			} else {
				cells = append(cells, "·")
			}
		}
		out = append(out, cells)
	}
	return markdownTable(header, out), nil
}

type namedDecl struct {
	name string
	node ast.Node
}

// namedDecls returns the names declared by a top-level declaration: one for
// a function or method, one per spec for grouped declarations.
func namedDecls(decl ast.Decl) []namedDecl {
	switch d := decl.(type) {
	case *ast.FuncDecl:
		name := d.Name.Name
		if recv := receiverName(d); recv != "" {
			name = recv + "." + name
		}
		return []namedDecl{{name, d}}
	case *ast.GenDecl:
		var out []namedDecl
		for _, spec := range d.Specs {
			switch s := spec.(type) {
			case *ast.ValueSpec:
				for _, n := range s.Names {
					out = append(out, namedDecl{n.Name, s})
				}
			case *ast.TypeSpec:
				out = append(out, namedDecl{s.Name.Name, s})
			}
		}
		return out
	}
	return nil
}

// languageConstant reports whether n is a config.Language<Name> selector and
// returns the lower-case language name.
func languageConstant(n ast.Node) (string, bool) {
	sel, ok := n.(*ast.SelectorExpr)
	if !ok {
		return "", false
	}
	pkg, ok := sel.X.(*ast.Ident)
	if !ok || pkg.Name != "config" {
		return "", false
	}
	name, ok := strings.CutPrefix(sel.Sel.Name, "Language")
	if !ok || name == "" || name == "Unknown" || name == "All" {
		return "", false
	}
	return strings.ToLower(name), true
}

func goFiles(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var files []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".go") && !strings.HasSuffix(e.Name(), "_test.go") {
			files = append(files, filepath.Join(dir, e.Name()))
		}
	}
	return files, nil
}

// commandsTable lists every cli.Command literal under dir with its usage
// line. Nested Commands fields produce "parent child" names.
func commandsTable(root, dir string) (string, error) {
	type row struct{ name, usage, file string }
	var rows []row
	fset := token.NewFileSet()
	files, err := goFiles(filepath.Join(root, dir))
	if err != nil {
		return "", err
	}
	for _, path := range files {
		f, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			return "", err
		}
		seen := map[*ast.CompositeLit]bool{}
		var visit func(lit *ast.CompositeLit, prefix string)
		visit = func(lit *ast.CompositeLit, prefix string) {
			seen[lit] = true
			fields := literalStrings(lit)
			name := prefix + fields["Name"]
			rows = append(rows, row{name, fields["Usage"], filepath.ToSlash(filepath.Join(dir, filepath.Base(path)))})
			for _, kv := range keyValues(lit) {
				if key, ok := kv.Key.(*ast.Ident); !ok || key.Name != "Commands" {
					continue
				}
				list, ok := kv.Value.(*ast.CompositeLit)
				if !ok {
					continue
				}
				for _, elt := range list.Elts {
					if child, ok := compositeLit(elt); ok {
						visit(child, name+" ")
					}
				}
			}
		}
		ast.Inspect(f, func(n ast.Node) bool {
			lit, ok := n.(*ast.CompositeLit)
			if ok && !seen[lit] && isCLICommand(lit.Type) {
				visit(lit, "")
			}
			return true
		})
	}
	if len(rows) == 0 {
		return "", fmt.Errorf("no cli.Command literals under %s", dir)
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].name < rows[j].name })
	var out [][]string
	for _, r := range rows {
		out = append(out, []string{"`" + r.name + "`", r.usage, "`" + r.file + "`"})
	}
	return markdownTable([]string{"Command", "Usage", "Defined in"}, out), nil
}

func isCLICommand(t ast.Expr) bool {
	sel, ok := t.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	pkg, ok := sel.X.(*ast.Ident)
	return ok && pkg.Name == "cli" && sel.Sel.Name == "Command"
}

func compositeLit(e ast.Expr) (*ast.CompositeLit, bool) {
	if u, ok := e.(*ast.UnaryExpr); ok {
		e = u.X
	}
	lit, ok := e.(*ast.CompositeLit)
	return lit, ok
}

func keyValues(lit *ast.CompositeLit) []*ast.KeyValueExpr {
	var out []*ast.KeyValueExpr
	for _, elt := range lit.Elts {
		if kv, ok := elt.(*ast.KeyValueExpr); ok {
			out = append(out, kv)
		}
	}
	return out
}

// literalStrings returns the string-literal fields of a composite literal.
func literalStrings(lit *ast.CompositeLit) map[string]string {
	out := map[string]string{}
	for _, kv := range keyValues(lit) {
		key, ok := kv.Key.(*ast.Ident)
		if !ok {
			continue
		}
		if lit, ok := kv.Value.(*ast.BasicLit); ok && lit.Kind == token.STRING {
			if v, err := strconv.Unquote(lit.Value); err == nil {
				out[key.Name] = v
			}
		}
	}
	return out
}

// workflowsTable lists the GitHub Actions workflows in dir with their
// triggers and jobs.
func workflowsTable(root, dir string) (string, error) {
	type workflow struct {
		Name string         `yaml:"name"`
		On   any            `yaml:"on"`
		Jobs map[string]any `yaml:"jobs"`
	}
	entries, err := os.ReadDir(filepath.Join(root, dir))
	if err != nil {
		return "", err
	}
	var rows [][]string
	for _, e := range entries {
		ext := filepath.Ext(e.Name())
		if e.IsDir() || (ext != ".yaml" && ext != ".yml") {
			continue
		}
		w, err := yaml.Read[workflow](filepath.Join(root, dir, e.Name()))
		if err != nil {
			return "", fmt.Errorf("%s: %w", e.Name(), err)
		}
		var triggers []string
		switch on := w.On.(type) {
		case string:
			triggers = []string{on}
		case []any:
			for _, t := range on {
				triggers = append(triggers, fmt.Sprint(t))
			}
		case map[string]any:
			triggers = slices.Sorted(maps.Keys(on))
		}
		jobs := slices.Sorted(maps.Keys(w.Jobs))
		rows = append(rows, []string{w.Name, "`" + e.Name() + "`", strings.Join(triggers, ", "), strings.Join(jobs, ", ")})
	}
	if len(rows) == 0 {
		return "", fmt.Errorf("no workflows under %s", dir)
	}
	return markdownTable([]string{"Workflow", "File", "Triggers", "Jobs"}, rows), nil
}
