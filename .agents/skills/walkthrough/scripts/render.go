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

// render turns an agent-authored walkthrough.json into a self-contained
// walkthrough.html. It computes every fact about the repository itself (the
// package list, imports, line counts, doc synopses, module path, commit) so
// the JSON only has to carry what a human or agent should write: layer
// assignments and narrative.
//
// Usage:
//
//	go run .agents/skills/walkthrough/scripts/render.go -init -out walkthrough.json
//	go run .agents/skills/walkthrough/scripts/render.go -in walkthrough.json -out walkthrough.html
//	go run .agents/skills/walkthrough/scripts/render.go -in walkthrough.json -check
package main

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
)

//go:embed template.html
var defaultTemplate []byte

const otherLayer = "other"

// Site is the JSON contract between the skill and the template. Fields marked
// "computed" are filled by this program and ignored on input.
type Site struct {
	Title         string        `json:"title"`
	Subtitle      string        `json:"subtitle,omitempty"`
	SHA           string        `json:"sha,omitempty"`      // computed unless given
	ShortSHA      string        `json:"shortSha,omitempty"` // computed
	Repo          string        `json:"repo,omitempty"`     // computed from module path unless given
	GitRef        string        `json:"gitRef,omitempty"`
	Module        string        `json:"module,omitempty"`   // computed
	TotalLOC      int           `json:"totalLoc,omitempty"` // computed
	Layers        []Layer       `json:"layers"`
	Pkgs          []Pkg         `json:"pkgs"`
	Steps         []Step        `json:"steps,omitempty"`
	Flows         []Flow        `json:"flows,omitempty"`
	Guides        []Guide       `json:"guides,omitempty"`
	Findings      []Finding     `json:"findings,omitempty"`
	Langs         []Lang        `json:"langs,omitempty"`
	ContractCols  []string      `json:"contractCols,omitempty"`
	ContractHooks []ContractRow `json:"contractHooks,omitempty"`
	DefaultPkg    string        `json:"defaultPkg,omitempty"`
}

type Layer struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Hint  string `json:"hint,omitempty"`
}

type Pkg struct {
	ID      string   `json:"id"`
	Layer   string   `json:"layer"`
	Side    string   `json:"side,omitempty"`
	Desc    string   `json:"desc,omitempty"`
	LOC     int      `json:"loc"`     // computed
	Imports []string `json:"imports"` // computed
}

type Step struct {
	Title string   `json:"t"`
	Body  string   `json:"body"`
	Look  []string `json:"look,omitempty"`
	Pkg   string   `json:"pkg,omitempty"`
}

type Flow struct {
	ID     string      `json:"id"`
	Title  string      `json:"title"`
	Cmd    string      `json:"cmd"`
	File   string      `json:"file"`
	Pkg    string      `json:"pkg,omitempty"`
	Inputs []string    `json:"inputs,omitempty"`
	Stages []FlowStage `json:"stages"`
}

type FlowStage struct {
	Title string `json:"title"`
	Where string `json:"where"`
	Body  string `json:"body"`
	Out   string `json:"out,omitempty"`
}

type Guide struct {
	ID       string      `json:"id"`
	Title    string      `json:"title"`
	Subtitle string      `json:"subtitle,omitempty"`
	Audience string      `json:"audience,omitempty"`
	Steps    []GuideStep `json:"steps"`
}

type GuideStep struct {
	Title string   `json:"t"`
	Runs  string   `json:"runs,omitempty"`
	Code  string   `json:"code,omitempty"`
	Body  string   `json:"body"`
	Look  []string `json:"look,omitempty"`
	Pkg   string   `json:"pkg,omitempty"`
}

type Finding struct {
	Title string `json:"t"`
	Body  string `json:"b"`
	File  string `json:"f"`
	Pkg   string `json:"pkg,omitempty"`
}

type Lang struct {
	Name   string `json:"n"`
	Pkg    string `json:"pkg"`
	Gen    string `json:"gen,omitempty"`
	Inst   string `json:"inst,omitempty"`
	Fmt    string `json:"fmt,omitempty"`
	Pub    string `json:"pub,omitempty"`
	Detail string `json:"detail,omitempty"`
	LOC    int    `json:"loc"` // computed from Pkg
}

