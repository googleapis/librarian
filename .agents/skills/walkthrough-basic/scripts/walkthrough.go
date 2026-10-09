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

//go:build ignore

// Command walkthrough extracts facts from a Go module and renders them,
// together with an agent-written narrative, into a single interactive HTML
// walkthrough.
//
//	go run walkthrough.go analyze -root . -out facts.json
//	go run walkthrough.go render -root . -facts facts.json -narrative narrative.json -out walkthrough.html
package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"go/ast"
	"go/build"
	"go/doc"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	dataPlaceholder = "__WALKTHROUGH_DATA__"
	maxExported     = 60
	maxInlineLines  = 600
)

// Facts is everything that can be derived mechanically from the source tree.
type Facts struct {
	Repo        string    `json:"repo"`
	Module      string    `json:"module"`
	GoVersion   string    `json:"goVersion"`
	SHA         string    `json:"sha"`
	ShortSHA    string    `json:"shortSha"`
	Ref         string    `json:"ref"`
	GeneratedAt string    `json:"generatedAt"`
	TotalLOC    int       `json:"totalLoc"`
	Packages    []Package `json:"packages"`
	Commands    []Command `json:"commands"`
	Docs        []string  `json:"docs"`
}

// Package describes one Go package in the module.
type Package struct {
	ID         string     `json:"id"`
	Name       string     `json:"name"`
	Doc        string     `json:"doc,omitempty"`
	LOC        int        `json:"loc"`
	Files      []FileStat `json:"files"`
	Tests      int        `json:"tests"`
	Imports    []string   `json:"imports"`
	ImportedBy []string   `json:"importedBy"`
	TestOnly   bool       `json:"testOnly,omitempty"`
	Exported   []string   `json:"exported"`
	MoreExport int        `json:"moreExported,omitempty"`
	testDeps   []string
}

// FileStat is a file name with its line count.
type FileStat struct {
	Name  string `json:"name"`
	Lines int    `json:"lines"`
}

// Command is a CLI command found as a composite literal of a *.Command type.
type Command struct {
	Name   string `json:"name"`
	Usage  string `json:"usage,omitempty"`
	Action string `json:"action,omitempty"`
	Pkg    string `json:"pkg"`
	File   string `json:"file"`
	Line   int    `json:"line"`
}

// Narrative is the agent-written interpretation of the facts.
type Narrative struct {
	Title        string            `json:"title"`
	Summary      string            `json:"summary"`
	Layers       []Layer           `json:"layers"`
	Descriptions map[string]string `json:"descriptions"`
	Tours        []Tour            `json:"tours"`
	Flows        []Flow            `json:"flows"`
	Findings     []Finding         `json:"findings"`
}

// Layer is an architectural tier on the map.
type Layer struct {
	ID       string   `json:"id"`
	Title    string   `json:"title"`
	Hint     string   `json:"hint"`
	Packages []string `json:"packages"`
}

// Tour is an ordered, step-by-step walkthrough.
type Tour struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Subtitle string `json:"subtitle"`
	Group    string `json:"group"`
	Steps    []Step `json:"steps"`
}

// Step is a single tour step.
type Step struct {
	T    string   `json:"t"`
	Runs string   `json:"runs,omitempty"`
	Body string   `json:"body"`
	Look []string `json:"look"`
	Pkg  string   `json:"pkg,omitempty"`
}

// Flow traces one command through the code.
type Flow struct {
	ID    string      `json:"id"`
	Title string      `json:"title"`
	Cmd   string      `json:"cmd"`
	File  string      `json:"file"`
	Steps [][2]string `json:"steps"`
}

// Finding is an architectural observation backed by a file.
type Finding struct {
	T   string `json:"t"`
	B   string `json:"b"`
	F   string `json:"f"`
	Pkg string `json:"pkg,omitempty"`
}

