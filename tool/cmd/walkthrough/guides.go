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
	"os"
	"path/filepath"
	"slices"
	"strings"
)

func buildSteps(stats *repoStats) []TourStep {
	rustTmpl := stats.tmplCounts["rust"]
	swiftTmpl := stats.tmplCounts["swift"]
	dartTmpl := stats.tmplCounts["dart"]
	totalTmpl := stats.totalTemplates

	return []TourStep{
		{
			Title: "One binary, one YAML file",
			Body: `<p>Librarian is a single Go binary. <a class="src" data-src="cmd/librarian/main.go">cmd/librarian/main.go</a> does nothing but call <code>librarian.Run(ctx, os.Args...)</code>; the urfave/cli v3 application, every subcommand and all orchestration live in <a class="src" data-src="internal/librarian/librarian.go">internal/librarian</a>. A <code>--verbose</code> flag is handled in a <code>Before</code> hook inside <a class="src" data-src="internal/librarian/librarian.go">librarian.go</a> that sets <code>command.Verbose</code> and the <code>slog</code> level.</p>` +
				`<p>Each command starts by reading <code>librarian.yaml</code> from the <em>current directory</em> with <code>yaml.Read[config.Config]</code>. There is no repository-root discovery: consumers run Librarian from the root of a <code>google-cloud-LANG</code> repository.</p>`,
			Look: []string{"cmd/librarian/main.go", "internal/librarian/librarian.go", "internal/librarian/config.go"},
			Pkg:  "cmd/librarian",
		},
		{
			Title: "librarian.yaml is the schema of everything",
			Body: `<p><a class="src" data-src="internal/config/">internal/config</a> is pure data: structs and constants that map 1:1 to <code>librarian.yaml</code>, enforced by repository rules with zero functions. <code>Config</code> holds <code>language</code>, <code>version</code> (the pinned Librarian release), <code>sources</code> (pinned upstream commits with SHA-256 checksums), <code>tools</code> (toolchain versions), <code>default</code> (per-repository defaults) and <code>libraries</code>.</p>` +
				`<p>A <code>Library</code> names its APIs, output directory, <code>keep</code> rules, an optional <code>preview</code> overlay, declarative <code>postprocess</code> steps and one language block (<code>GoModule</code>, <code>JavaModule</code>, <code>RustCrate</code>, <code>SwiftPackage</code>, ...). The schema document <a class="src" data-src="doc/config-schema.md">doc/config-schema.md</a> is generated from these structs.</p>`,
			Look: []string{"internal/config/config.go", "internal/config/language.go", "doc/config-schema.md"},
			Pkg:  "internal/config",
		},
		{
			Title: "Loading, defaults and preview libraries",
			Body: `<p>Reading is cheap; most of the work is filling in what the file leaves out. <a class="src" data-src="internal/librarian/library.go">internal/librarian/library.go</a> runs <code>applyDefaults</code> (derive <code>apis</code> and <code>output</code> from the name), <code>fillDefaults</code> (copy <code>default.keep</code>, <code>default.output</code>, then <code>fillGo</code>, <code>fillRust</code>, <code>fillDart</code>, ...) and <code>fillLibraryDefaults</code>, which calls language-specific <code>Fill</code> helpers.</p>` +
				`<p>If a <code>LIBRARY-preview</code> variant is requested, <code>resolvePreview</code> overlays the non-zero fields of the <code>preview</code> block and merges the language block. Validation happens in <code>tidy</code> and at the end of <code>add</code> and <code>bump</code>.</p>`,
			Look: []string{"internal/librarian/library.go", "internal/librarian/tidy.go"},
			Pkg:  "internal/librarian",
		},
		{
			Title: "Sources: pinned googleapis tarballs",
			Body: `<p>Librarian never clones googleapis. <code>sources.googleapis</code> (and <code>discovery</code>, <code>showcase</code>, <code>protobuf</code>, <code>conformance</code>) pin a commit and a tarball checksum. <a class="src" data-src="internal/librarian/source.go">internal/librarian/source.go</a> loads all of them in parallel with <code>errgroup</code>; <a class="src" data-src="internal/fetch/">internal/fetch</a> downloads <code>archive/COMMIT.tar.gz</code> from GitHub, verifies SHA-256, retries, and extracts into <code>$LIBRARIAN_CACHE</code> (default <code>~/.cache/librarian</code>).</p>` +
				`<p>The result is a <code>sources.Sources</code> value of resolved directories passed to every generator. <code>librarian update sources.googleapis</code> advances the pin by streaming the latest branch tarball through SHA-256.</p>`,
			Look: []string{"internal/librarian/source.go", "internal/fetch/fetch.go", "internal/sources/sources.go", "internal/librarian/update.go"},
			Pkg:  "internal/fetch",
		},
		{
			Title: "The generate command",
			Body: `<p><a class="src" data-src="internal/librarian/generate.go">internal/librarian/generate.go</a> selects libraries by name, <code>-preview</code> suffix or <code>--all</code>, resolves their defaults, and calls <code>cleanLibraries</code>, which removes previous output while honoring <code>keep</code> rules.</p>` +
				`<p>Generation then runs in parallel across CPU cores (Java runs sequentially due to repo-wide POM/BOM edits): <code>LANG.Generate(ctx, cfg, lib, sources)</code> followed by <code>LANG.Format(ctx, lib)</code> where applicable, closing with repository-level post steps (<code>java.PostGenerate</code>, <code>rust.UpdateWorkspace</code>, or documentation index generation).</p>`,
			Look: []string{"internal/librarian/generate.go", "internal/librarian/clean.go", "internal/librarian/docindex.go"},
			Pkg:  "internal/librarian",
		},
		{
			Title: "Language dispatch: switch, not interface",
			Body:  `<p>There is no Go interface that language packages implement. <code>internal/librarian</code> dispatches explicitly via <code>switch cfg.Language</code> blocks plus maps in <a class="src" data-src="internal/librarian/tidy.go">internal/librarian/tidy.go</a>; each language package exports only the plain functions it supports (<code>Generate</code>, <code>Format</code>, <code>Clean</code>, <code>Install</code>, <code>Bump</code>, <code>Publish</code>, ...). See the <em>Languages</em> tab for the live AST-verified contract table.</p>`,
			Look:  []string{"internal/librarian/generate.go", "internal/librarian/library.go", "internal/librarian/tidy.go"},
			Pkg:   "internal/librarian",
		},
		{
			Title: "Two generation strategies",
			Body: `<p><strong>External micro-generators.</strong> Go, Java, Node.js, PHP, Python and Ruby invoke <code>protoc</code> plus language GAPIC generator plugins placed into <code>$LIBRARIAN_BIN/LANG_tools</code> by <code>librarian install</code> using <a class="src" data-src="internal/tool/protoc/">internal/tool</a> (Maven, pip, pnpm, gem, Composer, or <code>go install</code>).</p>` +
				`<p><strong>In-process sidekick.</strong> Dart, Rust and Swift build a <code>parser.ModelConfig</code>, call <code>parser.CreateModel</code> to create a language-neutral <code>api.API</code> graph, annotate nodes with language-specific types, and render Go templates directly in process. Every subprocess goes through <a class="src" data-src="internal/command/">internal/command</a>.</p>`,
			Look: []string{"internal/librarian/golang/generate.go", "internal/librarian/rust/generate.go", "internal/tool/protoc/protoc.go", "internal/librarian/install.go"},
			Pkg:  "internal/tool/protoc",
		},
		{
			Title: "Inside sidekick: specification to api.API",
			Body: `<p><a class="src" data-src="internal/sidekick/parser/">parser.CreateModel</a> selects a front-end by <code>SpecificationFormat</code>: <code>ParseProtobuf</code> (runs <code>protoc --descriptor_set_out</code> over active source roots), <code>ParseOpenAPI</code> (libopenapi v3) or <code>ParseDisco</code> (Discovery JSON documents).</p>` +
				`<p>Mixins (Locations, IAMPolicy, Operations) are wired in from descriptors when listed in the service config. Fixed normalization passes in <a class="src" data-src="internal/sidekick/api/">internal/sidekick/api</a> then resolve pagination, recursive types, cross-references, resource names, documentation overrides and <a class="src" data-src="internal/serviceconfig/sdk.yaml">sdk.yaml</a> rules.</p>`,
			Look: []string{"internal/sidekick/parser/parser.go", "internal/sidekick/parser/protobuf.go", "internal/sidekick/api/api.go", "internal/serviceconfig/sdk.yaml"},
			Pkg:  "internal/sidekick/parser",
		},
		{
			Title: "Codecs and templates",
			Body: fmt.Sprintf(`<p>A codec such as <a class="src" data-src="internal/sidekick/swift/">sidekick/swift</a> runs <code>annotateModel</code>, attaching a typed struct to the <code>Codec any</code> field of each of the sixteen <code>api</code> node types (API, Service, Method, Message, Field, Enum, OneOf, ...). Templates never compute names; they read pre-computed annotations.</p>`+
				`<p><a class="src" data-src="internal/sidekick/language/">internal/sidekick/language</a> wraps Go <code>text/template</code>: <code>WalkTemplatesDir</code> maps the embedded template tree directly onto output directories. The repository currently contains %d <code>.gotmpl</code> files (Rust %d, Swift %d, Dart %d).</p>`, totalTmpl, rustTmpl, swiftTmpl, dartTmpl),
			Look: []string{"internal/sidekick/language/gotemplate.go", "internal/sidekick/language/walk_templates_dir.go", "internal/sidekick/swift/codec.go"},
			Pkg:  "internal/sidekick/language",
		},
		{
			Title: "Post-processing and metadata",
			Body: `<p>Go, Java, Node.js, PHP, Python, Rust and Swift write <code>.repo-metadata.json</code> through <a class="src" data-src="internal/repometadata/">internal/repometadata</a>; Go and Python refresh <code>snippet_metadata*.json</code> through <a class="src" data-src="internal/snippetmetadata/">internal/snippetmetadata</a>; license headers come from <a class="src" data-src="internal/license/">internal/license</a>.</p>` +
				`<p>PHP and Python stage output under <code>owl-bot-staging</code> and run their post-processors; Node.js runs <code>combine-library</code>; Java runs declarative <a class="src" data-src="internal/postprocessing/">internal/postprocessing</a> edits. Directory moves preserve handwritten files matching <code>keep</code> rules via <a class="src" data-src="internal/filesystem/">internal/filesystem</a>.</p>`,
			Look: []string{"internal/postprocessing/fileops.go", "internal/repometadata/repometadata.go", "internal/librarian/python/generate.go", "internal/librarian/java/postgenerate.go"},
			Pkg:  "internal/postprocessing",
		},
		{
			Title: "The release pipeline",
			Body: `<p>Three commands form Librarian's native release flow: <a class="src" data-src="internal/librarian/bump.go">bump</a> inspects git history from the last tag (or pub.dev for Dart) to compute next SemVer versions and changelog updates; <a class="src" data-src="internal/librarian/publish.go">publish</a> pushes packages to registries; and <a class="src" data-src="internal/librarian/tag.go">tag</a> tags released commits.</p>` +
				`<p>Languages managed by release-please (Go, Node.js, Python, Ruby) have their manifests kept in sync by <a class="src" data-src="internal/librarian/release_please.go">librarian add</a>.</p>`,
			Look: []string{"internal/librarian/bump.go", "internal/librarian/publish.go", "internal/librarian/tag.go", "internal/librarian/release_please.go"},
			Pkg:  "internal/semver",
		},
		{
			Title: "Tests, layering rules and CI",
			Body:  `<p><a class="src" data-src="all_test.go">all_test.go</a> and <a class="src" data-src="dependency_test.go">dependency_test.go</a> enforce license headers, formatting, and architectural package boundaries across the module. CI workflows run <code>go test -race</code> with an 80% coverage threshold via <a class="src" data-src="tool/cmd/coverage/">tool/cmd/coverage</a> and smoke-generate libraries against language repositories.</p>`,
			Look:  []string{"all_test.go", "dependency_test.go", ".github/workflows/ci.yaml", "action.yaml"},
			Pkg:   "tool/cmd",
		},
		{
			Title: "Where to explore next",
			Body: `<p>Jump into any deep-dive path in the <strong>Generation guides</strong> tab or return to <strong>Paths</strong> in the top bar at any time:</p>` +
				`<ul class="list-disc pl-5 space-y-1">` +
				`<li><strong>Sidekick vs. Micro-Generators</strong>: how in-process sidekick compares to external protoc GAPIC plugins.</li>` +
				`<li><strong>Modify Existing Language</strong>: changing annotations or <code>.gotmpl</code> templates in Rust, Swift, or Dart.</li>` +
				`<li><strong>Add a New Language</strong>: scaffolding <code>internal/sidekick/LANG</code> and wiring <code>internal/librarian/LANG</code>.</li>` +
				`<li><strong>New Sources &amp; Custom Outputs</strong>: adding input parsers in <code>sidekick/parser</code> or rendering custom non-SDK outputs via templates.</li>` +
				`</ul>`,
			Look: []string{"CONTRIBUTING.md", ".agents/skills/new-sidekick/SKILL.md", ".agents/skills/walkthrough/SKILL.md"},
		},
	}
}