type ContractRow struct {
	Hook   string   `json:"hook"`
	Caller string   `json:"caller,omitempty"`
	Cells  []string `json:"cells"`
}

func main() {
	var in, out, root, tmplPath string
	var initSkeleton, check bool
	flag.StringVar(&in, "in", "", "walkthrough JSON authored by the skill")
	flag.StringVar(&out, "out", "", "output path: .html when rendering, .json with -init (default: stdout)")
	flag.StringVar(&root, "root", "", "module root (default: directory of go env GOMOD)")
	flag.StringVar(&tmplPath, "template", "", "HTML template with a {{.JSON}} slot (default: the embedded template)")
	flag.BoolVar(&initSkeleton, "init", false, "write a skeleton JSON listing every package with its doc synopsis")
	flag.BoolVar(&check, "check", false, "validate -in without writing HTML")
	flag.Parse()
	if err := run(context.Background(), in, out, root, tmplPath, initSkeleton, check); err != nil {
		fmt.Fprintln(os.Stderr, "render:", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, in, out, root, tmplPath string, initSkeleton, check bool) error {
	facts, err := loadFacts(ctx, root)
	if err != nil {
		return err
	}
	if initSkeleton {
		if out != "" {
			if _, err := os.Stat(out); err == nil {
				return fmt.Errorf("%s already exists; -init will not overwrite authored content", out)
			}
		}
		return writeOutput(out, skeleton(facts))
	}
	if in == "" {
		return errors.New("-in is required (or use -init)")
	}
	site, err := readSite(in)
	if err != nil {
		return err
	}
	merge(site, facts)
	if errs := validate(site, facts); len(errs) > 0 {
		return fmt.Errorf("%d problem(s) in %s:\n  - %s", len(errs), in, strings.Join(errs, "\n  - "))
	}
	if check {
		fmt.Printf("ok: %d packages, %d layers, %d tour steps, %d flows, %d guides at %s\n",
			len(site.Pkgs), len(site.Layers), len(site.Steps), len(site.Flows), len(site.Guides), site.ShortSHA)
		return nil
	}
	tmpl := defaultTemplate
	if tmplPath != "" {
		if tmpl, err = os.ReadFile(tmplPath); err != nil {
			return err
		}
	}
	if !bytes.Contains(tmpl, []byte("{{.JSON}}")) {
		return errors.New("template has no {{.JSON}} slot")
	}
	// json.Marshal escapes <, > and & as \u003c etc., so the payload can never
	// close the <script type="application/json"> block it is embedded in.
	payload, err := json.Marshal(site)
	if err != nil {
		return err
	}
	html := bytes.Replace(tmpl, []byte("{{.JSON}}"), payload, 1)
	if out == "" {
		out = "walkthrough.html"
	}
	if err := os.WriteFile(out, html, 0o644); err != nil {
		return err
	}
	fmt.Printf("wrote %s (%d bytes) for %s@%s\n", out, len(html), site.Repo, site.ShortSHA)
	return nil
}

// facts is everything derived from the checkout rather than authored.
type facts struct {
	root, module, sha string
	pkgs              map[string]*Pkg // keyed by module-relative ID, in go list order
	order             []string
	docs              map[string]string
}

type goListPkg struct {
	ImportPath string
	Dir        string
	Doc        string
	GoFiles    []string
	Imports    []string
	Module     *struct{ Path, Dir string }
}

func loadFacts(ctx context.Context, root string) (*facts, error) {
	if root == "" {
		gomod, err := goCmd(ctx, ".", "env", "GOMOD")
		if err != nil {
			return nil, err
		}
		gomod = strings.TrimSpace(gomod)
		if gomod == "" || gomod == os.DevNull {
			return nil, errors.New("not inside a Go module (run from the repository or pass -root)")
		}
		root = filepath.Dir(gomod)
	}
	raw, err := goCmd(ctx, root, "list", "-json=ImportPath,Dir,Doc,GoFiles,Imports,Module", "./...")
	if err != nil {
		return nil, err
	}
	f := &facts{root: root, pkgs: map[string]*Pkg{}, docs: map[string]string{}}
	dec := json.NewDecoder(strings.NewReader(raw))
	var listed []goListPkg
	for {
		var p goListPkg
		if err := dec.Decode(&p); err == io.EOF {
			break
		} else if err != nil {
			return nil, fmt.Errorf("parsing go list output: %w", err)
		}
		listed = append(listed, p)
	}
	if len(listed) == 0 || listed[0].Module == nil {
		return nil, errors.New("go list found no module packages under " + root)
	}
	f.module = listed[0].Module.Path
	// rel maps an import path inside the module to a module-relative ID; the
	// module root itself is ".".
	rel := func(importPath string) (string, bool) {
		if importPath == f.module {
			return ".", true
		}
		return strings.CutPrefix(importPath, f.module+"/")
	}
	for _, p := range listed {
		id, ok := rel(p.ImportPath)
		if !ok || len(p.GoFiles) == 0 { // skip test-only directories
			continue
		}
		loc := 0
		for _, gf := range p.GoFiles {
			b, err := os.ReadFile(filepath.Join(p.Dir, gf))
			if err != nil {
				return nil, err
			}
			loc += bytes.Count(b, []byte{'\n'})
		}
		var imports []string
		for _, imp := range p.Imports {
			if target, ok := rel(imp); ok {
				imports = append(imports, target)
			}
		}
		slices.Sort(imports)
		f.pkgs[id] = &Pkg{ID: id, LOC: loc, Imports: imports}
		f.docs[id] = p.Doc
		f.order = append(f.order, id)
	}
	if sha, err := exec.CommandContext(ctx, "git", "-C", root, "rev-parse", "HEAD").Output(); err == nil {
		f.sha = strings.TrimSpace(string(sha))
	} else {
		f.sha = "HEAD"
	}
	return f, nil
}

func goCmd(ctx context.Context, dir string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "go", args...)
	cmd.Dir = dir
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("go %s: %w\n%s", strings.Join(args, " "), err, stderr.String())
	}
	return string(out), nil
}