// File is inlined source or a directory listing for the in-page inspector.
type File struct {
	Dir     bool     `json:"dir,omitempty"`
	Lines   int      `json:"lines"`
	Trunc   bool     `json:"trunc,omitempty"`
	Content string   `json:"content,omitempty"`
	Pkg     string   `json:"pkg,omitempty"`
	Entries []string `json:"entries,omitempty"`
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: walkthrough analyze|render [flags]")
		os.Exit(2)
	}
	var err error
	switch os.Args[1] {
	case "analyze":
		err = runAnalyze(os.Args[2:])
	case "render":
		err = runRender(os.Args[2:])
	default:
		err = fmt.Errorf("unknown subcommand %q", os.Args[1])
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "walkthrough:", err)
		os.Exit(1)
	}
}

func runAnalyze(args []string) error {
	fs := flag.NewFlagSet("analyze", flag.ExitOnError)
	root := fs.String("root", ".", "module root")
	out := fs.String("out", "facts.json", "output facts file")
	fs.Parse(args)
	facts, err := analyze(*root)
	if err != nil {
		return err
	}
	b, err := json.MarshalIndent(facts, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(*out, b, 0o644); err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "%s @ %s: %d packages, %d LOC, %d commands -> %s\n",
		facts.Module, facts.ShortSHA, len(facts.Packages), facts.TotalLOC, len(facts.Commands), *out)
	return nil
}

func analyze(root string) (*Facts, error) {
	root, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	gomod, err := os.ReadFile(filepath.Join(root, "go.mod"))
	if err != nil {
		return nil, err
	}
	f := &Facts{GeneratedAt: time.Now().UTC().Format(time.RFC3339)}
	for _, line := range strings.Split(string(gomod), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 2 && fields[0] == "module" {
			f.Module = fields[1]
		}
		if len(fields) == 2 && fields[0] == "go" {
			f.GoVersion = "Go " + fields[1]
		}
	}
	f.SHA = gitOut(root, "rev-parse", "HEAD")
	if len(f.SHA) >= 8 {
		f.ShortSHA = f.SHA[:8]
	}
	f.Ref = gitOut(root, "rev-parse", "--abbrev-ref", "HEAD")
	f.Repo = githubRepo(gitOut(root, "remote", "get-url", "origin"))
	byID := map[string]*Package{}
	err = filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel := filepath.ToSlash(mustRel(root, p))
		if !d.IsDir() {
			if strings.HasSuffix(p, ".md") && (strings.Count(rel, "/") == 0 || strings.HasPrefix(rel, "doc/")) {
				f.Docs = append(f.Docs, rel)
			}
			return nil
		}
		if p != root && skipDir(d.Name()) {
			return filepath.SkipDir
		}
		if p != root && fileExists(filepath.Join(p, "go.mod")) {
			return filepath.SkipDir
		}
		pkg, cmds, err := loadPackage(root, p, f.Module)
		if err != nil || pkg == nil {
			return err
		}
		byID[pkg.ID] = pkg
		f.Packages = append(f.Packages, *pkg)
		f.Commands = append(f.Commands, cmds...)
		return nil
	})
	if err != nil {
		return nil, err
	}
	importedBy := map[string][]string{}
	testUsers := map[string]bool{}
	for _, p := range f.Packages {
		for _, imp := range p.Imports {
			importedBy[imp] = append(importedBy[imp], p.ID)
		}
		for _, imp := range p.testDeps {
			testUsers[imp] = true
		}
	}
	for i := range f.Packages {
		p := &f.Packages[i]
		p.ImportedBy = nonNil(importedBy[p.ID])
		sort.Strings(p.ImportedBy)
		p.TestOnly = p.Name != "main" && testUsers[p.ID] && len(p.ImportedBy) == 0
		f.TotalLOC += p.LOC
	}
	sort.Slice(f.Packages, func(i, j int) bool { return f.Packages[i].ID < f.Packages[j].ID })
	return f, nil
}