func buildFlows() []CommandFlow {
	return []CommandFlow{
		{
			ID: "generate", Title: "generate", Cmd: "librarian generate LIBRARY  |  librarian generate --all", File: "internal/librarian/generate.go",
			Inputs: []string{
				"<code>librarian.yaml</code> (repo config)",
				"CLI flags (<code>LIBRARY</code>, <code>-preview</code>, <code>--all</code>)",
				"Pinned tarballs (<code>sources.googleapis</code>, <code>discovery</code>, <code>showcase</code>)",
			},
			Stages: []FlowStage{
				{
					Title: "Load Config & Defaults",
					Where: `<a class="src" data-src="internal/librarian/library.go">library.go</a>`,
					Body:  "Read <code>librarian.yaml</code> from CWD, expand defaults via <code>applyDefaults</code>, and overlay any <code>-preview</code> blocks.",
					Out:   "*config.Config",
				},
				{
					Title: "Fetch & Verify Sources",
					Where: `<a class="src" data-src="internal/librarian/source.go">source.go</a>`,
					Body:  "Parallel <code>errgroup</code> fetches pinned upstream tarballs into <code>$LIBRARIAN_CACHE</code> and verifies SHA-256 digests.",
					Out:   "*sources.Sources",
				},
				{
					Title: "Clean Target Directories",
					Where: `<a class="src" data-src="internal/librarian/clean.go">clean.go</a>`,
					Body:  "Remove stale generated output across selected libraries while preserving handwritten files matching <code>keep:</code> rules.",
					Out:   "clean output tree",
				},
				{
					Title: "Parallel Generate & Format",
					Where: `<a class="src" data-src="internal/librarian/generate.go">internal/librarian/&lt;lang&gt;</a>`,
					Body:  "Worker pool across CPU cores (Java sequential) runs <code>LANG.Generate(...)</code> (Sidekick or <code>protoc</code>) then <code>LANG.Format(...)</code>.",
					Out:   "formatted client code",
				},
				{
					Title: "Repo Post-Processing",
					Where: `<a class="src" data-src="internal/librarian/docindex.go">docindex.go</a>`,
					Body:  "Run <code>java.PostGenerate</code>, <code>rust.UpdateWorkspace</code>, or write documentation metadata indices.",
					Out:   "ready working tree",
				},
			},
		},
		{
			ID: "add", Title: "add", Cmd: "librarian add google/cloud/secretmanager/v1", File: "internal/librarian/add.go",
			Inputs: []string{
				"API path arg (<code>google/cloud/.../vN</code>)",
				"<code>librarian.yaml</code>",
				"Cached <code>sources.googleapis</code> tree",
			},
			Stages: []FlowStage{
				{
					Title: "Validate Upstream API",
					Where: `<a class="src" data-src="internal/librarian/add.go">add.go</a>`,
					Body:  "Ensure <code>sources.googleapis</code> is fetched and confirm the target API directory and service YAML exist.",
					Out:   "verified API path",
				},
				{
					Title: "Resolve Target Library",
					Where: `<a class="src" data-src="internal/librarian/add.go">add.go</a>`,
					Body:  "Locate an existing multi-version package via <code>FindExistingLibraryForNewAPI</code> or derive a new library name.",
					Out:   "*config.Library",
				},
				{
					Title: "Language Add & Dependencies",
					Where: `<a class="src" data-src="internal/librarian/add.go">internal/librarian/&lt;lang&gt;</a>`,
					Body:  "Invoke <code>LANG.Add(...)</code> and resolve mixin or Cargo workspace dependencies.",
					Out:   "updated manifest entries",
				},
				{
					Title: "Sync & Tidy Config",
					Where: `<a class="src" data-src="internal/librarian/tidy.go">tidy.go</a>`,
					Body:  "Sync release-please files via <a class=\"src\" data-src=\"internal/librarian/release_please.go\">release_please.go</a>, strip derivable defaults, and format <code>librarian.yaml</code>.",
					Out:   "tidied librarian.yaml",
				},
			},
		},
		{
			ID: "update", Title: "update", Cmd: "librarian update version sources.googleapis", File: "internal/librarian/update.go",
			Inputs: []string{
				"Target pins (<code>version</code>, <code>sources.NAME</code>)",
				"<code>librarian.yaml</code>",
				"Upstream Git / Go module metadata",
			},
			Stages: []FlowStage{
				{
					Title: "Parse Target Pins",
					Where: `<a class="src" data-src="internal/librarian/update.go">update.go</a>`,
					Body:  "Read <code>librarian.yaml</code> and select requested <code>version</code> or <code>sources.*</code> pins.",
					Out:   "target pin list",
				},
				{
					Title: "Query Upstream HEAD",
					Where: `<a class="src" data-src="internal/librarian/update.go">update.go</a>`,
					Body:  "Resolve latest module release via <code>go list -m</code> or upstream GitHub branch HEAD commit SHA.",
					Out:   "commit SHA / version",
				},
				{
					Title: "Stream SHA-256 Digest",
					Where: `<a class="src" data-src="internal/fetch/">internal/fetch</a>`,
					Body:  "Stream upstream GitHub archive tarball through SHA-256 without writing unverified files.",
					Out:   "verified sha256 pin",
				},
				{
					Title: "Write Formatted YAML",
					Where: `<a class="src" data-src="internal/yaml/">internal/yaml</a>`,
					Body:  "Persist updated pins to <code>librarian.yaml</code> preserving Apache license headers and <code>yamlfmt</code> formatting.",
					Out:   "updated librarian.yaml",
				},
			},
		},
		{
			ID: "release", Title: "bump → publish → tag", Cmd: "librarian bump --all  ·  librarian publish  ·  librarian tag", File: "internal/librarian/bump.go",
			Inputs: []string{
				"Clean git working tree",
				"Git commits since last release tag (or <code>pub.dev</code>)",
				"<code>librarian.yaml</code> + registry credentials",
			},
			Stages: []FlowStage{
				{
					Title: "Detect Changed Libraries",
					Where: `<a class="src" data-src="internal/git/">internal/git</a>`,
					Body:  "Verify clean working tree via <code>git status --porcelain</code> and diff commits since each library's last release tag.",
					Out:   "changed library set",
				},
				{
					Title: "Bump Versions & Changelogs",
					Where: `<a class="src" data-src="internal/librarian/bump.go">bump.go</a>`,
					Body:  "Run <code>LANG.Bump(...)</code> to compute SemVer bumps, update package manifests and changelogs, and tidy config for the release PR.",
					Out:   "release PR commit",
				},
				{
					Title: "Publish Artifacts",
					Where: `<a class="src" data-src="internal/librarian/publish.go">publish.go</a>`,
					Body:  "Publish packages to <code>crates.io</code> (Rust), <code>pub.dev</code> (Dart), or per-package GitHub repositories (Swift).",
					Out:   "published packages",
				},
				{
					Title: "Create Git Release Tags",
					Where: `<a class="src" data-src="internal/librarian/tag.go">tag.go</a>`,
					Body:  "Diff merged release commit and create annotated git tags matching <code>default.tag_format</code>.",
					Out:   "release tags",
				},
			},
		},
	}
}

