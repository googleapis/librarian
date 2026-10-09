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
//
// Every file is read from git objects at HEAD, never from the work tree, so
// the output depends only on the commit: untracked, ignored, staged-only and
// uncommitted content, symlinks, and files outside the repository can never
// be read.
package main

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"go/ast"
	"go/build"
	"go/doc"
	"go/parser"
	"go/token"
	"go/types"
	"html"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	dataPlaceholder = "__WALKTHROUGH_DATA__"
	hashPlaceholder = "__WALKTHROUGH_SCRIPT_HASH__"
	maxExported     = 60
	maxInlineLines  = 600
	maxReadBytes    = 8 << 20
)

// Facts is everything that can be derived mechanically from the source tree.
type Facts struct {
	Repo          string     `json:"repo"`
	Module        string     `json:"module"`
	GoVersion     string     `json:"goVersion"`
	SHA           string     `json:"sha"`
	ShortSHA      string     `json:"shortSha"`
	Ref           string     `json:"ref"`
	Tag           string     `json:"tag"`
	CommitDate    string     `json:"commitDate"`
	Dirty         bool       `json:"dirty"`
	Platform      string     `json:"platform"`
	TotalLOC      int        `json:"totalLoc"`
	Packages      []Package  `json:"packages"`
	Commands      []Command  `json:"commands"`
	Docs          []string   `json:"docs"`
	NestedModules []string   `json:"nestedModules"`
	Contracts     []Contract `json:"contracts"`
}

// Contract is an implicit interface: exported functions that several sibling
// packages (same parent directory) each define, with the code that calls them.
type Contract struct {
	Parent  string   `json:"parent"`
	Members []string `json:"members"`
	Hooks   []Hook   `json:"hooks"`
}

// Hook is one function of a Contract.
type Hook struct {
	Name    string   `json:"name"`
	Sig     string   `json:"sig"`
	Members []string `json:"members"`
	Callers []string `json:"callers"`
}

// Package describes one Go package in the module.
type Package struct {
	ID         string            `json:"id"`
	Name       string            `json:"name"`
	Doc        string            `json:"doc,omitempty"`
	LOC        int               `json:"loc"`
	Files      []FileStat        `json:"files"`
	Tests      int               `json:"tests"`
	Imports    []string          `json:"imports"`
	ImportedBy []string          `json:"importedBy"`
	TestOnly   bool              `json:"testOnly"`
	Exported   []string          `json:"exported"`
	MoreExport int               `json:"moreExported,omitempty"`
	TestDeps   []string          `json:"-"`
	funcs      map[string]string // exported top-level function name -> signature
	refs       []fileRefs
}

// fileRefs records, for one file, how imported packages are named and which
// qualified references (pkg.Func) each top-level function makes.
type fileRefs struct {
	named   map[string]string // import alias -> package ID
	unnamed []string          // package IDs imported without an alias
	calls   []call
}

type call struct{ from, qual, sel string }

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
	Title         string            `json:"title"`
	Summary       string            `json:"summary"`
	Layers        []Layer           `json:"layers"`
	Descriptions  map[string]string `json:"descriptions"`
	Tour          *Tour             `json:"tour"`
	GuidesHeading string            `json:"guidesHeading"`
	Guides        []Guide           `json:"guides"`
	Flows         []Flow            `json:"flows"`
	Families      []Family          `json:"families"`
	Findings      []Finding         `json:"findings"`
}

// Layer is an architectural tier on the map.
type Layer struct {
	ID       string   `json:"id"`
	Title    string   `json:"title"`
	Hint     string   `json:"hint"`
	Packages []string `json:"packages"`
}

// Tour is the end-to-end guided tour.
type Tour struct {
	Title    string `json:"title"`
	Subtitle string `json:"subtitle"`
	Steps    []Step `json:"steps"`
}

// Guide is a focused deep dive for a specific audience.
type Guide struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Subtitle string `json:"subtitle"`
	Audience string `json:"audience"`
	Steps    []Step `json:"steps"`
}

// Step is a single tour or guide step.
type Step struct {
	T    string   `json:"t"`
	Runs string   `json:"runs,omitempty"`
	Code string   `json:"code,omitempty"`
	Body string   `json:"body"`
	Look []string `json:"look"`
	Pkg  string   `json:"pkg,omitempty"`
}

// Flow traces one command through the code in stages.
type Flow struct {
	ID      string   `json:"id"`
	Title   string   `json:"title"`
	Cmd     string   `json:"cmd"`
	File    string   `json:"file"`
	Inputs  []string `json:"inputs"`
	Stages  []Stage  `json:"stages"`
	Outputs []string `json:"outputs"`
}

// Stage is one hop of a Flow.
type Stage struct {
	Title string `json:"title"`
	Where string `json:"where"`
	Body  string `json:"body"`
	Out   string `json:"out,omitempty"`
}

// Family describes the members of a computed Contract in a comparison table.
type Family struct {
	Parent  string               `json:"parent"`
	Title   string               `json:"title"`
	Intro   string               `json:"intro"`
	Columns []string             `json:"columns"`
	Rows    map[string]FamilyRow `json:"rows"`
}

// FamilyRow is one member's cells and expandable detail.
type FamilyRow struct {
	Cells  []string `json:"cells"`
	Detail string   `json:"detail"`
}

// Finding is an architectural observation backed by a file.
type Finding struct {
	T   string `json:"t"`
	B   string `json:"b"`
	F   string `json:"f"`
	Pkg string `json:"pkg,omitempty"`
}