// skeletonSite mirrors Site without computed fields.
type skeletonSite struct {
	Title  string        `json:"title"`
	Layers []Layer       `json:"layers"`
	Pkgs   []skeletonPkg `json:"pkgs"`
	Steps  []Step        `json:"steps"`
}

type skeletonPkg struct {
	ID    string `json:"id"`
	Layer string `json:"layer"`
	Desc  string `json:"desc"`
}

func skeleton(f *facts) *skeletonSite {
	s := &skeletonSite{
		Title:  f.module + " architecture walkthrough",
		Layers: []Layer{{ID: "entry", Title: "Entry points"}, {ID: "core", Title: "Core"}, {ID: "infra", Title: "Infrastructure"}, {ID: "test", Title: "Test support"}},
	}
	for _, id := range f.order {
		s.Pkgs = append(s.Pkgs, skeletonPkg{ID: id, Layer: otherLayer, Desc: f.docs[id]})
	}
	s.Steps = []Step{{Title: "TODO: first step", Body: "<p>TODO: replace with narrative; cite files with <a data-src=\"go.mod\">go.mod</a>.</p>", Look: []string{"go.mod"}}}
	return s
}

func readSite(path string) (*Site, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.DisallowUnknownFields()
	var s Site
	if err := dec.Decode(&s); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}
	return &s, nil
}