func buildFindings(stats *repoStats) []Finding {
	importers := make(map[string][]string)
	for _, p := range stats.pkgs {
		for _, dep := range p.Imports {
			importers[dep] = append(importers[dep], p.ID)
		}
	}
	var out []Finding
	out = append(out, Finding{
		Title: "Dispatch uses explicit switches rather than a Go interface",
		Body:  "Language packages export plain functions rather than implementing a monolithic Go interface; adding a language updates explicit <code>switch cfg.Language</code> blocks across <code>internal/librarian</code>.",
		File:  "internal/librarian/generate.go",
		Pkg:   "internal/librarian",
	})
	if hasPkg(stats.pkgs, "internal/sidekick/codec_sample") && len(importers["internal/sidekick/codec_sample"]) == 0 {
		out = append(out, Finding{
			Title: "codec_sample is a standalone reference skeleton",
			Body:  "<code>internal/sidekick/codec_sample</code> compiles and runs unit tests in isolation as a minimal example, while <code>internal/sidekick/swift</code> serves as the full production blueprint.",
			File:  "internal/sidekick/codec_sample/",
			Pkg:   "internal/sidekick/codec_sample",
		})
	}
	if len(importers["internal/postprocessing"]) == 1 && importers["internal/postprocessing"][0] == "internal/librarian/java" {
		out = append(out, Finding{
			Title: "Declarative postprocess: rules are currently Java-only",
			Body:  "<code>config.Library.Postprocess</code> is parsed for all repositories, while <code>internal/postprocessing</code> is invoked by <code>internal/librarian/java</code>.",
			File:  "internal/postprocessing/",
			Pkg:   "internal/postprocessing",
		})
	}
	out = append(out,
		Finding{
			Title: "Layering boundaries are enforced by dependency_test.go",
			Body:  "<code>TestDependencyRules</code> checks non-test imports across every package in the repository so lower layers never accidentally import higher orchestration layers.",
			File:  "dependency_test.go",
			Pkg:   "internal/librarian",
		},
		Finding{
			Title: "librarian.yaml is read strictly from the current directory",
			Body:  "Commands read <code>librarian.yaml</code> in the working directory without walking parent directories; run commands from the target language repository root.",
			File:  "internal/librarian/librarian.go",
			Pkg:   "internal/librarian",
		},
	)
	return out
}

