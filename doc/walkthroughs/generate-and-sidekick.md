---
title: "Inside generate: how sidekick produces code"
summary: Follow librarian generate from librarian.yaml to formatted source files for the languages whose generator lives inside Librarian, and see which inputs it reads, what is shared across languages, and what each language owns.
audience: For engineers who want to understand the generate step and the sidekick engine
order: 2
nav: Inside generate
---

`librarian generate` is one command with two very different implementations
behind it. For Go, Java, Node.js, PHP, Python and Ruby, Librarian builds a
`protoc` command line and an upstream GAPIC generator does the work. For Rust,
Dart and Swift, Librarian *is* the generator: a Go package tree under
`internal/sidekick` parses the API definition into a language-neutral model,
annotates it with language-specific names and types, and renders Go
`text/template` files. That in-process engine is called **sidekick**, and this
walkthrough follows it stage by stage.

The short answer to "does it depend on anything in the language repository?"
is no: every parser, annotation and template is compiled into the Librarian
binary. The language repository contributes `librarian.yaml` (which APIs, which
options, which pinned sources), `keep` lists for handwritten files, and the
ecosystem formatter on `PATH`.

```flow
title: One library through the sidekick pipeline
lanes:
  - id: lang
    label: Language repository
    kind: lang
  - id: lib
    label: internal/librarian/<lang>
    kind: lib
  - id: sk
    label: internal/sidekick (shared)
    kind: lib
  - id: ext
    label: External processes
    kind: ext
nodes:
  - id: yaml
    lane: lang
    label: librarian.yaml + pinned sources
    step: 2
    col: 0
  - id: mc
    lane: lib
    label: Library -> ModelConfig
    step: 4
    col: 1
  - id: parse
    lane: sk
    label: parser.CreateModel
    step: 5
    col: 2
  - id: protoc
    lane: ext
    label: protoc --descriptor_set_out
    step: 6
    col: 2
  - id: model
    lane: sk
    label: api.API (+ xref, validate)
    step: 7
    col: 3
  - id: annotate
    lane: sk
    label: <lang>.annotateModel
    step: 8
    col: 4
  - id: render
    lane: sk
    label: templates -> files
    step: 9
    col: 5
  - id: fmt
    lane: ext
    label: cargo fmt, dart format, swift-format
    step: 10
    col: 6
  - id: post
    lane: lib
    label: post-processing
    step: 10
    col: 6
edges:
  - yaml -> mc
  - mc -> parse
  - parse -> protoc | protobuf only
  - protoc -> model | descriptors
  - parse -> model
  - model -> annotate
  - annotate -> render
  - render -> fmt
  - fmt -> post
```

## Two generators behind one command
<!-- step runs="Developer machine or CI in the language repository" code="librarian" -->

The split is decided by one `switch` in `generateLibraries`. Each case owns
the concurrency strategy and the order of generate, format and post-process
steps for its language.

```excerpt
file: internal/librarian/generate.go
symbol: generateLibraries
end: "+20"
highlight: "switch cfg.Language"
caption: The dispatch point; what each case calls is where the two families diverge.
```

| | External generator | In-process (sidekick) |
|---|---|---|
| Languages | go, java, nodejs, php, python, ruby | rust, rust_prost, swift, dart |
| Who parses the API | `protoc` plus a plugin such as `protoc-gen-go_gapic` | `internal/sidekick/parser` (calls `protoc` only for descriptors) |
| Where the templates live | the upstream generator repository | `internal/sidekick/<lang>/templates/**/*.gotmpl`, embedded |
| Input formats | protobuf only | protobuf; OpenAPI and Discovery for some languages |
| What Librarian owns | command line, tool pins, post-processing | everything |

The rest of this page is about the right-hand column. For the left-hand
column, the per-language packages under `internal/librarian/<lang>` are the
whole story and [Changing what sidekick generates for one
language](modify-language-generation.html) contrasts the two at the end.

## What feeds a generation
<!-- step runs="Language repository checkout" code="librarian.yaml + pinned sources" -->

Sidekick reads up to six inputs for one library:

1. **API definition files.** `.proto` files from `googleapis` (or another
   configured root), or a Discovery document, or an OpenAPI v3 document.
2. **The service configuration YAML** (`type: google.api.Service`) next to the
   protos. It supplies the API title, description, the package name authority,
   mixin selectors, documentation URIs and `method_settings`.