func loadPackage(root, dir, module string) (*Package, []Command, error) {
	bp, err := build.ImportDir(dir, build.ImportComment)
	if err != nil {
		var noGo *build.NoGoError
		if errors.As(err, &noGo) {
			return nil, nil, nil
		}
		var multi *build.MultiplePackageError
		if errors.As(err, &multi) {
			return nil, nil, nil
		}
		return nil, nil, fmt.Errorf("%s: %w", dir, err)
	}
	if len(bp.GoFiles) == 0 {
		return nil, nil, nil
	}
	id := filepath.ToSlash(mustRel(root, dir))
	pkg := &Package{
		ID:      id,
		Name:    bp.Name,
		Doc:     bp.Doc,
		Tests:   len(bp.TestGoFiles) + len(bp.XTestGoFiles),
		Imports: internalImports(bp.Imports, module),
	}
	pkg.testDeps = internalImports(append(bp.TestImports, bp.XTestImports...), module)
	fset := token.NewFileSet()
	var cmds []Command
	var exported []string
	for _, name := range bp.GoFiles {
		src, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			return nil, nil, err
		}
		lines := countLines(src)
		pkg.LOC += lines
		pkg.Files = append(pkg.Files, FileStat{Name: name, Lines: lines})
		file, err := parser.ParseFile(fset, name, src, parser.SkipObjectResolution)
		if err != nil {
			return nil, nil, err
		}
		exported = append(exported, exportedDecls(file)...)
		cmds = append(cmds, findCommands(fset, file, id, path.Join(id, name))...)
	}
	sort.Strings(exported)
	if len(exported) > maxExported {
		pkg.MoreExport = len(exported) - maxExported
		exported = exported[:maxExported]
	}
	pkg.Exported = nonNil(exported)
	if pkg.Doc == "" {
		pkg.Doc = docSynopsis(dir, bp.GoFiles)
	}
	return pkg, cmds, nil
}

func docSynopsis(dir string, files []string) string {
	fset := token.NewFileSet()
	for _, name := range files {
		f, err := parser.ParseFile(fset, filepath.Join(dir, name), nil, parser.PackageClauseOnly|parser.ParseComments)
		if err == nil && f.Doc != nil {
			return new(doc.Package).Synopsis(f.Doc.Text())
		}
	}
	return ""
}

func exportedDecls(f *ast.File) []string {
	var out []string
	for _, decl := range f.Decls {
		switch d := decl.(type) {
		case *ast.FuncDecl:
			if d.Recv == nil && d.Name.IsExported() {
				out = append(out, "func "+d.Name.Name)
			}
		case *ast.GenDecl:
			if d.Tok != token.TYPE {
				continue
			}
			for _, s := range d.Specs {
				if ts := s.(*ast.TypeSpec); ts.Name.IsExported() {
					out = append(out, "type "+ts.Name.Name)
				}
			}
		}
	}
	return out
}

// findCommands detects CLI command definitions such as urfave/cli
// &cli.Command{Name: ...} or cobra &cobra.Command{Use: ...}.
func findCommands(fset *token.FileSet, f *ast.File, pkg, file string) []Command {
	var out []Command
	ast.Inspect(f, func(n ast.Node) bool {
		lit, ok := n.(*ast.CompositeLit)
		if !ok {
			return true
		}
		sel, ok := lit.Type.(*ast.SelectorExpr)
		if !ok || sel.Sel.Name != "Command" {
			return true
		}
		c := Command{Pkg: pkg, File: file, Line: fset.Position(lit.Pos()).Line}
		for _, elt := range lit.Elts {
			kv, ok := elt.(*ast.KeyValueExpr)
			if !ok {
				continue
			}
			key, ok := kv.Key.(*ast.Ident)
			if !ok {
				continue
			}
			switch key.Name {
			case "Name", "Use":
				c.Name = stringLit(kv.Value)
			case "Usage", "Short":
				c.Usage = stringLit(kv.Value)
			case "Action", "RunE", "Run":
				c.Action = exprName(kv.Value)
			}
		}
		if c.Name != "" {
			out = append(out, c)
		}
		return true
	})
	return out
}