func hasPkg(pkgs []Pkg, id string) bool {
	for _, p := range pkgs {
		if p.ID == id {
			return true
		}
	}
	return false
}

func buildGuides(stats *repoStats, extraDir string) ([]Guide, error) {
	guides := []Guide{
		{
			ID:       "sidekick-vs-micro",
			Title:    "Sidekick vs. External Micro-Generators",
			Subtitle: "How Librarian orchestrates generation and how in-process sidekick compares to external protoc GAPIC plugins",
			Audience: "Generation architecture · Core concepts",
			Steps: []GuideStep{
				{
					Title: "One command, two generator engines",
					Runs:  "librarian generate",
					Code:  "internal/librarian/generate.go",
					Body: `<p>Every client library repository uses the exact same entry point: <code>librarian generate [LIBRARY | --all]</code>. Orchestration in <a class="src" data-src="internal/librarian/generate.go">internal/librarian/generate.go</a> handles source tarball caching, library default expansion, preview overlays, output directory cleaning (respecting <code>keep</code> lists), parallel worker pools, and post-generation steps.</p>` +
						`<p>Inside <code>LANG.Generate(ctx, cfg, lib, sources)</code>, however, languages split into two architectural models: <strong>external micro-generators</strong> (6 languages) and <strong>in-process sidekick codecs</strong> (3 languages: Dart, Rust, Swift).</p>`,
					Look: []string{"internal/librarian/generate.go", "internal/librarian/clean.go"},
					Pkg:  "internal/librarian",
				},
				{
					Title: "External micro-generators (Go, Java, Node.js, PHP, Python, Ruby)",
					Runs:  "Subprocess via protoc",
					Code:  "internal/tool/ + internal/librarian/<lang>",
					Body: `<p>Historically, each Google Cloud language team built a standalone GAPIC generator plugin for <code>protoc</code> in its own repository (such as <code>protoc-gen-go_gapic</code>, <code>gapic-generator-java</code>, <code>gapic-generator-python</code>, and <code>gapic-generator-typescript</code>).</p>` +
						`<p>For these languages, <code>librarian install</code> uses <a class="src" data-src="internal/tool/protoc/">internal/tool/*</a> (Maven, pip, pnpm, gem, Composer, or <code>go install</code>) to download pinned generator binaries into <code>$LIBRARIAN_BIN/LANG_tools</code>. During <code>librarian generate</code>, <a class="src" data-src="internal/librarian/golang/generate.go">internal/librarian/golang</a> (and sibling packages) assemble <code>protoc</code> flags, invoke <a class="src" data-src="internal/tool/protoc/protoc.go">protoc.RunOrSystem</a>, stage output files, and run post-processors (like Python's synthtool or Node's <code>combine-library</code>).</p>`,
					Look: []string{"internal/tool/protoc/protoc.go", "internal/librarian/golang/generate.go", "internal/librarian/python/generate.go"},
					Pkg:  "internal/tool/protoc",
				},
				{
					Title: "In-process Sidekick (Dart, Rust, Swift)",
					Runs:  "In-process Go code",
					Code:  "internal/sidekick/",
					Body: `<p>Sidekick brings code generation directly inside the Librarian Go binary so every language shares one front-end parser, one semantic model, and one template engine with zero subprocess overhead per template file:</p>` +
						`<ol class="list-disc pl-5 space-y-1">` +
						`<li><strong>Front-end (<a class="src" data-src="internal/sidekick/parser/parser.go">internal/sidekick/parser</a>)</strong>: Parses Protobuf descriptors, OpenAPI v3 specs, or Discovery JSON into a single normalized graph (<a class="src" data-src="internal/sidekick/api/">internal/sidekick/api.API</a>).</li>` +
						`<li><strong>Model normalization</strong>: Resolves pagination, LRO polling types, resource patterns, recursive field boxing, cross-references, and <a class="src" data-src="internal/serviceconfig/sdk.yaml">sdk.yaml</a> overrides once for all languages.</li>` +
						`<li><strong>Language Codec (<a class="src" data-src="internal/sidekick/swift/">sidekick/swift</a>, <a class="src" data-src="internal/sidekick/rust/">sidekick/rust</a>, <a class="src" data-src="internal/sidekick/dart/">sidekick/dart</a>)</strong>: Attaches typed language metadata to each AST node's <code>Codec any</code> field and renders embedded Go <code>.gotmpl</code> templates directly to disk.</li>` +
						`</ol>`,
					Look: []string{"internal/sidekick/parser/parser.go", "internal/sidekick/api/model.go", "internal/sidekick/language/walk_templates_dir.go"},
					Pkg:  "internal/sidekick/parser",
				},
				{
					Title: "Hybrid generators (Rust gRPC + Swift Protobufs)",
					Runs:  "Sidekick + protoc",
					Code:  "internal/sidekick/rust_prost + internal/librarian/swift",
					Body: `<p>Languages do not have to choose strictly all-or-nothing:</p>` +
						`<ul class="list-disc pl-5 space-y-1">` +
						`<li><strong>Swift</strong> uses sidekick templates for idiomatic client services, pagination, LROs, and HTTP/JSON serialization, while invoking <code>protoc --swift_out</code> in <a class="src" data-src="internal/librarian/swift/">internal/librarian/swift</a> when generating wire-level SwiftProtobuf types.</li>` +
						`<li><strong>Rust</strong> uses <a class="src" data-src="internal/sidekick/rust_prost/">internal/sidekick/rust_prost</a> to run <code>protoc</code> with <code>prost</code>/<code>tonic</code> for raw gRPC transport crates while wrapping and idiomatizing public crates through the Rust sidekick codec.</li>` +
						`</ul>`,
					Look: []string{"internal/sidekick/rust_prost/", "internal/librarian/swift/", "internal/librarian/rust/"},
					Pkg:  "internal/sidekick/rust_prost",
				},
			},
		},
		{
			ID:       "modify-language",
			Title:    "Modifying Sidekick Generation for an Existing Language",
			Subtitle: "Step-by-step workflow for changing generated code, annotations, or templates in Rust, Swift, or Dart",
			Audience: "Hands-on contributor guide · Existing Sidekick languages",
			Steps: []GuideStep{
				{
					Title: "Locate the template emitting the generated file",
					Runs:  "Code inspection",
					Code:  "internal/sidekick/<lang>/templates/",
					Body: fmt.Sprintf(`<p>Every file generated by sidekick corresponds directly to a template file inside <code>internal/sidekick/&lt;lang&gt;/templates/</code> (currently %d templates in Rust, %d in Swift, and %d in Dart).</p>`+
						`<p><a class="src" data-src="internal/sidekick/language/walk_templates_dir.go">language.WalkTemplatesDir</a> mirrors directory layouts: top-level files render once per library (e.g., <code>Package.swift.gotmpl</code> or <code>Cargo.toml.gotmpl</code>), while templates inside service/message/enum subdirectories render per model element using partials invoked with <code>{{include "partial_name" .}}</code>.</p>`,
						stats.tmplCounts["rust"], stats.tmplCounts["swift"], stats.tmplCounts["dart"]),
					Look: []string{"internal/sidekick/swift/templates/", "internal/sidekick/rust/templates/", "internal/sidekick/dart/templates/"},
					Pkg:  "internal/sidekick/swift",
				},
				{
					Title: "Keep logic out of templates: update Codec annotations in Go",
					Runs:  "Go code edit",
					Code:  "internal/sidekick/<lang>/",
					Body: `<p>A strict design principle across all sidekick codecs is that <strong>templates never compute identifier casing, imports, type mappings, or conditional protocol rules</strong>. Instead, each language codec defines strongly typed annotation structs attached to the <code>Codec any</code> slot on <code>api.API</code>, <code>api.Service</code>, <code>api.Method</code>, <code>api.Message</code>, <code>api.Field</code>, and <code>api.Enum</code>.</p>` +
						`<p>If your change only tweaks formatting or doc comments, edit the <code>.gotmpl</code> file directly. If it introduces new conditional behavior, helper types, or imports, add a field to the corresponding annotation struct (for example in <a class="src" data-src="internal/sidekick/swift/codec.go">internal/sidekick/swift/codec.go</a>) and populate it during <code>annotateModel</code>.</p>`,
					Look: []string{"internal/sidekick/swift/codec.go", "internal/sidekick/rust/annotate_model.go", "internal/sidekick/dart/annotate.go"},
					Pkg:  "internal/sidekick/rust",
				},
				{
					Title: "Verify with fast in-process codec unit tests",
					Runs:  "go test",
					Code:  "internal/sidekick/<lang>/*_test.go",
					Body: `<p>Because sidekick runs completely in-process over synthetic <a class="src" data-src="internal/sample/">internal/sample</a> models or small test protos without network access or external generator plugins, unit tests run in milliseconds:</p>` +
						`<p><code>go test -race ./internal/sidekick/swift/...</code> (or <code>rust</code> / <code>dart</code>).</p>` +
						`<p>Add or update targeted unit tests (such as <code>generate_method_signatures_test.go</code> or <code>generate_message_swift_test.go</code>) asserting on the generated snippet before testing against a full language monorepo.</p>`,
					Look: []string{"internal/sidekick/swift/generate_service_swift_test.go", "internal/sample/"},
					Pkg:  "internal/sample",
				},
				{
					Title: "Smoke-test against a real library repository",
					Runs:  "librarian generate",
					Code:  "cmd/librarian",
					Body: `<p>From a local checkout of <code>google-cloud-rust</code>, <code>google-cloud-swift</code>, or <code>google-cloud-dart</code>, run your local Librarian checkout directly:</p>` +
						`<p><code>go run /path/to/librarian/cmd/librarian generate secretmanager</code></p>` +
						`<p>Inspect <code>git diff</code> in the client repository and run the target language's compiler and formatter to verify end-to-end validity.</p>`,
					Look: []string{"cmd/librarian/main.go", "internal/librarian/generate.go"},
					Pkg:  "cmd/librarian",
				},
			},
		},
		{
			ID:       "new-language",
			Title:    "Adding a New Language Generator with Sidekick",
			Subtitle: "End-to-end blueprint for onboarding a new target language into Librarian and Sidekick",
			Audience: "New language authors · Architecture blueprint",
			Steps: []GuideStep{
				{
					Title: "Start from the Swift codec and new-sidekick skill",
					Runs:  "Scaffolding",
					Code:  ".agents/skills/new-sidekick + internal/sidekick/swift",
					Body: `<p><a class="src" data-src="internal/sidekick/swift/">internal/sidekick/swift</a> is the reference architecture for new sidekick generators, and <a class="src" data-src=".agents/skills/new-sidekick/SKILL.md">.agents/skills/new-sidekick</a> guides agents in scaffolding decomposed, well-tested codecs.</p>` +
						`<p>Adding a language spans three layers: configuration structs in <code>internal/config</code>, the pure template codec in <code>internal/sidekick/LANG</code>, and lifecycle hooks in <code>internal/librarian/LANG</code>.</p>`,
					Look: []string{".agents/skills/new-sidekick/SKILL.md", "internal/sidekick/swift/", "internal/sidekick/codec_sample/"},
					Pkg:  "internal/sidekick/swift",
				},
				{
					Title: "1. Add pure data schema structs in internal/config",
					Runs:  "Schema definition",
					Code:  "internal/config/",
					Body:  `<p>Define a <code>LanguageLANG = "lang"</code> constant and pure YAML data structs in <a class="src" data-src="internal/config/">internal/config</a> (e.g., package naming overrides, dependency pins, or module options on <code>config.Library</code> and <code>config.Default</code>). Remember: <code>internal/config</code> contains zero functions or business logic.</p>`,
					Look:  []string{"internal/config/config.go", "internal/config/language.go"},
					Pkg:   "internal/config",
				},
				{
					Title: "2. Build the codec in internal/sidekick/<lang>",
					Runs:  "Codec + templates",
					Code:  "internal/sidekick/<lang>/",
					Body: `<p>Create <code>internal/sidekick/LANG</code> importing only <code>internal/sidekick/api</code>, <code>internal/sidekick/language</code>, <code>internal/config</code>, and <code>internal/license</code>:</p>` +
						`<ul class="list-disc pl-5 space-y-1">` +
						`<li><strong>Model annotations (<code>annotate_model.go</code>)</strong>: Walk <code>model.Services</code>, <code>model.Messages</code>, and <code>model.Enums</code>, attaching typed metadata (escaped keywords, camelCase/PascalCase identifiers, import lists, pagination helpers) to each node's <code>Codec</code> field.</li>` +
						`<li><strong>Embedded templates (<code>templates/*.gotmpl</code>)</strong>: Use <code>//go:embed all:templates</code> and pass the embedded FS to <a class="src" data-src="internal/sidekick/language/walk_templates_dir.go">language.WalkTemplatesDir</a>.</li>` +
						`</ul>`,
					Look: []string{"internal/sidekick/codec_sample/", "internal/sidekick/language/walk_templates_dir.go"},
					Pkg:  "internal/sidekick/language",
				},
				{
					Title: "3. Implement internal/librarian/<lang> and register dispatch sites",
					Runs:  "CLI wiring",
					Code:  "internal/librarian/<lang>/ + internal/librarian/",
					Body: `<p>Create <code>internal/librarian/LANG</code> exporting plain lifecycle functions: <code>Generate(ctx, cfg, lib, sources)</code> (calls <code>parser.CreateModel</code> then your sidekick codec), <code>Format(ctx, lib)</code> (invokes your language formatter via <code>command.Run</code>), <code>DefaultOutput</code>, <code>DefaultLibraryName</code>, and <code>Add</code>.</p>` +
						`<p>Add <code>case config.LanguageLANG:</code> to the dispatch switches in <a class="src" data-src="internal/librarian/generate.go">generate.go</a>, <a class="src" data-src="internal/librarian/library.go">library.go</a>, <a class="src" data-src="internal/librarian/add.go">add.go</a>, and <a class="src" data-src="internal/librarian/tidy.go">tidy.go</a>. Run <code>go test ./...</code> — <code>TestDependencyRules</code> automatically verifies your imports respect all layer boundaries!</p>`,
					Look: []string{"internal/librarian/dart/", "internal/librarian/generate.go", "dependency_test.go"},
					Pkg:  "internal/librarian/dart",
				},
			},
		},
		{
			ID:       "custom-source-output",
			Title:    "New Input Sources & Custom Non-Cloud-SDK Outputs",
			Subtitle: "Adding specification parsers in sidekick/parser or generating non-Cloud-SDK outputs purely via custom templates",
			Audience: "Platform engineers · Custom code generation & new specification formats",
			Steps: []GuideStep{
				{
					Title: "Architecture split: front-end parsers vs. template back-ends",
					Runs:  "Architecture",
					Code:  "internal/sidekick/parser + internal/sidekick/language",
					Body: `<p>Sidekick cleanly separates <em>what API is being described</em> (<a class="src" data-src="internal/sidekick/parser/">internal/sidekick/parser</a> → <a class="src" data-src="internal/sidekick/api/">internal/sidekick/api.API</a>) from <em>what files get written to disk</em> (<a class="src" data-src="internal/sidekick/language/">internal/sidekick/language</a> + <code>templates/</code>).</p>` +
						`<p>Because of this split, supporting a new input specification format touches only <code>parser</code>, while generating completely non-Cloud-SDK artifacts (server stubs, CLI commands, Terraform resources, custom internal clients, or documentation bundles) is driven almost entirely by templates!</p>`,
					Look: []string{"internal/sidekick/parser/parser.go", "internal/sidekick/api/model.go", "internal/sidekick/language/walk_templates_dir.go"},
					Pkg:  "internal/sidekick/api",
				},
				{
					Title: "Generating custom / non-Cloud-SDK outputs (template-driven)",
					Runs:  "Templates + Codec",
					Code:  "internal/sidekick/<target>/templates/",
					Body: `<p>If your input is already Protobuf, OpenAPI v3, or Google Discovery JSON, <strong>you do not need to write any parser code at all</strong>:</p>` +
						`<ul class="list-disc pl-5 space-y-1">` +
						`<li><strong>Same language, different artifact shape</strong> (e.g., generating a CLI tool, mock server, MCP tool server, or custom internal framework client instead of a Google Cloud SDK): create a dedicated template directory tree (e.g. <code>templates/server/</code> or a new lightweight target package in <code>internal/sidekick/</code>) passed to <a class="src" data-src="internal/sidekick/language/walk_templates_dir.go">language.WalkTemplatesDir</a>.</li>` +
						`<li>Every service, RPC method, HTTP binding rule (<code>httprule</code>), pagination token, request field, and doc comment is already pre-resolved on <code>api.API</code> — your <code>.gotmpl</code> files simply emit whatever project layout, manifest files, and source files you want.</li>` +
						`</ul>`,
					Look: []string{"internal/sidekick/language/walk_templates_dir.go", "internal/sidekick/language/gotemplate.go"},
					Pkg:  "internal/sidekick/language",
				},
				{
					Title: "Adding a brand-new input specification source format",
					Runs:  "Front-end parser",
					Code:  "internal/sidekick/parser/",
					Body: `<p>If your API specifications live in a new source format beyond Protobuf (<code>ParseProtobuf</code>), OpenAPI v3 (<code>ParseOpenAPI</code>), or Discovery JSON (<code>ParseDisco</code>):</p>` +
						`<ol class="list-disc pl-5 space-y-1">` +
						`<li>Add a parser front-end function in <a class="src" data-src="internal/sidekick/parser/">internal/sidekick/parser</a> (modeled on <a class="src" data-src="internal/sidekick/parser/discovery/">parser/discovery</a>) that reads your specification files and constructs an <code>*api.API</code> populated with <code>Services</code>, <code>Messages</code>, <code>Fields</code>, and <code>Enums</code>.</li>` +
						`<li>Wire a <code>SpecificationFormat</code> branch in <code>parser.CreateModel</code> (<a class="src" data-src="internal/sidekick/parser/parser.go">parser.go</a>).</li>` +
						`<li>Run the existing normalization passes (<code>CrossReference</code>, <code>UpdateMethodPagination</code>, <code>LabelRecursiveFields</code>, <code>Validate</code>) so every existing sidekick codec and template works immediately with the new source!</li>` +
						`</ol>`,
					Look: []string{"internal/sidekick/parser/parser.go", "internal/sidekick/parser/discovery/", "internal/sources/sources.go"},
					Pkg:  "internal/sidekick/parser/discovery",
				},
			},
		},
	}

	custom, err := loadCustomGuides(extraDir)
	if err != nil {
		return nil, err
	}
	guides = append(guides, custom...)
	return guides, nil
}

func loadCustomGuides(extraDir string) ([]Guide, error) {
	if extraDir == "" {
		return nil, nil
	}
	entries, err := os.ReadDir(extraDir)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var out []Guide
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		path := filepath.Join(extraDir, entry.Name())
		raw, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		trimmed := strings.TrimSpace(string(raw))
		if strings.HasPrefix(trimmed, "[") {
			var list []Guide
			if err := json.Unmarshal(raw, &list); err != nil {
				return nil, fmt.Errorf("parsing %s: %w", path, err)
			}
			for i := range list {
				list[i].Custom = true
			}
			out = append(out, list...)
		} else {
			var g Guide
			if err := json.Unmarshal(raw, &g); err != nil {
				return nil, fmt.Errorf("parsing %s: %w", path, err)
			}
			g.Custom = true
			out = append(out, g)
		}
	}
	slices.SortFunc(out, func(a, b Guide) int { return strings.Compare(a.ID, b.ID) })
	return out, nil
}
