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
	"bytes"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

const modulePrefix = "github.com/googleapis/librarian/"

var defaultLayers = []Layer{
	{ID: "entry", Title: "Entry points", Hint: "process start and developer tools"},
	{ID: "orch", Title: "Command orchestration", Hint: "urfave/cli commands, config loading, dispatch"},
	{ID: "lang", Title: "Language integrations", Hint: "one package per language; plain exported functions"},
	{ID: "engine", Title: "Generation engine (sidekick) and tool installers", Hint: "in-process codecs on the left, external toolchains on the right"},
	{ID: "domain", Title: "Domain services", Hint: "know about librarian.yaml and API metadata, not about commands"},
	{ID: "infra", Title: "Infrastructure", Hint: "no knowledge of commands, languages or configuration"},
	{ID: "test", Title: "Test support", Hint: "imported only by tests"},
}

type repoStats struct {
	goVersion        string
	librarianVersion string
	totalLOC         int
	pkgs             []Pkg
	tmplCounts       map[string]int
	totalTemplates   int
	langSymbols      map[string]map[string]bool
	langs            []LangInfo
	contractCols     []string
	contractHooks    []ContractRow
}

func analyzeRepo(root string) (*repoStats, error) {
	stats := &repoStats{
		goVersion:        readGoVersion(root),
		librarianVersion: readLibrarianVersion(root),
		tmplCounts:       make(map[string]int),
		langSymbols:      make(map[string]map[string]bool),
	}

	rawPkgs, err := scanGoPackages(root, stats)
	if err != nil {
		return nil, err
	}

	knownIDs := make(map[string]bool, len(rawPkgs))
	for _, p := range rawPkgs {
		knownIDs[p.ID] = true
		stats.totalLOC += p.LOC
	}
	for i := range rawPkgs {
		filtered := []string{}
		for _, imp := range rawPkgs[i].Imports {
			if knownIDs[imp] && imp != rawPkgs[i].ID && !slices.Contains(filtered, imp) {
				filtered = append(filtered, imp)
			}
		}
		slices.Sort(filtered)
		rawPkgs[i].Imports = filtered
	}
	stats.pkgs = rawPkgs
	stats.langs = buildLangInfos(stats)
	stats.contractCols = []string{"dart", "go", "java", "nodejs", "php", "python", "ruby", "rust", "swift"}
	stats.contractHooks = buildContractRows(stats.langSymbols)
	return stats, nil
}

type rawPkgAcc struct {
	id      string
	path    string
	loc     int
	doc     string
	imports []string
}

