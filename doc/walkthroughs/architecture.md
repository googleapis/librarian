---
title: Librarian architecture tour
summary: A map of the repository's packages by layer, then a guided pass from process start to release, with the implicit language contract and a list of observations about where code and documentation have drifted.
audience: For engineers new to the repository who want the big picture first
order: 0
nav: Architecture
---

Librarian is one Go module with about fifty packages. This tour groups them
into layers, then walks the layers in the order a command passes through
them. It is the entry point; the other walkthroughs go deeper into the
[command workflow](workflow.html) and the
[generation engine](generate-and-sidekick.html). The map below and the
tables on this page are derived from the tree when the site is built, so
they cannot go stale; only the assignment of packages to layers is written
by hand, and a new package that fits no layer fails the build.

```generated
kind: imports
layers:
  - id: entry
    label: Entry points
    kind: user
    match: [cmd, cmd/, tool/]
  - id: orch
    label: Command orchestration
    kind: lib
    match: [internal/librarian]
  - id: lang
    label: Language integrations
    kind: lib
    match: [internal/librarian/]
  - id: test
    label: Test support
    kind: user
    match: [internal/testhelper, internal/sample, internal/sidekick/api/apitest]
  - id: engine
    label: Generation engine
    kind: lib
    match: [internal/sidekick/]
  - id: tools
    label: Tool installers
    kind: lib
    match: [internal/tool/]
  - id: domain
    label: Domain services
    kind: lang
    match:
      - internal/config
      - internal/serviceconfig
      - internal/sources
      - internal/repometadata
      - internal/postprocessing
      - internal/snippetmetadata
      - internal/semver
      - internal/proto
      - internal/license
      - internal/docuploader
  - id: infra
    label: Infrastructure
    kind: ext
    match:
      - internal/command
      - internal/cache
      - internal/fetch
      - internal/filesystem
      - internal/git
      - internal/yaml
```

## The package map
<!-- step runs="Reading" code="librarian" -->

Lower layers know nothing about the layers above them: infrastructure does
not know about configuration, domain services do not know about commands,
and only `internal/librarian` knows the names of the languages. Reading the
import paths top-down: `cmd/` and `tool/cmd/` are entry points,
`internal/librarian` is command orchestration with one subpackage per
language, `internal/sidekick` is the generation engine, `internal/tool`
installs external generators, and the remaining `internal/*` packages are
domain services and infrastructure.

```generated
kind: packages
```

The rules that keep the layers apart are not a convention; they are a test.
`TestDependencyRules` reads the import graph with `go list` and fails on any
forbidden edge, so a pull request cannot cross a layer by accident.

```excerpt
file: dependency_test.go
symbol: dependencyRules
```

## One binary, one YAML file
<!-- step runs="Developer machine or CI in a language repository" code="librarian" -->

`main` does nothing but hand control to `internal/librarian`, where the
`urfave/cli` application, every subcommand and all orchestration live.

```excerpt
file: cmd/librarian/main.go
symbol: main
caption: The whole entry point.
```

The commands, read from the `cli.Command` literals in `internal/librarian`:

```generated
kind: commands
```

Every command begins by reading `librarian.yaml` from the current
directory. There is no repository-root discovery and no `--config` flag;
consumers run Librarian from the root of a `google-cloud-<lang>` repository.

```excerpt
file: internal/yaml/yaml.go
symbol: Read
caption: Generic YAML read; config.LibrarianYAML is a relative path.
```

## librarian.yaml is the schema of everything
<!-- step runs="Language repository" code="librarian (internal/config)" -->

`internal/config` maps 1:1 onto the file. `Config` holds the language, the
pinned Librarian version, pinned upstream sources with checksums, tool
versions, repository-wide defaults and the list of libraries.

```excerpt
file: internal/config/config.go
symbol: Config
end: "Libraries []*Library"
caption: The top level of librarian.yaml; doc/config-schema.md is generated from these comments.
```

A `Library` names its APIs, output directory, `keep` rules, an optional
`preview` overlay, declarative `postprocess` steps and one language block.
There is no other state file: `librarian.yaml` plus git history is
everything Librarian knows about a repository.

## Loading, defaults and preview libraries
<!-- step runs="In memory" code="librarian (internal/librarian/library.go)" -->

Reading is cheap; most of the work is filling in what the file leaves out.
`applyDefaults` derives the API path and output directory from the library
name, copies repository defaults, and calls the per-language `fill`
functions; `resolvePreview` overlays the `preview` block when a
`<name>-preview` variant is requested.

```excerpt
file: internal/librarian/library.go
symbol: applyDefaults
caption: The resolution pipeline every command runs before touching a library.
```

Validation (`validateTools`, `validateLibraries` and the Java and PHP
validators) happens in `tidy` and at the end of `add` and `bump`;
`generate` does not validate.