// Check is a finding computed from facts and layers at render time.
type Check struct {
	Title  string     `json:"title"`
	Detail string     `json:"detail"`
	Rows   [][]string `json:"rows"`
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

// repo is a read-only view of the git tree at HEAD below root.
type repo struct {
	root  string
	blobs map[string]string   // regular file path relative to root -> blob id
	dirs  map[string][]string // directory ("." for root) -> sorted entries; subdirectories end in "/"
	cache map[string][]byte
	cat   *exec.Cmd
	in    io.WriteCloser
	out   *bufio.Reader
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

// gitArgs disables repository-configured hooks and file system monitors, so
// running git inside an untrusted checkout cannot execute its commands.
func gitArgs(root string, args ...string) []string {
	return append([]string{"-c", "core.fsmonitor=false", "-c", "core.hooksPath=/dev/null", "-C", root}, args...)
}

func openRepo(root string) (*repo, error) {
	root, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	out, err := exec.Command("git", gitArgs(root, "ls-tree", "-r", "-z", "HEAD")...).Output()
	if err != nil {
		return nil, fmt.Errorf("%s must be inside a git work tree with a HEAD commit: %w", root, err)
	}
	r := &repo{root: root, blobs: map[string]string{}, dirs: map[string][]string{".": nil}, cache: map[string][]byte{}}
	seen := map[string]bool{}
	add := func(dir, entry string) {
		if !seen[dir+"\x00"+entry] {
			seen[dir+"\x00"+entry] = true
			r.dirs[dir] = append(r.dirs[dir], entry)
		}
	}
	for _, rec := range strings.Split(string(out), "\x00") {
		meta, p, ok := strings.Cut(rec, "\t")
		fields := strings.Fields(meta)
		// Keep regular files only; symlinks (120000) and submodules are skipped.
		if !ok || len(fields) != 3 || fields[1] != "blob" || (fields[0] != "100644" && fields[0] != "100755") {
			continue
		}
		r.blobs[p] = fields[2]
		add(path.Dir(p), path.Base(p))
		for d := path.Dir(p); d != "."; d = path.Dir(d) {
			add(path.Dir(d), path.Base(d)+"/")
		}
	}
	for d := range r.dirs {
		sort.Strings(r.dirs[d])
	}
	return r, nil
}

func (r *repo) git(args ...string) string {
	out, err := exec.Command("git", gitArgs(r.root, args...)...).Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// read returns a file's content at HEAD, at most maxReadBytes, and whether it
// was cut short. A single "git cat-file --batch" process serves all reads.
func (r *repo) read(rel string) ([]byte, bool, error) {
	id, ok := r.blobs[rel]
	if !ok {
		return nil, false, fmt.Errorf("%s is not a file at HEAD", rel)
	}
	if b, ok := r.cache[rel]; ok {
		return b, len(b) == maxReadBytes, nil
	}
	if r.cat == nil {
		r.cat = exec.Command("git", gitArgs(r.root, "cat-file", "--batch")...)
		var err error
		if r.in, err = r.cat.StdinPipe(); err != nil {
			return nil, false, err
		}
		stdout, err := r.cat.StdoutPipe()
		if err != nil {
			return nil, false, err
		}
		r.out = bufio.NewReader(stdout)
		if err := r.cat.Start(); err != nil {
			return nil, false, err
		}
	}
	if _, err := fmt.Fprintln(r.in, id); err != nil {
		return nil, false, err
	}
	header, err := r.out.ReadString('\n')
	if err != nil {
		return nil, false, err
	}
	var gotID, kind string
	var size int64
	if _, err := fmt.Sscan(header, &gotID, &kind, &size); err != nil || kind != "blob" {
		return nil, false, fmt.Errorf("git cat-file %s: unexpected header %q", rel, header)
	}
	n := min(size, maxReadBytes)
	b := make([]byte, n)
	if _, err := io.ReadFull(r.out, b); err != nil {
		return nil, false, err
	}
	if _, err := io.CopyN(io.Discard, r.out, size-n+1); err != nil { // rest plus trailing newline
		return nil, false, err
	}
	r.cache[rel] = b
	return b, size > maxReadBytes, nil
}

var refRE = regexp.MustCompile(`^[A-Za-z0-9_.@+-][A-Za-z0-9_./@+-]*$`)

// resolve validates a narrative reference and returns it relative to root
// without a trailing slash. A reference must be a clean relative path, use a
// conservative character set, name a regular file at HEAD (or a directory
// when it ends in "/"), and not look like a credential file.
func (r *repo) resolve(ref string) (string, error) {
	isDir := strings.HasSuffix(ref, "/")
	rel := strings.TrimSuffix(ref, "/")
	if rel == "" || rel == "." {
		rel = "."
	} else if !refRE.MatchString(rel) || path.Clean(rel) != rel || rel == ".." || strings.HasPrefix(rel, "../") {
		return "", fmt.Errorf("reference %q must be a clean relative path using only letters, digits and _ . / @ + -", ref)
	}
	if isSecretName(path.Base(rel)) {
		return "", fmt.Errorf("reference %q looks like a credential or secret file", ref)
	}
	_, isFile := r.blobs[rel]
	_, isDirAtHead := r.dirs[rel]
	switch {
	case isDir && isDirAtHead, !isDir && isFile:
		return rel, nil
	case !isDir && isDirAtHead:
		return "", fmt.Errorf("directory reference %q must end with /", ref)
	}
	return "", fmt.Errorf("reference %q is not a regular file or directory at HEAD", ref)
}

// isSecretName reports whether a file name looks like it holds credentials.
func isSecretName(name string) bool {
	n := strings.ToLower(name)
	for _, p := range []string{".env", "id_rsa", "id_dsa", "id_ecdsa", "id_ed25519", "credentials", ".git-credentials"} {
		if strings.HasPrefix(n, p) {
			return true
		}
	}
	for _, sfx := range []string{".pem", ".key", ".p12", ".pfx", ".jks", ".keystore", ".kdbx"} {
		if strings.HasSuffix(n, sfx) {
			return true
		}
	}
	switch n {
	case ".npmrc", ".netrc", ".pypirc", ".htpasswd", ".dockercfg", "secrets.yaml", "secrets.yml", "secrets.json":
		return true
	}
	return false
}

func runAnalyze(args []string) error {
	fset := flag.NewFlagSet("analyze", flag.ExitOnError)
	root := fset.String("root", ".", "module root")
	out := fset.String("out", "facts.json", "output facts file")
	goos := fset.String("goos", "linux", "GOOS used to select files by build constraints")
	goarch := fset.String("goarch", "amd64", "GOARCH used to select files by build constraints")
	fset.Parse(args)
	r, err := openRepo(*root)
	if err != nil {
		return err
	}
	facts, err := analyze(r, *goos, *goarch)
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
	fmt.Fprintf(os.Stderr, "%s @ %s (dirty=%t, %s): %d packages, %d LOC, %d commands -> %s\n",
		facts.Module, facts.ShortSHA, facts.Dirty, facts.Platform, len(facts.Packages), facts.TotalLOC, len(facts.Commands), *out)
	return nil
}

func analyze(r *repo, goos, goarch string) (*Facts, error) {
	gomod, _, err := r.read("go.mod")
	if err != nil {
		return nil, fmt.Errorf("go.mod at HEAD: %w", err)
	}
	f := &Facts{
		SHA:        r.git("rev-parse", "HEAD"),
		Ref:        r.git("rev-parse", "--abbrev-ref", "HEAD"),
		Tag:        r.git("describe", "--tags", "--abbrev=0"),
		CommitDate: r.git("show", "-s", "--format=%cI", "HEAD"),
		Dirty:      r.git("status", "--porcelain", "--untracked-files=no") != "",
		Repo:       githubRepo(r.git("remote", "get-url", "origin")),
		Platform:   goos + "/" + goarch,
	}
	if len(f.SHA) >= 8 {
		f.ShortSHA = f.SHA[:8]
	}
	for _, line := range strings.Split(string(gomod), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 2 && fields[0] == "module" && moduleRE.MatchString(fields[1]) {
			f.Module = fields[1]
		}
		if len(fields) == 2 && fields[0] == "go" {
			f.GoVersion = "Go " + fields[1]
		}
	}
	ctx := build.Default
	ctx.GOOS, ctx.GOARCH, ctx.CgoEnabled, ctx.BuildTags = goos, goarch, true, nil
	ctx.ReadDir = r.readDir
	ctx.OpenFile = func(p string) (io.ReadCloser, error) {
		b, _, err := r.read(r.rel(p))
		return io.NopCloser(bytes.NewReader(b)), err
	}
	ctx.IsDir = func(p string) bool { _, ok := r.dirs[r.rel(p)]; return ok }
	goDirs := map[string]bool{}
	for p := range r.blobs {
		switch {
		case skipPath(p):
		case path.Base(p) == "go.mod" && p != "go.mod":
			f.NestedModules = append(f.NestedModules, path.Dir(p))
		case strings.HasSuffix(p, ".go"):
			goDirs[path.Dir(p)] = true
		case strings.HasSuffix(p, ".md") && strings.Count(p, "/") <= 2:
			f.Docs = append(f.Docs, p)
		}
	}
	sort.Strings(f.NestedModules)
	sort.Strings(f.Docs)
	f.Docs = nonNil(f.Docs)
	f.NestedModules = nonNil(f.NestedModules)
	dirs := make([]string, 0, len(goDirs))
	for d := range goDirs {
		if !inModule(d, f.NestedModules) {
			dirs = append(dirs, d)
		}
	}
	sort.Strings(dirs)
	for _, d := range dirs {
		pkg, cmds, err := loadPackage(&ctx, r, d, f.Module)
		if err != nil {
			return nil, err
		}
		if pkg != nil {
			f.Packages = append(f.Packages, *pkg)
			f.Commands = append(f.Commands, cmds...)
		}
	}
	importedBy := map[string][]string{}
	testUsers := map[string]bool{}
	for _, p := range f.Packages {
		for _, imp := range p.Imports {
			importedBy[imp] = append(importedBy[imp], p.ID)
		}
		for _, imp := range p.TestDeps {
			testUsers[imp] = true
		}
	}
	for i := range f.Packages {
		p := &f.Packages[i]
		p.ImportedBy = nonNil(importedBy[p.ID])
		p.TestOnly = p.Name != "main" && testUsers[p.ID] && len(p.ImportedBy) == 0
		f.TotalLOC += p.LOC
	}
	f.Commands = nonNil(f.Commands)
	f.Contracts = contracts(f.Packages)
	return f, nil
}

// contracts finds groups of at least three sibling packages and the exported
// functions that two or more of them define, plus every caller of those
// functions anywhere in the module (calls or function-value references).
func contracts(pkgs []Package) []Contract {
	nameOf := map[string]string{}
	for _, p := range pkgs {
		nameOf[p.ID] = p.Name
	}
	callers := map[string]map[string]bool{} // "pkgID.Func" -> callers
	for _, p := range pkgs {
		for _, fr := range p.refs {
			alias := map[string]string{}
			for _, id := range fr.unnamed {
				alias[nameOf[id]] = id
			}
			for a, id := range fr.named {
				alias[a] = id
			}
			for _, c := range fr.calls {
				if id, ok := alias[c.qual]; ok {
					key := id + "." + c.sel
					if callers[key] == nil {
						callers[key] = map[string]bool{}
					}
					callers[key][p.ID+"."+c.from] = true
				}
			}
		}
	}
	groups := map[string][]Package{}
	for _, p := range pkgs {
		if p.ID != "." && p.Name != "main" && !p.TestOnly {
			groups[path.Dir(p.ID)] = append(groups[path.Dir(p.ID)], p)
		}
	}
	var out []Contract
	for _, parent := range sortedKeys(groups) {
		members := groups[parent]
		if len(members) < 3 {
			continue
		}
		c := Contract{Parent: parent}
		byName := map[string]*Hook{}
		for _, m := range members {
			c.Members = append(c.Members, m.ID)
			for name, sig := range m.funcs {
				h := byName[name]
				if h == nil {
					h = &Hook{Name: name, Sig: sig}
					byName[name] = h
				}
				h.Members = append(h.Members, m.ID)
				for caller := range callers[m.ID+"."+name] {
					h.Callers = append(h.Callers, caller)
				}
			}
		}
		for _, h := range byName {
			if len(h.Members) < 2 {
				continue
			}
			sort.Strings(h.Members)
			h.Callers = nonNil(dedupe(sortedStrings(h.Callers)))
			c.Hooks = append(c.Hooks, *h)
		}
		if len(c.Hooks) == 0 {
			continue
		}
		sort.Slice(c.Hooks, func(i, j int) bool {
			a, b := c.Hooks[i], c.Hooks[j]
			if len(a.Members) != len(b.Members) {
				return len(a.Members) > len(b.Members)
			}
			return a.Name < b.Name
		})
		out = append(out, c)
	}
	return nonNil(out)
}

// fileInfo describes a file at HEAD for go/build.
type fileInfo struct{ name string }

func (fi fileInfo) Name() string       { return fi.name }
func (fi fileInfo) Size() int64        { return 0 }
func (fi fileInfo) Mode() fs.FileMode  { return 0o644 }
func (fi fileInfo) ModTime() time.Time { return time.Time{} }
func (fi fileInfo) IsDir() bool        { return false }
func (fi fileInfo) Sys() any           { return nil }

func (r *repo) rel(abs string) string {
	return filepath.ToSlash(mustRel(r.root, abs))
}

// readDir lists the regular files of a directory at HEAD for go/build.
func (r *repo) readDir(dir string) ([]fs.FileInfo, error) {
	entries, ok := r.dirs[r.rel(dir)]
	if !ok {
		return nil, fmt.Errorf("%s is not a directory at HEAD", dir)
	}
	var out []fs.FileInfo
	for _, e := range entries {
		if !strings.HasSuffix(e, "/") {
			out = append(out, fileInfo{e})
		}
	}
	return out, nil
}

func loadPackage(ctx *build.Context, r *repo, id, module string) (*Package, []Command, error) {
	dir := filepath.Join(r.root, filepath.FromSlash(id))
	bp, err := ctx.ImportDir(dir, build.ImportComment)
	if err != nil {
		var noGo *build.NoGoError
		var multi *build.MultiplePackageError
		if errors.As(err, &noGo) || errors.As(err, &multi) {
			return nil, nil, nil
		}
		return nil, nil, fmt.Errorf("%s: %w", id, err)
	}
	if len(bp.GoFiles) == 0 {
		return nil, nil, nil
	}
	pkg := &Package{
		ID:       id,
		Name:     bp.Name,
		Doc:      bp.Doc,
		Tests:    len(bp.TestGoFiles) + len(bp.XTestGoFiles),
		Imports:  internalImports(bp.Imports, module),
		TestDeps: internalImports(append(bp.TestImports, bp.XTestImports...), module),
		funcs:    map[string]string{},
	}
	fset := token.NewFileSet()
	var cmds []Command
	var exported []string
	for _, name := range bp.GoFiles {
		src, _, err := r.read(path.Join(id, name))
		if err != nil {
			return nil, nil, err
		}
		lines := countLines(src)
		pkg.LOC += lines
		pkg.Files = append(pkg.Files, FileStat{Name: name, Lines: lines})
		file, err := parser.ParseFile(fset, name, src, parser.SkipObjectResolution|parser.ParseComments)
		if err != nil {
			return nil, nil, err
		}
		if pkg.Doc == "" && file.Doc != nil {
			pkg.Doc = new(doc.Package).Synopsis(file.Doc.Text())
		}
		exported = append(exported, exportedDecls(file)...)
		for name, sig := range exportedFuncs(file) {
			pkg.funcs[name] = sig
		}
		pkg.refs = append(pkg.refs, fileCalls(file, module))
		cmds = append(cmds, findCommands(fset, file, id, path.Join(id, name))...)
	}
	sort.Strings(exported)
	if len(exported) > maxExported {
		pkg.MoreExport = len(exported) - maxExported
		exported = exported[:maxExported]
	}
	pkg.Exported = nonNil(exported)
	return pkg, cmds, nil
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

// exportedFuncs returns exported top-level functions with a short signature
// built from parameter names (or types when unnamed).
func exportedFuncs(f *ast.File) map[string]string {
	out := map[string]string{}
	for _, decl := range f.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Recv != nil || !fn.Name.IsExported() {
			continue
		}
		var params []string
		for _, field := range fn.Type.Params.List {
			if len(field.Names) == 0 {
				params = append(params, types.ExprString(field.Type))
			}
			for _, n := range field.Names {
				params = append(params, n.Name)
			}
		}
		out[fn.Name.Name] = fn.Name.Name + "(" + strings.Join(params, ", ") + ")"
	}
	return out
}

// fileCalls records the file's internal imports and every qualified
// reference (x.F, called or passed as a value) inside each top-level function
// (including closures) or package-level variable initializer.
func fileCalls(f *ast.File, module string) fileRefs {
	fr := fileRefs{named: map[string]string{}}
	for _, spec := range f.Imports {
		ip, err := strconv.Unquote(spec.Path.Value)
		if err != nil {
			continue
		}
		ids := internalImports([]string{ip}, module)
		if len(ids) == 0 {
			continue
		}
		if spec.Name != nil {
			fr.named[spec.Name.Name] = ids[0]
		} else {
			fr.unnamed = append(fr.unnamed, ids[0])
		}
	}
	record := func(from string, node ast.Node) {
		ast.Inspect(node, func(n ast.Node) bool {
			if sel, ok := n.(*ast.SelectorExpr); ok {
				if x, ok := sel.X.(*ast.Ident); ok {
					fr.calls = append(fr.calls, call{from: from, qual: x.Name, sel: sel.Sel.Name})
				}
			}
			return true
		})
	}
	for _, decl := range f.Decls {
		switch d := decl.(type) {
		case *ast.FuncDecl:
			if d.Body == nil {
				continue
			}
			from := d.Name.Name
			if d.Recv != nil && len(d.Recv.List) == 1 {
				from = strings.TrimPrefix(types.ExprString(d.Recv.List[0].Type), "*") + "." + from
			}
			record(from, d.Body)
		case *ast.GenDecl:
			for _, spec := range d.Specs {
				if vs, ok := spec.(*ast.ValueSpec); ok && len(vs.Names) > 0 {
					for _, v := range vs.Values {
						record(vs.Names[0].Name, v)
					}
				}
			}
		}
	}
	return fr
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
		switch t := lit.Type.(type) {
		case *ast.SelectorExpr:
			if t.Sel.Name != "Command" {
				return true
			}
		case *ast.Ident:
			if t.Name != "Command" {
				return true
			}
		default:
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
		if s, err := strconv.Unquote(b.Value); err == nil {
			return s
		}
	}
	return ""
}

// errorCtors are calls that build an error rather than doing the work.
var errorCtors = map[string]bool{"fmt.Errorf": true, "errors.New": true, "errors.Join": true}

// exprName names a command action. For an inline function literal it names
// the call in the literal's final top-level return statement, which is
// usually the real implementation; otherwise the logic is inline.
// A final error constructor means the work happened earlier, inline.
func exprName(e ast.Expr) string {
	switch v := e.(type) {
	case *ast.Ident:
		return v.Name
	case *ast.SelectorExpr:
		return exprName(v.X) + "." + v.Sel.Name
	case *ast.FuncLit:
		if n := len(v.Body.List); n > 0 {
			if ret, ok := v.Body.List[n-1].(*ast.ReturnStmt); ok && len(ret.Results) == 1 {
				if call, ok := ret.Results[0].(*ast.CallExpr); ok {
					if s := exprName(call.Fun); s != "" && !errorCtors[s] {
						return s
					}
				}
			}
		}
		return "inline"
	}
	return ""
}

// defaultTemplate locates the template next to this script's own source file,
// never relative to the analyzed repository, so a repository cannot supply its
// own template.
func defaultTemplate() string {
	_, file, _, ok := runtime.Caller(0)
	if !ok || !filepath.IsAbs(file) {
		return ""
	}
	return filepath.Join(filepath.Dir(file), "..", "assets", "template.html")
}

func runRender(args []string) error {
	fset := flag.NewFlagSet("render", flag.ExitOnError)
	root := fset.String("root", ".", "module root")
	factsPath := fset.String("facts", "facts.json", "facts file from analyze")
	narrPath := fset.String("narrative", "narrative.json", "agent-written narrative file")
	tmplPath := fset.String("template", defaultTemplate(), "HTML template (defaults to ../assets/template.html next to this script)")
	out := fset.String("out", "walkthrough.html", "output HTML file")
	inlineAll := fset.Bool("inline-all", false, "inline every package source file, not only referenced ones")
	fset.Parse(args)
	r, err := openRepo(*root)
	if err != nil {
		return err
	}
	var facts Facts
	if err := readJSON(*factsPath, &facts); err != nil {
		return err
	}
	if head := r.git("rev-parse", "HEAD"); head != facts.SHA {
		return fmt.Errorf("facts were extracted at %s but HEAD is %s; re-run analyze", facts.SHA, head)
	}
	raw, err := os.ReadFile(*narrPath)
	if err != nil {
		return err
	}
	raw, err = expandTokens(r, raw, &facts)
	if err != nil {
		return err
	}
	var narr Narrative
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&narr); err != nil {
		return fmt.Errorf("%s: %w", *narrPath, err)
	}
	refs, warnings, err := validate(r, &facts, &narr)
	for _, w := range warnings {
		fmt.Fprintln(os.Stderr, "warning:", w)
	}
	if err != nil {
		return err
	}
	for _, c := range facts.Commands {
		refs = append(refs, c.File)
	}
	if *inlineAll {
		for _, p := range facts.Packages {
			for _, f := range p.Files {
				refs = append(refs, path.Join(p.ID, f.Name))
			}
		}
	}
	files, err := collectFiles(r, &facts, refs)
	if err != nil {
		return err
	}
	data, err := json.Marshal(struct {
		Facts
		Narrative
		Checks []Check         `json:"checks"`
		Files  map[string]File `json:"files"`
	}{facts, narr, computeChecks(&facts, &narr), files})
	if err != nil {
		return err
	}
	page, err := fillTemplate(*tmplPath, data)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(*out), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(*out, page, 0o644); err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "wrote %s (%d KB, %d files inlined, %d guides, %d flows, %d families, %d findings)\n",
		*out, len(page)/1024, len(files), len(narr.Guides), len(narr.Flows), len(narr.Families), len(narr.Findings))
	return nil
}

// fillTemplate injects the data and pins the application script in the
// Content-Security-Policy by hash, so no other script (including inline
// event handlers) can run and the page cannot make network requests.
func fillTemplate(tmplPath string, data []byte) ([]byte, error) {
	if tmplPath == "" {
		return nil, errors.New("cannot locate the template; pass -template")
	}
	tmpl, err := os.ReadFile(tmplPath)
	if err != nil {
		return nil, err
	}
	// Browsers normalize newlines before hashing inline scripts.
	tmpl = bytes.ReplaceAll(bytes.ReplaceAll(tmpl, []byte("\r\n"), []byte("\n")), []byte("\r"), []byte("\n"))
	for i, c := range tmpl {
		if c > 0x7f {
			return nil, fmt.Errorf("%s: non-ASCII byte at offset %d; use \\u escapes in script and entities in HTML so the script hash survives viewers that guess the wrong encoding", tmplPath, i)
		}
	}
	for _, ph := range []string{dataPlaceholder, hashPlaceholder} {
		if bytes.Count(tmpl, []byte(ph)) != 1 {
			return nil, fmt.Errorf("%s must contain %s exactly once", tmplPath, ph)
		}
	}
	const open, end = "<script>", "</script>"
	i := bytes.LastIndex(tmpl, []byte(open))
	j := bytes.LastIndex(tmpl, []byte(end))
	if i < 0 || j < i {
		return nil, fmt.Errorf("%s: no application <script> block", tmplPath)
	}
	sum := sha256.Sum256(tmpl[i+len(open) : j])
	page := bytes.Replace(tmpl, []byte(hashPlaceholder), []byte(base64.StdEncoding.EncodeToString(sum[:])), 1)
	// json.Marshal escapes <, > and &, so the data cannot close its <script>.
	return bytes.Replace(page, []byte(dataPlaceholder), asciiJSON(data), 1), nil
}

// asciiJSON rewrites non-ASCII characters in JSON as \u escapes, so the page
// is pure ASCII and decodes identically under any ASCII-compatible encoding.
func asciiJSON(b []byte) []byte {
	var out bytes.Buffer
	for _, r := range string(b) {
		switch {
		case r < 0x80:
			out.WriteRune(r)
		case r > 0xffff:
			r -= 0x10000
			fmt.Fprintf(&out, `\u%04x\u%04x`, 0xd800+(r>>10), 0xdc00+(r&0x3ff))
		default:
			fmt.Fprintf(&out, `\u%04x`, r)
		}
	}
	return out.Bytes()
}

var (
	githubRE = regexp.MustCompile(`^(?:https://github\.com/|git@github\.com:|ssh://git@github\.com/)([A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+?)(?:\.git)?/?$`)
	moduleRE = regexp.MustCompile(`^[A-Za-z0-9._~+/-]+$`)
	idRE     = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*$`)
)

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

var tokenRE = regexp.MustCompile(`\{\{(\w+)(?::([^}]+))?\}\}`)

// expandTokens replaces {{name}} and {{name:arg}} tokens with facts so that
// the narrative never hardcodes numbers.
func expandTokens(r *repo, raw []byte, f *Facts) ([]byte, error) {
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
		case name == "tag" && f.Tag != "":
			v = f.Tag
		case name == "tracked" && strings.HasSuffix(arg, "/"):
			if _, err := r.resolve(arg); err != nil {
				errs = append(errs, fmt.Sprintf("token %s: %v", m, err))
				return m
			}
			n := 0
			for p := range r.blobs {
				if strings.HasPrefix(p, arg) || arg == "./" {
					n++
				}
			}
			v = strconv.Itoa(n)
		case name == "loc" && hasPkg:
			v = strconv.Itoa(p.LOC)
		case name == "files" && hasPkg:
			v = strconv.Itoa(len(p.Files))
		case name == "imports" && hasPkg:
			v = strconv.Itoa(len(p.Imports))
		case name == "importedBy" && hasPkg:
			v = strconv.Itoa(len(p.ImportedBy))
		case name == "lines":
			rel, err := r.resolve(arg)
			if err == nil {
				var b []byte
				if b, _, err = r.read(rel); err == nil {
					v = strconv.Itoa(countLines(b))
					break
				}
			}
			errs = append(errs, fmt.Sprintf("token %s: %v", m, err))
			return m
		default:
			errs = append(errs, fmt.Sprintf("unknown token or package in %s", m))
			return m
		}
		// Values are spliced into raw JSON, so escape them as JSON string content.
		q, _ := json.Marshal(v)
		return q[1 : len(q)-1]
	})
	if len(errs) > 0 {
		return nil, errors.New(strings.Join(errs, "\n"))
	}
	return out, nil
}

var (
	allowedTagRE = regexp.MustCompile(`&lt;(/?)(p|code|strong|em|ul|ol|li)&gt;|&lt;()(br)&gt;`)
	linkRE       = regexp.MustCompile(`&lt;a class=&#34;(src|pkgref)&#34; data-(src|pkg)=&#34;([A-Za-z0-9_./-]+)&#34;&gt;`)
	srcAttrRE    = regexp.MustCompile(`data-src="([^"]+)"`)
	pkgAttrRE    = regexp.MustCompile(`data-pkg="([^"]+)"`)
)