func scanGoPackages(root string, stats *repoStats) ([]Pkg, error) {
	fset := token.NewFileSet()
	accByID := make(map[string]*rawPkgAcc)

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if d.IsDir() {
			if rel == "." {
				return nil
			}
			if strings.HasPrefix(d.Name(), ".") || d.Name() == "testdata" || d.Name() == "_site" || d.Name() == "vendor" {
				return filepath.SkipDir
			}
			return nil
		}
		rel = filepath.ToSlash(rel)
		if strings.HasSuffix(rel, ".gotmpl") {
			stats.totalTemplates++
			if rest, ok := strings.CutPrefix(rel, "internal/sidekick/"); ok {
				if lang, _, ok := strings.Cut(rest, "/"); ok {
					stats.tmplCounts[lang]++
				}
			}
			return nil
		}
		if !strings.HasSuffix(rel, ".go") {
			return nil
		}
		dir := filepath.Dir(rel)
		if dir == "." {
			return nil
		}
		isTest := strings.HasSuffix(rel, "_test.go")
		pkgID, pkgPath := normalizePkgID(dir)
		acc := accByID[pkgID]
		if acc == nil {
			acc = &rawPkgAcc{id: pkgID, path: pkgPath}
			accByID[pkgID] = acc
		}
		if isTest {
			return nil
		}

		src, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		acc.loc += bytes.Count(src, []byte{'\n'})

		f, err := parser.ParseFile(fset, path, src, parser.ParseComments)
		if err != nil {
			return fmt.Errorf("parsing %s: %w", rel, err)
		}
		if acc.doc == "" && f.Doc != nil {
			if first, _, _ := strings.Cut(strings.TrimSpace(f.Doc.Text()), "\n\n"); first != "" {
				acc.doc = strings.ReplaceAll(first, "\n", " ")
			}
		}
		for _, imp := range f.Imports {
			impPath := strings.Trim(imp.Path.Value, `"`)
			if target, ok := strings.CutPrefix(impPath, modulePrefix); ok {
				targetID, _ := normalizePkgID(target)
				acc.imports = append(acc.imports, targetID)
			}
		}
		if lang, ok := strings.CutPrefix(dir, "internal/librarian/"); ok && !strings.Contains(lang, "/") {
			recordExportedSymbols(stats, lang, f)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	var pkgs []Pkg
	for _, acc := range accByID {
		layer, side, err := classifyLayer(acc.id)
		if err != nil {
			return nil, err
		}
		desc := curatedPkgDescriptions[acc.id]
		if desc == "" {
			desc = acc.doc
		}
		pkgs = append(pkgs, Pkg{
			ID:      acc.id,
			Layer:   layer,
			Side:    side,
			LOC:     acc.loc,
			Path:    acc.path,
			Desc:    desc,
			Imports: acc.imports,
		})
	}
	slices.SortFunc(pkgs, comparePkgs)
	return pkgs, nil
}

func normalizePkgID(dir string) (string, string) {
	if strings.HasPrefix(dir, "tool/") || dir == "tool" {
		return "tool/cmd", "tool/cmd/"
	}
	if dir == "cmd" {
		return "cmd/librarian", "cmd/librarian/"
	}
	return dir, ""
}

func recordExportedSymbols(stats *repoStats, lang string, f *ast.File) {
	syms := stats.langSymbols[lang]
	if syms == nil {
		syms = make(map[string]bool)
		stats.langSymbols[lang] = syms
	}
	for _, decl := range f.Decls {
		switch d := decl.(type) {
		case *ast.FuncDecl:
			if d.Recv == nil && d.Name.IsExported() {
				syms[d.Name.Name] = true
			}
		case *ast.GenDecl:
			for _, spec := range d.Specs {
				if vs, ok := spec.(*ast.ValueSpec); ok {
					for _, name := range vs.Names {
						if name.IsExported() {
							syms[name.Name] = true
						}
					}
				}
			}
		}
	}
}

func classifyLayer(id string) (string, string, error) {
	switch {
	case strings.HasPrefix(id, "cmd/") || id == "tool/cmd":
		return "entry", "", nil
	case id == "internal/librarian":
		return "orch", "", nil
	case strings.HasPrefix(id, "internal/librarian/"):
		return "lang", "", nil
	case strings.HasPrefix(id, "internal/sidekick/api/apitest") || id == "internal/testhelper" || id == "internal/sample":
		return "test", "", nil
	case strings.HasPrefix(id, "internal/sidekick/"):
		return "engine", "", nil
	case strings.HasPrefix(id, "internal/tool/"):
		return "engine", "tools", nil
	case slices.Contains([]string{
		"internal/config", "internal/serviceconfig", "internal/sources",
		"internal/repometadata", "internal/postprocessing", "internal/snippetmetadata",
		"internal/semver", "internal/proto", "internal/license", "internal/docuploader",
	}, id):
		return "domain", "", nil
	case slices.Contains([]string{
		"internal/command", "internal/cache", "internal/fetch",
		"internal/filesystem", "internal/git", "internal/yaml",
	}, id):
		return "infra", "", nil
	default:
		return "", "", fmt.Errorf("package %q does not match any architectural layer in tool/cmd/walkthrough/analyze.go", id)
	}
}

var pkgDisplayOrder = []string{
	"cmd/librarian", "tool/cmd",
	"internal/librarian",
	"internal/librarian/dart", "internal/librarian/golang", "internal/librarian/java",
	"internal/librarian/nodejs", "internal/librarian/php", "internal/librarian/python",
	"internal/librarian/ruby", "internal/librarian/rust", "internal/librarian/swift",
	"internal/sidekick/rust_prost", "internal/sidekick/rust", "internal/sidekick/swift",
	"internal/sidekick/dart", "internal/sidekick/codec_sample", "internal/sidekick/language",
	"internal/sidekick/parser", "internal/sidekick/parser/discovery",
	"internal/sidekick/parser/httprule", "internal/sidekick/parser/svcconfig",
	"internal/sidekick/protobuf", "internal/sidekick/api",
	"internal/tool/protoc", "internal/tool/maven", "internal/tool/pip",
	"internal/tool/pnpm", "internal/tool/gem", "internal/tool/composer",
	"internal/config", "internal/serviceconfig", "internal/sources", "internal/repometadata",
	"internal/postprocessing", "internal/snippetmetadata", "internal/semver",
	"internal/proto", "internal/license", "internal/docuploader",
	"internal/command", "internal/cache", "internal/fetch",
	"internal/filesystem", "internal/git", "internal/yaml",
	"internal/testhelper", "internal/sample", "internal/sidekick/api/apitest",
}

func comparePkgs(a, b Pkg) int {
	ia := slices.Index(pkgDisplayOrder, a.ID)
	ib := slices.Index(pkgDisplayOrder, b.ID)
	if ia >= 0 && ib >= 0 {
		return ia - ib
	}
	if ia >= 0 {
		return -1
	}
	if ib >= 0 {
		return 1
	}
	return strings.Compare(a.ID, b.ID)
}

func readGoVersion(root string) string {
	data, err := os.ReadFile(filepath.Join(root, "go.mod"))
	if err != nil {
		return "Go"
	}
	for line := range strings.Lines(string(data)) {
		if v, ok := strings.CutPrefix(strings.TrimSpace(line), "go "); ok {
			return "Go " + strings.TrimSpace(v)
		}
	}
	return "Go"
}

func readLibrarianVersion(root string) string {
	data, err := os.ReadFile(filepath.Join(root, ".release-please-manifest.json"))
	if err != nil {
		return "HEAD"
	}
	var m map[string]string
	if err := json.Unmarshal(data, &m); err != nil || m["."] == "" {
		return "HEAD"
	}
	return "v" + m["."]
}

func buildLangInfos(stats *repoStats) []LangInfo {
	locByPkg := make(map[string]int, len(stats.pkgs))
	for _, p := range stats.pkgs {
		locByPkg[p.ID] = p.LOC
	}
	return []LangInfo{
		{
			Name: "Dart", Pkg: "internal/librarian/dart", LOC: locByPkg["internal/librarian/dart"],
			Gen:  fmt.Sprintf("sidekick (<code>internal/sidekick/dart</code>, %d templates)", stats.tmplCounts["dart"]),
			Inst: "nothing; Dart SDK on <code>PATH</code>", Fmt: "<code>dart format</code>",
			Meta: false, Bump: true, Pub: "pub.dev (<code>dart pub publish</code>)", CI: "dart.yaml",
			Detail: "HTTP/JSON clients rendered directly by sidekick templates. <code>DeriveAPIPath</code> maps the package name to the API path. Bump reads published versions from pub.dev, diffs the public API with <code>dart-apitool</code>, and rewrites <code>pubspec.yaml</code> and <code>CHANGELOG.md</code>.",
		},
		{
			Name: "Go", Pkg: "internal/librarian/golang", LOC: locByPkg["internal/librarian/golang"],
			Gen:  "<code>protoc</code> + <code>protoc-gen-go</code>, <code>protoc-gen-go-grpc</code>, <code>protoc-gen-go_gapic</code>",
			Inst: "<code>go install</code> into <code>go_tools</code>", Fmt: "<code>goimports</code>",
			Meta: true, Bump: true, Pub: "release-please + module proxy (outside Librarian)", CI: "go.yaml",
			Detail: "Three <code>protoc</code> passes (messages, gRPC, GAPIC), then <code>go mod tidy</code>. Writes <code>.repo-metadata.json</code> and refreshes snippet metadata. <code>Fill</code> derives module paths and import paths.",
		},
		{
			Name: "Java", Pkg: "internal/librarian/java", LOC: locByPkg["internal/librarian/java"],
			Gen:  "<code>protoc</code> + <code>protoc-gen-java_gapic</code>, <code>protoc-gen-java_grpc</code>, <code>--java_out</code>",
			Inst: "<code>internal/tool/maven</code> into <code>java_tools</code>", Fmt: "<code>google-java-format</code> (all libraries at once)",
			Meta: true, Bump: false, Pub: "Maven Central (outside Librarian)", CI: "java.yaml",
			Detail: "Largest language integration and sole consumer of declarative <code>internal/postprocessing</code> rules. <code>PostGenerate</code> maintains POMs, BOMs and version files across the repository, which is why Java libraries generate sequentially.",
		},
		{
			Name: "Node.js", Pkg: "internal/librarian/nodejs", LOC: locByPkg["internal/librarian/nodejs"],
			Gen:  "<code>gapic-generator-typescript</code> (drives <code>protoc</code> itself)",
			Inst: "<code>internal/tool/pnpm</code> into <code>nodejs_tools</code>", Fmt: "inside the Node tools",
			Meta: true, Bump: false, Pub: "npm via release-please (outside Librarian)", CI: "nodejs.yaml",
			Detail: "Generates into a staging directory and runs <code>combine-library</code> to merge multiple API versions into one package. Supports mixed libraries (<code>IsMixedLibrary</code>) and finding an existing library for a new API version.",
		},
		{
			Name: "PHP", Pkg: "internal/librarian/php", LOC: locByPkg["internal/librarian/php"],
			Gen:  "<code>protoc</code> + <code>gapic-generator-php</code> plugin, <code>--php_out</code>",
			Inst: "<code>internal/tool/composer</code>, <code>pip</code>, <code>pnpm</code> into <code>php_tools</code>", Fmt: "<code>prettier</code> with the PHP plugin",
			Meta: false, Bump: false, Pub: "Packagist (outside Librarian)", CI: "php.yaml",
			Detail: "Generates into <code>owl-bot-staging</code>, then runs <code>php-post-processor</code> and the library's <code>owlbot.py</code>, using Composer, pip and pnpm together.",
		},
		{
			Name: "Python", Pkg: "internal/librarian/python", LOC: locByPkg["internal/librarian/python"],
			Gen:  "<code>protoc</code> + <code>protoc-gen-python_gapic</code>",
			Inst: "<code>internal/tool/pip</code> into <code>python_tools</code>", Fmt: "<code>nox -s format</code>",
			Meta: true, Bump: true, Pub: "PyPI via release-please (outside Librarian)", CI: "python.yaml",
			Detail: "Generates into <code>owl-bot-staging</code> and runs synthtool's <code>owlbot_main</code>. Refreshes snippet metadata and <code>.repo-metadata.json</code>. Supports per-library tags on <code>bump</code>.",
		},
		{
			Name: "Ruby", Pkg: "internal/librarian/ruby", LOC: locByPkg["internal/librarian/ruby"],
			Gen:  "<code>protoc</code> + <code>gapic-generator-ruby</code>, gRPC plugin, <code>--ruby_out</code>",
			Inst: "<code>internal/tool/gem</code> into <code>ruby_tools</code>", Fmt: "none (no-op)",
			Meta: false, Bump: false, Pub: "RubyGems via release-please (outside Librarian)", CI: "ruby.yaml",
			Detail: "Multi-wrapper gems driven by <code>toys</code> tasks. <code>librarian add --name</code> is Ruby-only. <code>AddManifest</code> and <code>AddPackage</code> keep release-please configs synchronized.",
		},
		{
			Name: "Rust", Pkg: "internal/librarian/rust", LOC: locByPkg["internal/librarian/rust"],
			Gen:  fmt.Sprintf("sidekick (<code>internal/sidekick/rust</code>, %d templates, plus <code>rust_prost</code>)", stats.tmplCounts["rust"]),
			Inst: "<code>cargo install --locked</code> for formatters", Fmt: "<code>taplo fmt</code>, <code>cargo fmt</code>",
			Meta: true, Bump: true, Pub: "crates.io (<code>cargo workspaces publish</code>)", CI: "sidekick.yaml",
			Detail: "After generation <code>UpdateWorkspace</code> updates Cargo workspace manifests. <code>ResolveDependencies</code> wires inter-crate dependencies. Publishes directly to crates.io and emits docs index metadata.",
		},
		{
			Name: "Swift", Pkg: "internal/librarian/swift", LOC: locByPkg["internal/librarian/swift"],
			Gen:  fmt.Sprintf("sidekick (<code>internal/sidekick/swift</code>, %d templates) + <code>protoc --swift_out</code>", stats.tmplCounts["swift"]),
			Inst: "<code>git clone</code> + <code>swift build</code> into <code>swift_tools</code>", Fmt: "<code>swift-format</code>",
			Meta: true, Bump: true, Pub: "per-package GitHub repositories <code>googleapis/swift-NAME</code>", CI: "sidekick.yaml",
			Detail: "The reference sidekick codec design. Supports mixed libraries and publishes by splitting monorepo git history into per-package GitHub repositories.",
		},
	}
}

func buildContractRows(syms map[string]map[string]bool) []ContractRow {
	langDirs := []string{"dart", "golang", "java", "nodejs", "php", "python", "ruby", "rust", "swift"}
	has := func(dir, sym string) bool { return syms[dir][sym] }

	row := func(hook, caller string, fn func(dir string) string) ContractRow {
		cells := make([]string, len(langDirs))
		for i, d := range langDirs {
			cells[i] = fn(d)
		}
		return ContractRow{Hook: hook, Caller: caller, Cells: cells}
	}

	mark := func(b bool) string {
		if b {
			return "✓"
		}
		return "·"
	}

	return []ContractRow{
		row("Generate(ctx, cfg, lib, sources)", "generateLibraries", func(d string) string { return mark(has(d, "Generate")) }),
		row("Format(ctx, lib)", "generateLibraries", func(d string) string {
			if d == "java" && has(d, "FormatAll") {
				return "✓ (all)"
			}
			return mark(has(d, "Format"))
		}),
		row("post-generate step", "generateLibraries", func(d string) string {
			switch {
			case has(d, "PostGenerate"):
				return "PostGenerate"
			case has(d, "UpdateWorkspace"):
				return "UpdateWorkspace + docindex"
			case has(d, "PackageName"):
				return "docindex"
			default:
				return "·"
			}
		}),
		row("Clean(lib)", "cleanLibraries", func(d string) string {
			switch {
			case has(d, "Clean"):
				return "✓"
			case has(d, "Keep"):
				return "Keep + keep"
			default:
				return "keep"
			}
		}),
		row("DefaultOutput(name, default)", "applyDefaults", func(d string) string { return mark(has(d, "DefaultOutput")) }),
		row("DeriveAPIPath(name)", "applyDefaults", func(d string) string { return mark(has(d, "DeriveAPIPath")) }),
		row("IsMixedLibrary(lib)", "applyDefaults", func(d string) string { return mark(has(d, "IsMixedLibrary")) }),
		row("Fill(lib)", "fillLibraryDefaults", func(d string) string { return mark(has(d, "Fill")) }),
		row("DefaultLibraryName(api), Add(...)", "addNewLibrary", func(d string) string { return mark(has(d, "DefaultLibraryName") || has(d, "Add")) }),
		row("FindExistingLibraryForNewAPI", "findExistingLibraryForAPI", func(d string) string { return mark(has(d, "FindExistingLibraryForNewAPI")) }),
		row("dependency resolution", "resolveDependencies", func(d string) string {
			switch {
			case has(d, "ResolveMixinDependencies"):
				return "Mixin"
			case has(d, "ResolveDependencies"):
				return "Deps"
			default:
				return "·"
			}
		}),
		row("Tidy(lib)", "tidyLanguageConfig", func(d string) string { return mark(has(d, "Tidy")) }),
		row("Validate(cfg)", "validateLibraries", func(d string) string { return mark(has(d, "Validate")) }),
		row("Bump(...)", "runBump", func(d string) string { return mark(has(d, "Bump")) }),
		row("Publish(ctx, params)", "publishCommand", func(d string) string { return mark(has(d, "Publish")) }),
		row("Install(ctx, tools)", "installCommand", func(d string) string { return mark(has(d, "Install")) }),
		row("InstallDir()", "debug env", func(d string) string { return mark(has(d, "InstallDir")) }),
		row("release-please extras", "syncToReleasePlease", func(d string) string {
			switch {
			case has(d, "ReleasePleaseExtraFiles"):
				return "ExtraFiles"
			case has(d, "AddManifest"):
				return "Manifest + Package"
			default:
				return "·"
			}
		}),
		row("doc index helpers", "GenerateDocIndex", func(d string) string {
			switch {
			case has(d, "PackageName") && has(d, "DocumentationURL"):
				return "PackageName + DocURL"
			case has(d, "DocumentationURL"):
				return "DocURL"
			default:
				return "·"
			}
		}),
	}
}

var curatedPkgDescriptions = map[string]string{
	"cmd/librarian":                      "<code>main</code> calls <code>librarian.Run</code>. Also holds generated <code>doc.go</code> help text and the multi-language Dockerfile.",
	"tool/cmd":                           "Developer tools: <code>coverage</code> (80% gate enforced in CI), <code>docgen</code> (regenerates <code>cmd/librarian/doc.go</code>), <code>builddockerimages</code>, <code>migrate</code>, and <code>walkthrough</code>.",
	"internal/librarian":                 "Command registration, <code>librarian.yaml</code> loading, defaults and preview resolution, source fetching, language dispatch (<code>switch cfg.Language</code>), and release orchestration.",
	"internal/librarian/dart":            "Dart: in-process sidekick codec producing HTTP/JSON clients; <code>dart format</code>; bump and publish to pub.dev.",
	"internal/librarian/golang":          "Go: three <code>protoc</code> passes with <code>protoc-gen-go</code>, <code>-go-grpc</code> and <code>-go_gapic</code>; <code>go mod tidy</code>; snippet and repo metadata; per-library tags.",
	"internal/librarian/java":            "Java: <code>protoc</code> + gapic-generator-java installed with Maven; sole user of declarative <code>internal/postprocessing</code>; POM and BOM maintenance; generated sequentially.",
	"internal/librarian/nodejs":          "Node.js: gapic-generator-typescript (which drives <code>protoc</code> itself) via pnpm; staging directory then <code>combine-library</code>.",
	"internal/librarian/php":             "PHP: <code>protoc</code> + gapic-generator-php via Composer (plus pip and pnpm for post-processing); <code>owl-bot-staging</code> then <code>php-post-processor</code> and <code>owlbot.py</code>.",
	"internal/librarian/python":          "Python: <code>protoc</code> + gapic-generator-python via pip; <code>owl-bot-staging</code> then synthtool <code>owlbot_main</code>; <code>nox -s format</code>; per-library bump.",
	"internal/librarian/ruby":            "Ruby: <code>protoc</code> + gapic-generator-ruby via gem; multi-wrapper gems and <code>toys</code> tasks.",
	"internal/librarian/rust":            "Rust: in-process sidekick (<code>rust</code> plus <code>rust_prost</code> hybrid for gRPC types); Cargo workspace updates; <code>taplo fmt</code> and <code>cargo fmt</code>; publishes to crates.io.",
	"internal/librarian/swift":           "Swift: in-process sidekick plus <code>protoc --swift_out</code>; <code>swift-format</code>; publishing splits git history into per-package GitHub repositories.",
	"internal/sidekick/rust_prost":       "Hybrid path for Rust gRPC types: runs <code>protoc</code> with prost/tonic and wraps the result with the Rust sidekick codec.",
	"internal/sidekick/rust":             "Rust codec: Go <code>text/template</code> templates; annotates the <code>api.API</code> model with Rust crate names, builders, transports and tracing.",
	"internal/sidekick/swift":            "Swift codec: reference codec structure recommended by <code>internal/sidekick/AGENTS.md</code> for adding new sidekick languages.",
	"internal/sidekick/dart":             "Dart codec: Go templates for HTTP/JSON client libraries.",
	"internal/sidekick/codec_sample":     "Minimal teaching skeleton illustrating how a sidekick language codec attaches annotations and renders templates.",
	"internal/sidekick/language":         "Go <code>text/template</code> wrapper: template directory walking (<code>WalkTemplatesDir</code>), <code>include</code>/<code>indent</code> helpers, strict missing-key mode, AST lint, doc-link rewriting.",
	"internal/sidekick/parser":           "<code>CreateModel</code>: protobuf (via <code>protoc</code> descriptor sets), OpenAPI and Discovery front-ends, mixins, and fixed post-passes producing an <code>api.API</code>.",
	"internal/sidekick/parser/discovery": "Discovery-document specification parser front-end producing <code>api.API</code> nodes.",
	"internal/sidekick/parser/httprule":  "<code>google.api.http</code> path template parser and segment representation.",
	"internal/sidekick/parser/svcconfig": "Service-config helpers shared by specification front-ends.",
	"internal/sidekick/protobuf":         "Resolves <code>.proto</code> input files within an API specification directory.",
	"internal/sidekick/api":              "Language-neutral intermediate representation: <code>API</code>, <code>Service</code>, <code>Method</code>, <code>Message</code>, <code>Field</code>, <code>Enum</code>, resources, pagination, LROs, validation. Sixteen node types carry a <code>Codec any</code> slot.",
	"internal/tool/protoc":               "Downloads pinned <code>protoc</code> releases into cache; <code>BinaryPathOrSystem</code> and <code>RunOrSystem</code> fall back to system <code>PATH</code>.",
	"internal/tool/maven":                "Installs gapic-generator-java and related JARs into <code>java_tools</code>.",
	"internal/tool/pip":                  "Installs Python generator tools and post-processors into an isolated virtual environment.",
	"internal/tool/pnpm":                 "Downloads pnpm and installs Node generator packages such as gapic-generator-typescript.",
	"internal/tool/gem":                  "Installs gapic-generator-ruby and gem dependencies.",
	"internal/tool/composer":             "Downloads Composer and installs PHP generator and post-processor packages.",
	"internal/config":                    "The <code>librarian.yaml</code> schema as pure Go data structs: <code>Config</code>, <code>Sources</code>, <code>Tools</code>, <code>Default</code>, <code>Library</code>, <code>API</code>, <code>Postprocess</code> and language blocks.",
	"internal/serviceconfig":             "Google API service config YAML reader plus embedded <code>sdk.yaml</code> exception overrides consulted by <code>Find</code>.",
	"internal/sources":                   "Resolved source roots (<code>googleapis</code>, <code>discovery</code>, <code>showcase</code>, ...) and path lookup across active roots.",
	"internal/repometadata":              "<code>.repo-metadata.json</code> model and generator.",
	"internal/postprocessing":            "Declarative <code>postprocess:</code> file edits: literal/regex replacements, file copy/remove, and method edits.",
	"internal/snippetmetadata":           "Reformats and updates version pins inside <code>snippet_metadata*.json</code> files.",
	"internal/semver":                    "SemVer parsing, comparison and next-version derivation including preview rules.",
	"internal/proto":                     "Enumerates <code>.proto</code> files and extracts package options.",
	"internal/license":                   "Canonical Apache 2.0 license header formatting across languages.",
	"internal/docuploader":               "<code>docs.metadata.json</code> and documentation tarball builder.",
	"internal/command":                   "Single subprocess runner: <code>Run</code>, <code>Output</code>, <code>RunStreaming</code> with <code>--verbose</code> command echoing.",
	"internal/cache":                     "Manages <code>$LIBRARIAN_CACHE</code> (default <code>~/.cache/librarian</code>) and <code>$LIBRARIAN_BIN</code>.",
	"internal/fetch":                     "GitHub tarball downloader with SHA-256 verification, retries, and cache extraction.",
	"internal/filesystem":                "Directory move-and-merge honoring <code>keep</code> rules, file copy, unzip, and empty directory pruning.",
	"internal/git":                       "Wrappers over the <code>git</code> CLI plus gitignore matching via go-git.",
	"internal/yaml":                      "Central YAML reader/writer with license header preservation, yamlfmt integration, and <code>!REF</code> support.",
	"internal/testhelper":                "Shared test fixtures: temporary git repos, fake binaries, and environment setup.",
	"internal/sample":                    "Canonical sample <code>config.Config</code>, <code>api.API</code>, and service config fixtures for unit tests.",
	"internal/sidekick/api/apitest":      "<code>go-cmp</code> comparison helpers for <code>internal/sidekick/api</code> structs.",
}