func stringLit(e ast.Expr) string {
	if b, ok := e.(*ast.BasicLit); ok && b.Kind == token.STRING {
		s, err := strconv.Unquote(b.Value)
		if err == nil {
			return s
		}
	}
	return ""
}

func exprName(e ast.Expr) string {
	switch v := e.(type) {
	case *ast.Ident:
		return v.Name
	case *ast.SelectorExpr:
		return exprName(v.X) + "." + v.Sel.Name
	case *ast.FuncLit:
		name := "func literal"
		ast.Inspect(v.Body, func(n ast.Node) bool {
			if _, ok := n.(*ast.FuncLit); ok {
				return false
			}
			if ret, ok := n.(*ast.ReturnStmt); ok && len(ret.Results) > 0 {
				if call, ok := ret.Results[0].(*ast.CallExpr); ok {
					if s := exprName(call.Fun); s != "" {
						name = s
					}
				}
			}
			return true
		})
		return name
	}
	return ""
}

func runRender(args []string) error {
	fs := flag.NewFlagSet("render", flag.ExitOnError)
	root := fs.String("root", ".", "module root")
	factsPath := fs.String("facts", "facts.json", "facts file from analyze")
	narrPath := fs.String("narrative", "narrative.json", "agent-written narrative file")
	tmplPath := fs.String("template", ".agents/skills/walkthrough-basic/assets/template.html", "HTML template")
	out := fs.String("out", "walkthrough.html", "output HTML file")
	inlineAll := fs.Bool("inline-all", false, "inline every package source file, not only referenced ones")
	fs.Parse(args)
	var facts Facts
	if err := readJSON(*factsPath, &facts); err != nil {
		return err
	}
	raw, err := os.ReadFile(*narrPath)
	if err != nil {
		return err
	}
	raw, err = expandTokens(*root, raw, &facts)
	if err != nil {
		return err
	}
	var narr Narrative
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&narr); err != nil {
		return fmt.Errorf("%s: %w", *narrPath, err)
	}
	refs, warnings, err := validate(*root, &facts, &narr, raw)
	for _, w := range warnings {
		fmt.Fprintln(os.Stderr, "warning:", w)
	}
	if err != nil {
		return err
	}
	if *inlineAll {
		for _, p := range facts.Packages {
			for _, f := range p.Files {
				refs = append(refs, path.Join(p.ID, f.Name))
			}
		}
	}
	files, err := collectFiles(*root, &facts, refs)
	if err != nil {
		return err
	}
	data, err := json.Marshal(struct {
		Facts
		Narrative
		Files map[string]File `json:"files"`
	}{facts, narr, files})
	if err != nil {
		return err
	}
	tmpl, err := os.ReadFile(*tmplPath)
	if err != nil {
		return err
	}
	if bytes.Count(tmpl, []byte(dataPlaceholder)) != 1 {
		return fmt.Errorf("%s must contain %s exactly once", *tmplPath, dataPlaceholder)
	}
	html := bytes.Replace(tmpl, []byte(dataPlaceholder), data, 1)
	if err := os.WriteFile(*out, html, 0o644); err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "wrote %s (%d KB, %d files inlined, %d tours, %d flows, %d findings)\n",
		*out, len(html)/1024, len(files), len(narr.Tours), len(narr.Flows), len(narr.Findings))
	return nil
}

var tokenRE = regexp.MustCompile(`\{\{(\w+)(?::([^}]+))?\}\}`)