## Sources: pinned tarballs, never clones
<!-- step runs="Developer machine or CI" code="librarian + GitHub" -->

`sources.googleapis` (and `discovery`, `showcase`, `protobuf`,
`conformance`) pin a commit and a tarball checksum. `fetchSource` resolves
each one to a directory in `$LIBRARIAN_CACHE`, downloading
`archive/<commit>.tar.gz` and verifying the SHA-256 only on a cache miss.

```excerpt
file: internal/librarian/source.go
symbol: fetchSource
caption: A dir override skips the network entirely, which is how tests and local experiments work.
```

`librarian update sources.googleapis` moves the pin by asking the GitHub API
for the branch head and streaming the new tarball through SHA-256; see
[the workflow walkthrough](workflow.html#step-5).

## The generate command
<!-- step runs="Developer machine or CI" code="librarian" -->

`generate` selects libraries by name, `-preview` suffix or `--all`, resolves
defaults, cleans previous output while honoring `keep`, then dispatches to
the language. Generation runs in parallel up to `NumCPU` libraries for most
languages; Java is sequential because its post step rewrites shared POM
files.

```excerpt
file: internal/librarian/generate.go
symbol: runGenerate
end: "+30"
```

## Language dispatch: switch, not interface
<!-- step runs="In memory" code="librarian" -->

There is no Go interface a language package implements. `internal/librarian`
has around fifteen `switch cfg.Language` sites plus two maps in `tidy.go`;
each language package exports the plain functions it supports and each
switch calls the ones it needs.

```excerpt
file: internal/librarian/generate.go
symbol: generateLibraries
end: "+20"
highlight: "switch cfg.Language"
```

The implicit contract is every declaration in `internal/librarian` that
names a language constant. The matrix below is read from the source each
time the site is built; a new language is complete when its column is as
full as its neighbours'.

```generated
kind: dispatch
```

A `fake` language exists as a test seam, and .NET has a configuration block
but no package. Adding a language means visiting every row above; [Adding a
new language to sidekick](extend-new-language.html) is that checklist.

## Two generation strategies
<!-- step runs="Developer machine or CI" code="librarian + external generators" -->

**External generators.** Go, Java, Node.js, PHP, Python and Ruby shell out
to `protoc` plus a GAPIC generator plugin. `librarian install` puts those
tools in `$LIBRARIAN_BIN/<lang>_tools` using `internal/tool` (Maven, pip,
pnpm, gem, Composer or `go install`), and the pinned `protoc` runs with that
directory on `PATH`.

```excerpt
file: internal/tool/protoc/protoc.go
symbol: RunOrSystem
caption: The pinned protoc, falling back to whatever is on PATH.
```

**In-process sidekick.** Dart, Rust and Swift build a `parser.ModelConfig`,
call `parser.CreateModel`, and render Go templates directly. Rust adds a
`rust_prost` hybrid for gRPC types and Swift a `protoc --swift_out` pass.
Every subprocess in either strategy goes through `internal/command`, so
`-v` prints each command line.

```excerpt
file: internal/command/command.go
symbol: Run
caption: The one way to run a subprocess.
```

## Inside sidekick: specification to api.API
<!-- step runs="In memory" code="librarian (internal/sidekick)" -->

`parser.CreateModel` picks a front end by specification format (protobuf
through `protoc --descriptor_set_out`, OpenAPI v3, or Discovery), then runs a
fixed sequence of passes from `internal/sidekick/api`: pagination detection,
recursive-field labelling, cross-referencing, resource identification,
include and skip lists, documentation patches and validation.

```excerpt
file: internal/sidekick/parser/parser.go
symbol: CreateModel
end: "+20"
highlight: "switch cfg.SpecificationFormat"
```

`internal/serviceconfig` contributes titles, descriptions and the embedded
`sdk.yaml` exception list. The result is a language-neutral `api.API` whose
nodes each carry a `Codec any` slot for language annotations. [Inside
generate](generate-and-sidekick.html) follows this path stage by stage.

## Codecs and templates
<!-- step runs="In memory, then the output directory" code="librarian (internal/sidekick/<lang>)" -->

A codec's `annotateModel` fills the `Codec` slots with language-specific
structs; templates never compute names, they read annotations. The template
engine in `internal/sidekick/language` wraps Go `text/template` with
template discovery, two helpers (`include`, `indent`), strict missing-key
mode and a lint that bans logic in templates.

```excerpt
file: internal/sidekick/language/walk_templates_dir.go
symbol: WalkTemplatesDir
end: "+24"
caption: A template with two dots in its name is an output file; one dot is a partial.
```

All templates are `.gotmpl`; the Mustache migration mentioned in
`internal/sidekick/AGENTS.md` is complete. The language package's `Format`
then runs `cargo fmt` and `taplo`, `swift-format` or `dart format`.

## Post-processing and metadata
<!-- step runs="Developer machine or CI" code="librarian + language-repository hooks" -->

Generated code is rarely the final artifact. Most languages write
`.repo-metadata.json` through `internal/repometadata`; Go and Python refresh
`snippet_metadata*.json`; license headers come from `internal/license`. PHP
and Python generate into `owl-bot-staging` and run their historical
post-processors; Node.js stages and runs `combine-library`.

```excerpt
file: internal/postprocessing/fileops.go
symbol: Apply
caption: The declarative postprocess block; parsed for every library, applied only by Java.
```

File moves honor `keep` rules through `internal/filesystem`, which is what
protects handwritten files across regenerations.

## The release pipeline
<!-- step runs="Release automation in the language repository" code="librarian + git" -->

Three hidden commands form the release flow for the languages Librarian
releases itself. `bump` requires a clean tree, finds changed libraries from
git tags and diffs, and asks the language package for the next version and
manifest edits. After the release PR merges, `publish` pushes packages (Dart
to pub.dev, Rust to crates.io, Swift into per-package repositories) and
`tag` compares the two latest versions of `librarian.yaml` to find released
libraries and creates local tags.

```excerpt
file: internal/librarian/bump.go
symbol: runBump
end: "+24"
```

Go, Node.js, Python and Ruby rely on release-please instead; `librarian add`
keeps its manifest and config files in sync. The [workflow
walkthrough](workflow.html#step-10) covers all three commands in detail.

## Tests, CI and consumers
<!-- step runs="GitHub Actions in this repository" code="librarian CI" -->

`all_test.go` enforces repository-wide rules: license headers, imports,
formatting, `go mod tidy` and `go generate` cleanliness;
`dependency_test.go` enforces the layering from step 1, and
`tool/cmd/walkthrough` fails `go test` when a page on this site points at a
line that no longer exists.

```excerpt
file: all_test.go
symbol: TestAddLicense
end: "+15"
```

One workflow per language sparse-checks-out the matching consumer
repository, runs `librarian install` and a smoke `librarian generate` for
one library on pull requests, and `generate --all` after merges; every job
runs through `tool/cmd/coverage` with an 80% gate.

```generated
kind: workflows
```

Consumers never vendor
Librarian: they pin `version:` in `librarian.yaml` and run
`go run github.com/googleapis/librarian/cmd/librarian@<version>`, or use the
composite action at the repository root.

```excerpt
file: action.yaml
start: "- name: Install librarian"
end: "+6"
```

## Observations
<!-- step runs="Reading" code="findings" -->

Places where the code and its documentation have drifted, or where a
package is unused. None of them block anything; each is verified against
the commit this site was built from, and the excerpts below will fail the
build if the underlying line changes.

- **No language interface.** Dispatch is a set of switches; a Go interface
  or registry would make the contract explicit (see step 7).
- **`internal/docuploader` is unused.** No package outside its own tests
  imports it.
- **`codec_sample` is unreferenced.** It is a teaching skeleton, and the
  `new-sidekick` skill points at it, but `internal/sidekick/AGENTS.md`
  names Swift as the reference.
- **`postprocessing` is Java-only.** The `postprocess:` block is parsed for
  every library but applied only by the Java package.
- **`AGENTS.md` still describes Mustache templates** although every
  template is `.gotmpl`.
- **`generate` help text disagrees with `update`.** The help says
  `librarian update googleapis`; the command takes `sources.googleapis`.

```excerpt
file: internal/librarian/generate.go
start: "librarian update googleapis"
end: "+1"
caption: Stale help text; update takes sources.<name>.
```

- **Ruby `Format` is a no-op** and tracked by an issue.

```excerpt
file: internal/librarian/ruby/format.go
start: "// TODO(https://github.com/googleapis/librarian/issues/6633)"
end: "+3"
```

- **Dart formats twice**, once inside the codec and once from
  `internal/librarian/dart`.
- **No Swift end-to-end job.** The `sidekick.yaml` integration job
  regenerates only `google-cloud-rust`.
- **`librarian.yaml` is read from the current directory**, with no walk-up
  to the repository root; worth stating prominently for consumers.

## Where to go next
<!-- step runs="Reading" code="walkthroughs" -->

- [The Librarian workflow, command by command](workflow.html): what each
  command does, where it runs, and which consumer repositories automate it.
- [Inside generate: how sidekick produces code](generate-and-sidekick.html):
  the engine stage by stage.
- [Adding a new language to sidekick](extend-new-language.html),
  [Supporting a different input format](extend-new-input.html) and
  [Changing what sidekick generates for one language](modify-language-generation.html):
  the three extension questions.

A good first exercise: pick a language from the dispatch matrix, run
`librarian install` and `librarian generate <one library>` in a sparse
checkout of its repository, and follow the calls with `-v`.