// merge overwrites every computed field from facts and appends any package the
// JSON omitted to the "other" layer so the map never silently drops code.
func merge(s *Site, f *facts) {
	s.Module = f.module
	if s.SHA == "" {
		s.SHA = f.sha
	}
	s.ShortSHA = s.SHA
	if len(s.ShortSHA) > 8 {
		s.ShortSHA = s.ShortSHA[:8]
	}
	if s.Repo == "" {
		s.Repo = repoFromModule(f.module)
	}
	if s.GitRef == "" {
		s.GitRef = "main"
	}
	seen := map[string]bool{}
	for i := range s.Pkgs {
		p := &s.Pkgs[i]
		seen[p.ID] = true
		if fp, ok := f.pkgs[p.ID]; ok {
			p.LOC, p.Imports = fp.LOC, fp.Imports
		}
		if p.Imports == nil {
			p.Imports = []string{}
		}
		if p.Desc == "" {
			p.Desc = f.docs[p.ID]
		}
	}
	missing := false
	for _, id := range f.order {
		if !seen[id] {
			fp := f.pkgs[id]
			s.Pkgs = append(s.Pkgs, Pkg{ID: id, Layer: otherLayer, Desc: f.docs[id], LOC: fp.LOC, Imports: fp.Imports})
			fmt.Fprintf(os.Stderr, "warning: %s is not in the JSON; shown as Unclassified\n", id)
			missing = true
		}
	}
	if missing && !slices.ContainsFunc(s.Layers, func(l Layer) bool { return l.ID == otherLayer }) {
		s.Layers = append(s.Layers, Layer{ID: otherLayer, Title: "Unclassified", Hint: "packages not yet assigned a layer in walkthrough.json"})
	}
	s.TotalLOC = 0
	for _, p := range s.Pkgs {
		s.TotalLOC += p.LOC
	}
	locByPkg := map[string]int{}
	for _, p := range s.Pkgs {
		locByPkg[p.ID] = p.LOC
	}
	for i := range s.Langs {
		s.Langs[i].LOC = locByPkg[s.Langs[i].Pkg]
	}
	if s.DefaultPkg == "" && len(s.Pkgs) > 0 {
		s.DefaultPkg = s.Pkgs[0].ID
	}
}

var (
	dataSrcRe = regexp.MustCompile(`data-src="([^"]+)"`)
	repoRe    = regexp.MustCompile(`^[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+$`)
	gitRefRe  = regexp.MustCompile(`^[A-Za-z0-9_./-]+$`)
)