// expandTokens replaces {{name}} and {{name:arg}} tokens with facts so that
// the narrative never hardcodes numbers.
func expandTokens(root string, raw []byte, f *Facts) ([]byte, error) {
	byID := map[string]Package{}
	for _, p := range f.Packages {
		byID[p.ID] = p
	}
	var errs []string
	out := tokenRE.ReplaceAllFunc(raw, func(m []byte) []byte {
		sub := tokenRE.FindSubmatch(m)
		name, arg := string(sub[1]), string(sub[2])
		p, hasPkg := byID[arg]
		var v string
		switch {
		case name == "packages":
			v = strconv.Itoa(len(f.Packages))
		case name == "totalLoc":
			v = strconv.Itoa(f.TotalLOC)
		case name == "commands":
			v = strconv.Itoa(len(f.Commands))
		case name == "sha":
			v = f.ShortSHA
		case name == "goVersion":
			v = f.GoVersion
		case name == "loc" && hasPkg:
			v = strconv.Itoa(p.LOC)
		case name == "files" && hasPkg:
			v = strconv.Itoa(len(p.Files))
		case name == "imports" && hasPkg:
			v = strconv.Itoa(len(p.Imports))
		case name == "importedBy" && hasPkg:
			v = strconv.Itoa(len(p.ImportedBy))
		case name == "lines":
			b, err := os.ReadFile(filepath.Join(root, arg))
			if err != nil {
				errs = append(errs, fmt.Sprintf("token %s: %v", m, err))
				return m
			}
			v = strconv.Itoa(countLines(b))
		default:
			errs = append(errs, fmt.Sprintf("unknown token or package in %s", m))
			return m
		}
		return []byte(v)
	})
	if len(errs) > 0 {
		return nil, errors.New(strings.Join(errs, "\n"))
	}
	return out, nil
}

var srcRefRE = regexp.MustCompile(`data-src=\\?["']([^"'\\]+)\\?["']`)
var pkgRefRE = regexp.MustCompile(`data-pkg=\\?["']([^"'\\]+)\\?["']`)

// validate checks that every package and file the narrative mentions exists,
// and that every package is placed on the map exactly once.
func validate(root string, f *Facts, n *Narrative, raw []byte) (refs, warnings []string, err error) {
	var errs []string
	pkgs := map[string]bool{}
	for _, p := range f.Packages {
		pkgs[p.ID] = true
	}
	checkPkg := func(where, id string) {
		if id != "" && !pkgs[id] {
			errs = append(errs, fmt.Sprintf("%s: unknown package %q", where, id))
		}
	}
	layerOf := map[string]int{}
	for i, l := range n.Layers {
		for _, id := range l.Packages {
			checkPkg("layer "+l.ID, id)
			if _, dup := layerOf[id]; dup {
				errs = append(errs, fmt.Sprintf("package %q is in more than one layer", id))
			}
			layerOf[id] = i
		}
	}
	var undescribed []string
	for _, p := range f.Packages {
		if _, ok := layerOf[p.ID]; !ok {
			errs = append(errs, fmt.Sprintf("package %q is not assigned to a layer", p.ID))
		}
		if _, ok := n.Descriptions[p.ID]; !ok {
			undescribed = append(undescribed, p.ID)
		}
		for _, imp := range p.Imports {
			li, lok := layerOf[p.ID]
			lj, rok := layerOf[imp]
			if lok && rok && lj < li && !p.TestOnly && n.Layers[li].ID != "test" {
				warnings = append(warnings, fmt.Sprintf("upward import: %s (%s) imports %s (%s)",
					p.ID, n.Layers[li].ID, imp, n.Layers[lj].ID))
			}
		}
	}
	if len(undescribed) > 0 {
		warnings = append(warnings, fmt.Sprintf("%d packages have no description, doc comments are used: %s",
			len(undescribed), strings.Join(undescribed, ", ")))
	}
	for id := range n.Descriptions {
		checkPkg("descriptions", id)
	}
	for _, t := range n.Tours {
		if len(t.Steps) == 0 {
			errs = append(errs, fmt.Sprintf("tour %q has no steps", t.ID))
		}
		for _, s := range t.Steps {
			checkPkg("tour "+t.ID, s.Pkg)
			refs = append(refs, s.Look...)
		}
	}
	for _, fl := range n.Flows {
		refs = append(refs, fl.File)
	}
	for _, fd := range n.Findings {
		checkPkg("finding "+fd.T, fd.Pkg)
		refs = append(refs, fd.F)
	}
	for _, m := range srcRefRE.FindAllSubmatch(raw, -1) {
		refs = append(refs, string(m[1]))
	}
	for _, m := range pkgRefRE.FindAllSubmatch(raw, -1) {
		checkPkg("data-pkg", string(m[1]))
	}
	refs = dedupe(refs)
	for _, r := range refs {
		if r == "" {
			errs = append(errs, "empty file reference")
			continue
		}
		info, err := os.Stat(filepath.Join(root, r))
		switch {
		case err != nil:
			errs = append(errs, fmt.Sprintf("file reference %q does not exist", r))
		case info.IsDir() && !strings.HasSuffix(r, "/"):
			errs = append(errs, fmt.Sprintf("directory reference %q must end with /", r))
		}
	}
	if len(errs) > 0 {
		return nil, warnings, fmt.Errorf("narrative is invalid:\n  %s", strings.Join(errs, "\n  "))
	}
	return refs, warnings, nil
}