// sanitize escapes all HTML and then re-enables only a fixed set of
// attribute-free tags and the two link forms the template understands.
func sanitize(s string) string {
	e := allowedTagRE.ReplaceAllString(html.EscapeString(s), "<$1$2$3$4>")
	e = linkRE.ReplaceAllStringFunc(e, func(m string) string {
		sub := linkRE.FindStringSubmatch(m)
		if (sub[1] == "src") != (sub[2] == "src") {
			return m
		}
		return fmt.Sprintf(`<a class="%s" data-%s="%s">`, sub[1], sub[2], sub[3])
	})
	return strings.ReplaceAll(e, "&lt;/a&gt;", "</a>")
}

var tagRE = regexp.MustCompile(`<(/?)(p|code|strong|em|ul|ol|li|a)[ >]`)

// balanced reports whether every allowed tag in sanitized HTML is closed in
// order, so a fragment cannot swallow the page markup that follows it.
func balanced(s string) bool {
	var stack []string
	for _, m := range tagRE.FindAllStringSubmatch(s, -1) {
		if m[1] == "" {
			stack = append(stack, m[2])
			continue
		}
		if len(stack) == 0 || stack[len(stack)-1] != m[2] {
			return false
		}
		stack = stack[:len(stack)-1]
	}
	return len(stack) == 0
}