// validate returns every problem at once so an agent can fix them in one pass.
func validate(s *Site, f *facts) []string {
	var errs []string
	add := func(format string, a ...any) { errs = append(errs, fmt.Sprintf(format, a...)) }

	if strings.TrimSpace(s.Title) == "" {
		add("title is required")
	}
	// repo and gitRef are interpolated into GitHub URLs by the page.
	if s.Repo != "" && !repoRe.MatchString(s.Repo) {
		add("repo %q must look like org/name", s.Repo)
	}
	if !gitRefRe.MatchString(s.GitRef) || strings.Contains(s.GitRef, "..") {
		add("gitRef %q must be a plain branch or tag name", s.GitRef)
	}
	unique := func(kind string) func(id string) {
		seen := map[string]bool{}
		return func(id string) {
			if id == "" {
				return
			}
			if seen[id] {
				add("duplicate %s id %q", kind, id)
			}
			seen[id] = true
		}
	}
	flowID, guideID := unique("flow"), unique("guide")
	layers := map[string]bool{}
	for _, l := range s.Layers {
		if l.ID == "" || l.Title == "" {
			add("layer %+v needs id and title", l)
		}
		if layers[l.ID] {
			add("duplicate layer id %q", l.ID)
		}
		layers[l.ID] = true
	}
	if len(layers) == 0 {
		add("at least one layer is required")
	}
	pkgs := map[string]bool{}
	for _, p := range s.Pkgs {
		if pkgs[p.ID] {
			add("duplicate package %q", p.ID)
		}
		pkgs[p.ID] = true
		if _, ok := f.pkgs[p.ID]; !ok {
			add("package %q is not in go list ./... (typo, or stale after a rename?)", p.ID)
		}
		if !layers[p.Layer] {
			add("package %q uses unknown layer %q", p.ID, p.Layer)
		}
	}
	checkPkg := func(where, id string) {
		if id != "" && !pkgs[id] {
			add("%s references unknown package %q", where, id)
		}
	}
	if s.DefaultPkg != "" {
		checkPkg("defaultPkg", s.DefaultPkg)
	}

	paths := map[string]string{} // path -> first place it was cited
	cite := func(where, p string) {
		if p == "" {
			return
		}
		if _, dup := paths[p]; !dup {
			paths[p] = where
		}
	}
	citeHTML := func(where, html string) {
		for _, m := range dataSrcRe.FindAllStringSubmatch(html, -1) {
			cite(where, m[1])
		}
	}
	for _, p := range s.Pkgs {
		citeHTML("pkgs."+p.ID+".desc", p.Desc)
	}
	for i, st := range s.Steps {
		w := fmt.Sprintf("steps[%d]", i)
		if st.Title == "" || st.Body == "" {
			add("%s needs t and body", w)
		}
		citeHTML(w, st.Body)
		for _, l := range st.Look {
			cite(w+".look", l)
		}
		checkPkg(w+".pkg", st.Pkg)
	}
	for i, fl := range s.Flows {
		w := fmt.Sprintf("flows[%d]", i)
		if fl.ID == "" || fl.Title == "" || fl.Cmd == "" || len(fl.Stages) == 0 {
			add("%s needs id, title, cmd and at least one stage", w)
		}
		flowID(fl.ID)
		cite(w+".file", fl.File)
		checkPkg(w+".pkg", fl.Pkg)
		for j, stg := range fl.Stages {
			if stg.Title == "" || stg.Body == "" {
				add("%s.stages[%d] needs title and body", w, j)
			}
			citeHTML(w, stg.Where)
			citeHTML(w, stg.Body)
		}
	}
	for i, g := range s.Guides {
		w := fmt.Sprintf("guides[%d]", i)
		if g.ID == "" || g.Title == "" || len(g.Steps) == 0 {
			add("%s needs id, title and at least one step", w)
		}
		guideID(g.ID)
		for j, st := range g.Steps {
			sw := fmt.Sprintf("%s.steps[%d]", w, j)
			if st.Title == "" || st.Body == "" {
				add("%s needs t and body", sw)
			}
			citeHTML(sw, st.Body)
			for _, l := range st.Look {
				cite(sw+".look", l)
			}
			checkPkg(sw+".pkg", st.Pkg)
		}
	}
	for i, fn := range s.Findings {
		w := fmt.Sprintf("findings[%d]", i)
		if fn.Title == "" || fn.Body == "" {
			add("%s needs t and b", w)
		}
		cite(w+".f", fn.File)
		citeHTML(w, fn.Body)
		checkPkg(w+".pkg", fn.Pkg)
	}
	for i, l := range s.Langs {
		w := fmt.Sprintf("langs[%d]", i)
		if l.Name == "" {
			add("%s needs n", w)
		}
		checkPkg(w+".pkg", l.Pkg)
		citeHTML(w, l.Detail)
	}
	for i, r := range s.ContractHooks {
		if len(r.Cells) != len(s.ContractCols) {
			add("contractHooks[%d] has %d cells, want %d (len(contractCols))", i, len(r.Cells), len(s.ContractCols))
		}
	}

	rootFS, err := os.OpenRoot(f.root)
	if err != nil {
		add("opening root: %v", err)
		return errs
	}
	defer rootFS.Close()
	cited := make([]string, 0, len(paths))
	for p := range paths {
		cited = append(cited, p)
	}
	slices.Sort(cited)
	for _, p := range cited {
		clean := strings.TrimSuffix(p, "/")
		if clean == "" || filepath.IsAbs(clean) || strings.HasPrefix(clean, "../") || clean == ".." || strings.Contains(clean, "/../") {
			add("%s cites %q: paths must be relative to the module root with no '..'", paths[p], p)
			continue
		}
		info, err := rootFS.Stat(filepath.FromSlash(clean))
		switch {
		case err != nil:
			add("%s cites %q which does not exist", paths[p], p)
		case strings.HasSuffix(p, "/") && !info.IsDir():
			add("%s cites %q with a trailing slash but it is a file", paths[p], p)
		case !strings.HasSuffix(p, "/") && info.IsDir():
			add("%s cites directory %q without a trailing slash; write %q", paths[p], p, p+"/")
		}
	}
	return errs
}

func repoFromModule(module string) string {
	rest, ok := strings.CutPrefix(module, "github.com/")
	if !ok {
		return ""
	}
	org, rest, ok := strings.Cut(rest, "/")
	if !ok {
		return ""
	}
	repo, _, _ := strings.Cut(rest, "/")
	return org + "/" + repo
}

func writeOutput(out string, v any) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')
	if out == "" {
		_, err = os.Stdout.Write(b)
		return err
	}
	return os.WriteFile(out, b, 0o644)
}