func collectFiles(root string, f *Facts, refs []string) (map[string]File, error) {
	files := map[string]File{}
	pkgOfDir := map[string]string{}
	for _, p := range f.Packages {
		pkgOfDir[p.ID+"/"] = p.ID
		refs = append(refs, p.ID+"/")
	}
	for _, r := range dedupe(refs) {
		full := filepath.Join(root, r)
		if strings.HasSuffix(r, "/") {
			entries, err := os.ReadDir(full)
			if err != nil {
				return nil, err
			}
			d := File{Dir: true, Pkg: pkgOfDir[r]}
			for _, e := range entries {
				if strings.HasPrefix(e.Name(), ".") {
					continue
				}
				if e.IsDir() {
					d.Entries = append(d.Entries, e.Name()+"/")
					continue
				}
				b, err := os.ReadFile(filepath.Join(full, e.Name()))
				if err != nil {
					return nil, err
				}
				n := countLines(b)
				d.Lines += n
				d.Entries = append(d.Entries, fmt.Sprintf("%s (%d lines)", e.Name(), n))
			}
			files[r] = d
			continue
		}
		b, err := os.ReadFile(full)
		if err != nil {
			return nil, err
		}
		lines := strings.Split(string(b), "\n")
		file := File{Lines: countLines(b), Pkg: pkgOfDir[path.Dir(r)+"/"]}
		if len(lines) > maxInlineLines {
			lines, file.Trunc = lines[:maxInlineLines], true
		}
		file.Content = strings.Join(lines, "\n")
		files[r] = file
	}
	return files, nil
}

func internalImports(imports []string, module string) []string {
	var out []string
	for _, imp := range imports {
		switch {
		case imp == module:
			out = append(out, ".")
		case strings.HasPrefix(imp, module+"/"):
			out = append(out, strings.TrimPrefix(imp, module+"/"))
		}
	}
	return nonNil(dedupe(out))
}

func skipDir(name string) bool {
	return strings.HasPrefix(name, ".") || strings.HasPrefix(name, "_") ||
		name == "testdata" || name == "vendor" || name == "node_modules"
}

func githubRepo(remote string) string {
	m := regexp.MustCompile(`github\.com[:/]([^/]+/[^/]+?)(\.git)?$`).FindStringSubmatch(remote)
	if m == nil {
		return ""
	}
	return m[1]
}

func gitOut(dir string, args ...string) string {
	out, err := exec.Command("git", append([]string{"-C", dir}, args...)...).Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func countLines(b []byte) int {
	n := bytes.Count(b, []byte("\n"))
	if len(b) > 0 && b[len(b)-1] != '\n' {
		n++
	}
	return n
}

func dedupe(in []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, s := range in {
		if !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	return out
}

func nonNil(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}

func mustRel(root, p string) string {
	rel, err := filepath.Rel(root, p)
	if err != nil {
		panic(err)
	}
	return rel
}

func fileExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}

func readJSON(p string, v any) error {
	b, err := os.ReadFile(p)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, v)
}