// htmlFields returns every narrative field rendered as HTML, except map
// values (descriptions and family rows), which validate handles directly.
func htmlFields(n *Narrative) []*string {
	fields := []*string{&n.Summary}
	steps := func(ss []Step) {
		for i := range ss {
			fields = append(fields, &ss[i].Body)
		}
	}
	if n.Tour != nil {
		steps(n.Tour.Steps)
	}
	for i := range n.Guides {
		steps(n.Guides[i].Steps)
	}
	for i := range n.Flows {
		fl := &n.Flows[i]
		for j := range fl.Inputs {
			fields = append(fields, &fl.Inputs[j])
		}
		for j := range fl.Outputs {
			fields = append(fields, &fl.Outputs[j])
		}
		for j := range fl.Stages {
			fields = append(fields, &fl.Stages[j].Body)
		}
	}
	for i := range n.Families {
		fields = append(fields, &n.Families[i].Intro)
	}
	for i := range n.Findings {
		fields = append(fields, &n.Findings[i].B)
	}
	return fields
}

// validate sanitizes the narrative and checks that every package and file it
// mentions exists, and that every package is placed on the map exactly once.
func validate(r *repo, f *Facts, n *Narrative) (refs, warnings []string, err error) {
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
	scan := func(s string) {
		if !balanced(s) {
			errs = append(errs, fmt.Sprintf("unbalanced or misnested tags in %q", truncate(s, 80)))
		}
		for _, m := range srcAttrRE.FindAllStringSubmatch(s, -1) {
			refs = append(refs, m[1])
		}
		for _, m := range pkgAttrRE.FindAllStringSubmatch(s, -1) {
			checkPkg("data-pkg", m[1])
		}
	}
	for id, d := range n.Descriptions {
		n.Descriptions[id] = sanitize(d)
		scan(n.Descriptions[id])
	}
	contractOf := map[string]Contract{}
	for _, c := range f.Contracts {
		contractOf[c.Parent] = c
	}
	for i := range n.Families {
		fam := &n.Families[i]
		c, ok := contractOf[fam.Parent]
		if !ok {
			errs = append(errs, fmt.Sprintf("family %q: no computed contract for that parent directory", fam.Parent))
		}
		members := map[string]bool{}
		for _, m := range c.Members {
			members[m] = true
		}
		for id, row := range fam.Rows {
			if ok && !members[id] {
				errs = append(errs, fmt.Sprintf("family %q: %q is not a member", fam.Parent, id))
			}
			if len(row.Cells) != len(fam.Columns) {
				errs = append(errs, fmt.Sprintf("family %q row %q: %d cells for %d columns", fam.Parent, id, len(row.Cells), len(fam.Columns)))
			}
			for j := range row.Cells {
				row.Cells[j] = sanitize(row.Cells[j])
				scan(row.Cells[j])
			}
			row.Detail = sanitize(row.Detail)
			scan(row.Detail)
			fam.Rows[id] = row
		}
	}
	for _, s := range htmlFields(n) {
		*s = sanitize(*s)
		scan(*s)
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
		li, ok := layerOf[p.ID]
		if !ok {
			errs = append(errs, fmt.Sprintf("package %q is not assigned to a layer", p.ID))
		}
		if _, ok := n.Descriptions[p.ID]; !ok {
			undescribed = append(undescribed, p.ID)
		}
		if !ok || p.TestOnly || n.Layers[li].ID == "test" {
			continue
		}
		for _, imp := range p.Imports {
			if lj, ok := layerOf[imp]; ok && lj < li {
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
	checkSteps := func(where string, ss []Step) {
		if len(ss) == 0 {
			errs = append(errs, where+" has no steps")
		}
		for _, s := range ss {
			checkPkg(where, s.Pkg)
			refs = append(refs, s.Look...)
			if s.Code != "" {
				refs = append(refs, s.Code)
			}
		}
	}
	if n.Tour != nil {
		checkSteps("tour", n.Tour.Steps)
	}
	ids := map[string]bool{}
	for _, g := range n.Guides {
		if !idRE.MatchString(g.ID) || ids["g:"+g.ID] {
			errs = append(errs, fmt.Sprintf("guide id %q must be unique and match %s", g.ID, idRE))
		}
		ids["g:"+g.ID] = true
		checkSteps("guide "+g.ID, g.Steps)
	}
	for _, fl := range n.Flows {
		if !idRE.MatchString(fl.ID) || ids["f:"+fl.ID] {
			errs = append(errs, fmt.Sprintf("flow id %q must be unique and match %s", fl.ID, idRE))
		}
		ids["f:"+fl.ID] = true
		if len(fl.Stages) == 0 {
			errs = append(errs, fmt.Sprintf("flow %q has no stages", fl.ID))
		}
		refs = append(refs, fl.File)
		for _, st := range fl.Stages {
			refs = append(refs, st.Where)
		}
	}
	for _, fd := range n.Findings {
		checkPkg("finding "+fd.T, fd.Pkg)
		refs = append(refs, fd.F)
	}
	refs = dedupe(refs)
	for _, ref := range refs {
		if _, err := r.resolve(ref); err != nil {
			errs = append(errs, err.Error())
		}
	}
	if len(errs) > 0 {
		return nil, warnings, fmt.Errorf("narrative is invalid:\n  %s", strings.Join(errs, "\n  "))
	}
	return refs, warnings, nil
}

// computeChecks derives findings directly from facts and the layer map.
func computeChecks(f *Facts, n *Narrative) []Check {
	layerOf := map[string]int{}
	for i, l := range n.Layers {
		for _, id := range l.Packages {
			layerOf[id] = i
		}
	}
	isTest := func(p Package) bool { return p.TestOnly || n.Layers[layerOf[p.ID]].ID == "test" }
	upward := Check{Title: "Upward imports", Detail: "Packages that import a package placed in a higher layer on the map.", Rows: [][]string{}}
	untested := Check{Title: "Packages without tests", Detail: "Non-main packages with no _test.go files.", Rows: [][]string{}}
	orphans := Check{Title: "Packages nothing imports", Detail: "Non-main, non-test packages that no other package in the module imports.", Rows: [][]string{}}
	for _, p := range f.Packages {
		if !isTest(p) {
			for _, imp := range p.Imports {
				if lj, ok := layerOf[imp]; ok && lj < layerOf[p.ID] {
					upward.Rows = append(upward.Rows, []string{p.ID, imp})
				}
			}
		}
		if p.Name != "main" && p.Tests == 0 && !isTest(p) {
			untested.Rows = append(untested.Rows, []string{p.ID})
		}
		if p.Name != "main" && len(p.ImportedBy) == 0 && !isTest(p) {
			orphans.Rows = append(orphans.Rows, []string{p.ID})
		}
	}
	return []Check{upward, untested, orphans}
}

func collectFiles(r *repo, f *Facts, refs []string) (map[string]File, error) {
	files := map[string]File{}
	pkgOfDir := map[string]string{}
	for _, p := range f.Packages {
		pkgOfDir[p.ID+"/"] = p.ID
		refs = append(refs, p.ID+"/")
	}
	for _, ref := range dedupe(refs) {
		rel, err := r.resolve(ref)
		if err != nil {
			return nil, err
		}
		if strings.HasSuffix(ref, "/") {
			d, err := r.listDir(rel)
			if err != nil {
				return nil, err
			}
			d.Pkg = pkgOfDir[ref]
			files[ref] = d
			continue
		}
		b, cut, err := r.read(rel)
		if err != nil {
			return nil, err
		}
		file := File{Lines: countLines(b), Trunc: cut, Pkg: pkgOfDir[path.Dir(rel)+"/"]}
		lines := strings.Split(strings.TrimSuffix(string(b), "\n"), "\n")
		if len(lines) > maxInlineLines {
			lines, file.Trunc = lines[:maxInlineLines], true
		}
		file.Content = strings.Join(lines, "\n")
		files[ref] = file
	}
	return files, nil
}

// listDir lists a directory at HEAD, omitting credential-like files.
func (r *repo) listDir(dir string) (File, error) {
	d := File{Dir: true}
	for _, e := range r.dirs[dir] {
		if strings.HasSuffix(e, "/") {
			d.Entries = append(d.Entries, e)
			continue
		}
		if isSecretName(e) {
			continue
		}
		b, _, err := r.read(path.Join(dir, e))
		if err != nil {
			return d, err
		}
		n := countLines(b)
		d.Lines += n
		d.Entries = append(d.Entries, fmt.Sprintf("%s (%d lines)", e, n))
	}
	return d, nil
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

// skipPath mirrors the go command, which ignores directories starting with
// "." or "_" and directories named testdata, plus common vendored trees.
func skipPath(p string) bool {
	for _, el := range strings.Split(path.Dir(p), "/") {
		if el != "." && (strings.HasPrefix(el, ".") || strings.HasPrefix(el, "_") ||
			el == "testdata" || el == "vendor" || el == "node_modules") {
			return true
		}
	}
	return false
}

func inModule(dir string, nested []string) bool {
	for _, m := range nested {
		if dir == m || strings.HasPrefix(dir, m+"/") {
			return true
		}
	}
	return false
}

func githubRepo(remote string) string {
	m := githubRE.FindStringSubmatch(remote)
	if m == nil {
		return ""
	}
	return m[1]
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

func sortedStrings(s []string) []string {
	sort.Strings(s)
	return s
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// nonNil makes empty slices encode as [] rather than null.
func nonNil[T any](s []T) []T {
	if s == nil {
		return []T{}
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

func readJSON(p string, v any) error {
	b, err := os.ReadFile(p)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, v)
}