3. **`sdk.yaml`**, embedded in the binary from `internal/serviceconfig`. It
   holds exceptions keyed by `googleapis` path: a different service config,
   per-language release level or transport, and for Discovery and OpenAPI
   sources the path of the document.
4. **`librarian.yaml` per-library options.** Output directory, specification
   format, include and skip lists, documentation and pagination overrides,
   and a block of language options (`rust:`, `dart:`, `swift:`).
5. **Pinned source archives**, fetched and cached by commit as described in
   [the workflow walkthrough](workflow.html#step-6).
6. **The `protoc` binary**, pinned under `tools.protoc`, used only to turn
   `.proto` files into a `FileDescriptorSet`.

The library's `roots` list says which source archives are searched, in
order. The names are an enumeration in code, not free-form, which matters
when a new kind of input needs a new repository.

```excerpt
file: internal/sources/sources.go
symbol: SourceConfig.Root
end: "+19"
caption: The five source roots sidekick can resolve paths against; googleapis is the default.
```

## Sources on disk
<!-- step runs="Developer machine or CI" code="librarian" -->

`runGenerate` loads all configured sources in parallel before touching any
library, then resolves defaults and preview variants per library, cleans
output directories while respecting `keep`, and only then generates.

```excerpt
file: internal/librarian/generate.go
symbol: runGenerate
end: "+48"
caption: Sources, defaults, clean, generate. Everything after LoadSources is offline.
```

For the protobuf parser, a path such as `google/cloud/secretmanager/v1` is
resolved with `ResolveDir` against the active roots; the parser compiles the
`.proto` files found directly in that directory (or exactly the
`include_list`, if given). The Discovery parser resolves its document the same
way, which is why a Discovery library must list `discovery` in its `roots`.

## From a Library to a ModelConfig
<!-- step runs="Developer machine or CI" code="librarian (internal/librarian/<lang>)" -->

Each sidekick language has a small builder in `internal/librarian/<lang>` that
folds `librarian.yaml`, `sdk.yaml` and the sources into a
`parser.ModelConfig`. The Rust one shows the whole shape: the specification
format picks the source path, `serviceconfig.Find` consults `sdk.yaml` and
then scans the API directory for a service config, and the release level is
handed to the codec.

```excerpt
file: internal/librarian/rust/codec.go
symbol: libraryToModelConfig
end: "+42"
highlight: "switch specFormat"
caption: specification_format decides whether the source is a Discovery document, an OpenAPI document or a proto directory.
```

```excerpt
file: internal/serviceconfig/serviceconfig.go
symbol: FindAPI
end: "+18"
caption: sdk.yaml entries match on the API path, the OpenAPI path or the Discovery path.
```

The three builders differ in how they pass language options on: Rust and
Dart flatten typed config structs into a `map[string]string` that the codec
parses again; Swift hands the typed `*config.Library` straight to its codec.
Both styles end in the same place, but the typed one needs fewer edits per
option; see [Changing what sidekick generates for one
language](modify-language-generation.html#step-2).

## Parsing: one switch, one model
<!-- step runs="Developer machine or CI" code="librarian (internal/sidekick/parser)" -->

`parser.CreateModel` is the only entry point. It switches on the
specification format, then runs the same format-independent passes over
whatever the parser returned: pagination detection, recursive-field
labelling, cross-referencing, resource identification, include and skip
lists, documentation patches, validation and name overrides.

```excerpt
file: internal/sidekick/parser/parser.go
symbol: CreateModel
end: "+46"
highlight: "switch cfg.SpecificationFormat"
caption: Every parser must produce a model that survives the passes below the switch.
```

Two of those passes define the contract a parser has to meet. Cross-reference
turns string IDs such as `.google.cloud.secretmanager.v1.Secret` into
pointers and fails on anything dangling, and validation insists that all
top-level elements share one package, with the mixin packages as the only
exception.

```excerpt
file: internal/sidekick/api/validate.go
symbol: Validate
end: "+16"
```

Pagination is worth calling out because it is not a protobuf feature: it is
detected from field names (`page_token`, `page_size` or `maxResults`,
`next_page_token`) and therefore works for every input format.

```excerpt
file: internal/sidekick/api/pagination.go
start: "const ("
end: "+6"
```

## The protobuf parser and protoc
<!-- step runs="Developer machine or CI" code="librarian + protoc" -->

The protobuf parser is the complete one and the only one that spawns a
process. It runs the pinned `protoc` once per library with every active root
as a `--proto_path`, asking for a descriptor set that includes imports,
source comments and options.

```excerpt
file: internal/sidekick/parser/protobuf.go
symbol: runProtoc
end: "+42"
caption: The only external process in parsing; everything after this reads the descriptor set in memory.
```

The descriptor set is then walked into `api.API`. Per method, the parser
reads the `google.api.*` annotations that the generated clients depend on:
HTTP bindings and additional bindings, routing headers, method signatures,
long-running operation info, streaming flags and the API version header.

```excerpt
file: internal/sidekick/parser/protobuf.go
symbol: processMethod
end: "+33"
caption: Each annotation becomes a plain field on api.Method; templates never see descriptors.
```

Mixins (`google.cloud.location`, `google.iam.v1`, `google.longrunning`) are
added from descriptors compiled into the binary when the service config lists
them, and the Operations mixin is forced in whenever a method returns
`google.longrunning.Operation`.

The OpenAPI and Discovery parsers synthesize what their formats lack: a
request message per operation, a single service for OpenAPI or one service
per Discovery resource, and HTTP bindings from path templates. What they
cannot synthesize, such as long-running operations from OpenAPI or routing
headers from either, simply stays empty in the model; [Supporting a different
input format](extend-new-input.html) has the full capability matrix.

## The model and the Codec fields
<!-- step runs="In memory" code="librarian (internal/sidekick/api)" -->

`api.API` is deliberately boring: names, documentation, services, methods,
messages, enums, resources, and symbol tables for lookups. It contains no
language concepts. The hook for languages is a single `Codec any` field on
the API and on every service, method, message, field, enum and enum value.

```excerpt
file: internal/sidekick/api/api.go
symbol: API
end: "Codec any"
caption: The language-neutral root of the model; Codec is where a language attaches its own struct.
```

There is no Go interface that a language must implement. The shared
`internal/sidekick/language` package offers helpers such as `PathParams`,
`QueryParams` and `HasNestedTypes`, and the template engine, but the
contract between the engine and a language is purely by convention: fill in
the `Codec` fields, then render templates that read them.

## The annotate pass
<!-- step runs="In memory" code="librarian (internal/sidekick/<lang>)" -->

Each language has an `annotateModel` driver that walks the model in a fixed
order and attaches its annotation structs. Everything a template will need,
from the Rust type of a field to the Swift name of an enum case or the
documentation already formatted as comments, is computed here. Templates
are not allowed to compute, so the annotation pass is where the language
logic lives.

```excerpt
file: internal/sidekick/rust/annotate_model.go
symbol: annotateModel
end: "+30"
caption: Rust annotates enums, then messages, then external types, then services and methods.
```

```excerpt
file: internal/sidekick/swift/annotate_model.go
symbol: codec.annotateModel
end: "+30"
caption: "Swift is the architectural reference in internal/sidekick/AGENTS.md: one file per node type, slim driver."
```

The codec object the driver carries is where `librarian.yaml` options land.
For Rust that is a string switch over the option map built in
`internal/librarian/rust`, which also shows how the specification format
changes defaults such as the `$alt` system parameter.

```excerpt
file: internal/sidekick/rust/codec.go
symbol: newCodec
end: "+30"
```

## Templates: Go text/template with the sharp edges removed
<!-- step runs="In memory, then files under the output directory" code="librarian (internal/sidekick/<lang>/templates)" -->

Templates are `*.gotmpl` files embedded with `//go:embed all:templates` and
parsed once per language package. The engine is standard `text/template`
with `missingkey=error` and exactly two helper functions, `include` and
`indent`.

```excerpt
file: internal/sidekick/language/gotemplate.go
symbol: ParseTemplatesDir
end: "+37"
caption: Template names are their paths without .gotmpl, so partials are included by path.
```

A validation step rejects templates that try to think: variable assignment,
`break` and `continue`, comparison and arithmetic built-ins and method calls
with arguments are all banned. This is what forces logic into the annotate
pass and keeps templates readable as the target language with holes in it.

```excerpt
file: internal/sidekick/language/gotemplate.go
start: "bannedIdentifiers = map[string]bool{"
end: "+17"
```

Which templates become files follows a naming rule rather than a manifest: a
template whose file name contains two or more dots (`lib.rs.gotmpl`,
`pubspec.yaml.gotmpl`) is written to the output directory with the last
extension removed; a single-dot name (`message.gotmpl`) is a partial.

```excerpt
file: internal/sidekick/language/walk_templates_dir.go
symbol: WalkTemplatesDir
end: "+24"
```

Languages differ in how they drive rendering. Rust picks a template root per
crate (`crate`, `nosvc`, or a `template-override`) and renders it whole; Dart
walks one tree and renames a few outputs; Swift renders one file per
message, enum and service and then walks a `package` tree.

```excerpt
file: internal/sidekick/rust/generate.go
symbol: codec.generatedFiles
end: "+12"
caption: Rust template roots; template-override lets a crate use a completely different set.
```

## Formatting and post-processing
<!-- step runs="Developer machine or CI" code="librarian + ecosystem formatters" -->

Generated files are not pretty; the ecosystem's own formatter fixes that.
Rust runs `taplo fmt` on `Cargo.toml` and `cargo fmt` on the crate, Dart runs
`dart format` (once inside the codec and once from `internal/librarian/dart`),
Swift runs `swift-format`. These tools come from the language repository's
toolchain and `tools:` pins, not from Librarian.

```excerpt
file: internal/librarian/rust/generate.go
start: "if err := command.Run(ctx, \"taplo\", \"fmt\","
end: "+6"
```

After formatting, `internal/librarian/<lang>` does whatever the ecosystem
needs beyond generated source. Rust scaffolds new crates, generates a
prost-based hybrid crate for gRPC root types, writes `.repo-metadata.json`
and a documentation index, and runs `cargo update --workspace` once at the
end. Swift dispatches on a module type that can also mean "run `protoc
--swift_out`" or "copy a handwritten storage module". Dart has nothing to do.

```excerpt
file: internal/librarian/swift/generate_module.go
symbol: generateModule
end: "+40"
caption: Not every Swift module goes through templates; module_type picks the strategy.
```

## What lives where
<!-- step runs="Everywhere" code="summary" -->

| Concern | Location | Shared across languages? |
|---|---|---|
| Specification format constants | `internal/config/specs.go` | yes |
| Source fetching, caching, roots | `internal/librarian/source.go`, `internal/sources` | yes |
| Service config and `sdk.yaml` | `internal/serviceconfig` | yes |
| Parsers | `internal/sidekick/parser`, `parser/discovery`, `parser/httprule` | yes |
| Model and post-parse passes | `internal/sidekick/api` | yes |
| Template engine | `internal/sidekick/language` | yes |
| `Library` to `ModelConfig`, format, post-process | `internal/librarian/<lang>` | per language |
| Codec, annotations, templates | `internal/sidekick/<lang>` | per language |
| Formatters, `protoc` | language repository toolchain and `tools:` pins | external |
| Which APIs, which options | `librarian.yaml` in the language repository | per repository |

Nothing in the generation path executes code from the language repository.
That is the main practical difference from the external-generator languages,
where Node.js runs a per-package `librarian.js`, PHP runs `owlbot.py`, Ruby
runs `toys` tasks and Java builds its generator from a directory in the
repository.

## How changes are verified
<!-- step runs="GitHub Actions in this repository" code="librarian CI" -->

Sidekick has no golden-directory diff; expected output lives as strings in
`_test.go` files next to each annotation, and parser tests need `protoc` on
`PATH`. The `Sidekick (Rust, Swift)` workflow runs those tests with a coverage
gate, and on pushes to `main` checks out `google-cloud-rust`, runs `librarian
install` and `librarian generate --all` against it, and compiles one crate.

```excerpt
file: .github/workflows/sidekick.yaml
start: "- name: Run tests and check coverage"
end: "+2"
```

```excerpt
file: .github/workflows/sidekick.yaml
start: "- name: Run librarian generate"
end: "+5"
caption: The integration job regenerates a real repository, so template changes show up as a diff there.
```

Dart has its own `dart.yaml` with unit tests only. Note that the repository's
generic coverage job excludes `internal/sidekick` and `internal/librarian/<lang>`,
so each sidekick language relies on its dedicated workflow.

## Where to go next
<!-- step runs="Reading" code="walkthroughs" -->

Three follow-up walkthroughs answer the questions this one raises:

- [Adding a new language to sidekick](extend-new-language.html): the exact
  files to create and the registration points to touch, using Dart as the
  template to copy.
- [Supporting a different input format](extend-new-input.html): how the
  format switch works, what each parser can and cannot populate, and which
  Google Cloud conventions are hard-coded along the way.
- [Changing what sidekick generates for one language](modify-language-generation.html):
  the ladder of knobs from a `librarian.yaml` option to a template edit, and
  how to verify a change.
