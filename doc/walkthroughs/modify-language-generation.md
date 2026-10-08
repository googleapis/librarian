---
title: Changing what sidekick generates for one language
summary: The ladder of places where generated Rust, Dart or Swift output can be changed, ordered from a one-line librarian.yaml option to a template or post-processing edit, with the verification loop for each and the contrast with languages whose generator lives upstream.
audience: For engineers changing generated output for Rust, Dart or Swift
order: 5
nav: Change one language
---

A request like "the Rust client should not emit this method" or "Swift doc
comments should link to the REST reference" can be satisfied at several
levels. The levels differ in how many libraries they affect, who can make
the change (a consumer editing `librarian.yaml` versus a Librarian release),
and how much testing they need. This page lists them from cheapest to most
invasive, shows where each lives for the three sidekick languages, and ends
with how the same request is handled for languages whose generator is an
external tool.

The generation pipeline itself is explained in
[Inside generate: how sidekick produces code](generate-and-sidekick.html);
adding a language is covered in
[Adding a new language to sidekick](extend-new-language.html).

## The ladder
<!-- step runs="Librarian repository or language repository" code="librarian" -->

Prefer the lowest rung that solves the problem. Each rung below changes
strictly more than the one above it.

| Rung | Where | Blast radius | Who ships it |
|---|---|---|---|
| 1 | A `librarian.yaml` option, read through `internal/config` | One library (or all, via `default`) | The language repository, no Librarian release |
| 2 | `internal/serviceconfig/sdk.yaml` | One API, every language that reads the field | Librarian release |
| 3 | `annotate_*.go` in `internal/sidekick/<lang>` | Every library of one language | Librarian release |
| 4 | `templates/**/*.gotmpl` in `internal/sidekick/<lang>` | Every library of one language | Librarian release |
| 5 | The formatter invocation in `internal/librarian/<lang>` | Every library of one language; output style only | Librarian release |
| 6 | Post-processing in `internal/librarian/<lang>` | Every library of one language; files around the generated code | Librarian release |

Changes to `internal/sidekick/parser` or `internal/sidekick/api` are not on
the ladder because they change the model every language sees;
`internal/sidekick/AGENTS.md` calls this the cross-language blast radius and
asks for tests in each language package when you touch them.

```flow
title: Where each rung acts in the pipeline
lanes:
  - id: lang
    label: language repository
    kind: lang
  - id: lib
    label: librarian
    kind: lib
  - id: ext
    label: external tools
    kind: ext
nodes:
  - id: yaml
    lane: lang
    label: librarian.yaml option
    step: 2
  - id: sdk
    lane: lib
    label: sdk.yaml
    step: 3
  - id: annotate
    lane: lib
    label: annotate_*.go
    step: 4
  - id: tmpl
    lane: lib
    label: templates
    step: 5
  - id: fmt
    lane: ext
    label: formatter
    step: 6
  - id: post
    lane: lib
    label: post-processing
    step: 7
edges:
  - yaml -> sdk | ModelConfig
  - sdk -> annotate | *api.API
  - annotate -> tmpl | Codec fields
  - tmpl -> fmt | files
  - fmt -> post
```

## Rung 1: a `librarian.yaml` option
<!-- step runs="Language repository (uses the option); Librarian repository (adds it)" code="librarian" -->

If the knob already exists, the change is a line in the consumer's
`librarian.yaml` and no Librarian release. The knobs are typed structs in
`internal/config`: `RustDefault`, `RustCrate` and `RustModule` in
`language.go`, `DartPackage` in `language.go`, `SwiftDefault`,
`SwiftPackage`, `SwiftModule` and `SwiftDependency` in `swift.go`. All of
them are documented in `doc/config-schema.md`.

How a typed field reaches the codec differs by language, and that decides
how many edits a *new* knob costs. Rust and Dart flatten the struct into a
`map[string]string` in `internal/librarian/<lang>/codec.go`:

```excerpt
file: internal/librarian/rust/codec.go
symbol: buildCodec
end: "+22"
caption: Every Rust option is a string key. The consumer never sees these names; they are an internal wire format.
```

and the codec parses the map back with a string switch:

```excerpt
file: internal/sidekick/rust/codec.go
symbol: newCodec
end: "+29"
caption: Defaults depend on the specification format; then each key sets a codec field. Unknown keys are an error.
```

Swift skips the map. The codec receives the `*config.Library` and the
`*config.SwiftModule` and reads the typed fields directly:

```excerpt
file: internal/sidekick/swift/codec.go
symbol: newCodec
end: "generationYear := library.CopyrightYear"
caption: The codec signature is the config contract.
```

```excerpt
file: internal/sidekick/swift/codec.go
start: "swiftCfg := library.Swift"
end: "+13"
caption: No string keys, no parsing; the Go compiler checks the field names.
```

So adding a knob is three edits for Rust or Dart (the struct field, the
`buildCodec` line, the `newCodec` or `annotateModel` case) and two for Swift
(the struct field and `newCodec`). In every case also update `fill<Lang>`
and `merge<Lang>` in `internal/librarian/library.go` so the field is
defaulted and survives into preview libraries, and run `go generate
./internal/config` to refresh `doc/config-schema.md`.

Some options never reach the codec. `included_ids`, `skipped_ids`,
`documentation_overrides`, `pagination_overrides` and the Discovery
`pollers` are copied into `parser.ModelConfig` and applied by the shared
post-parse passes, which makes them the right place for "drop this method"
or "fix this comment" requests that would otherwise become template
special cases.

## Rung 2: `sdk.yaml`
<!-- step runs="Librarian repository" code="librarian" -->

`internal/serviceconfig/sdk.yaml` is embedded in the binary and consulted
for every API path by `serviceconfig.Find`. It is the place for facts about
an API that are not in its service configuration or that one language needs
to override: `title`, `description`, `service_config` (when the YAML is not
where the directory scan expects it), `release_level.<lang>`,
`transports.<lang>`, and the `discovery` or `open_api` document that stands
in for the protos.

```excerpt
file: internal/serviceconfig/serviceconfig.go
symbol: FindAPI
end: "+17"
caption: Lookup matches the googleapis path, or the OpenAPI or Discovery document path, so one entry serves all three input formats.
```

Per-language values fall back to an `all` key and then to a derivation:

```excerpt
file: internal/serviceconfig/api.go
symbol: API.ReleaseLevel
end: "+18"
caption: Rust passes this into the codec as release-level, which changes README and crate-level docs wording.
```

What each sidekick language actually reads from an entry at this commit:
Rust uses `title`, `description`, `service_config`, `release_level`,
`discovery` and `open_api`; Dart uses `title`, `description` and
`service_config`; Swift uses `service_config` and matches Discovery sources
by API path (its `libraryToModelConfig` does not copy `title` or
`description` into the model override). `doc/sdk-yaml-principles.md` sets
the ground rules: the file describes APIs, not libraries, and must not grow
into a second `librarian.yaml`.

## Rung 3: annotations
<!-- step runs="Librarian repository" code="librarian" -->

If the fix needs a computed value, a new name, a different filter, a new
doc-comment transformation, it belongs in the annotate pass, because the
template engine cannot compute (see rung 4). The annotation structs are
what templates see, so "what does the template need to know?" is the design
question.

Where each language keeps them:

| Language | Driver | Node annotations | Helpers |
|---|---|---|---|
| Rust | `internal/sidekick/rust/annotate_model.go` | `annotate_{service,method,message,field,oneof,enum,enum_value}.go` (eight files with the driver) | naming, doc comments and rustdoc links in `codec.go` |
| Dart | `internal/sidekick/dart/annotate.go` | all in the same file | `dart.go` |
| Swift | `internal/sidekick/swift/annotate_model.go` | `annotate_{service,method,message,field,oneof,enum,enum_value,lro_any,sample_info}.go` (ten files) | about fifteen single-purpose files such as `names.go`, `doc_link.go`, `format_path.go` |

```excerpt
file: internal/sidekick/rust/annotate_model.go
symbol: annotateModel
end: "+27"
caption: Order matters. Enums and messages first, because service and method annotations look up their annotated types.
```

Each annotation file has a matching `_test.go`. The test shape from
`AGENTS.md`: build a model with `api.NewTest*`, run `annotateModel`, assert
the annotation struct with `cmp.Diff`, keep error and gating cases in
separate `TestAnnotateXxx_Error` and `TestAnnotateXxx_Gating` functions.
When editing Dart, resist adding to `annotate.go`; new logic goes in a new
file with its own test, which is how the file stops growing.

## Rung 4: templates
<!-- step runs="Librarian repository" code="librarian" -->

Change a template when the output shape changes but the information is
already on the model: a different attribute, a reordered block, a new file
assembled from existing partials. Templates are Go `text/template` with
only `include` and `indent` available and the comparison, arithmetic and
indexing built-ins banned, which is why rung 3 exists.

How the three languages pick which templates to render:

- **Rust** has ten template roots (`crate`, `nosvc`, `common`,
  `http-client`, `grpc-client`, `grpc-mock`, `convert-prost`, `mod`,
  `storage`, `bigquery`). A library gets `crate` or `nosvc` depending on
  whether it has services, unless `template-override` names another root:

```excerpt
file: internal/sidekick/rust/generate.go
symbol: codec.generatedFiles
end: "+12"
caption: template-override is how a library opts into a different output shape without a new codec.
```

- **Dart** walks a single tree and renames a few outputs on the way:

```excerpt
file: internal/sidekick/dart/generate.go
symbol: generatedFiles
end: "+17"
caption: main.dart becomes <package>.dart, LICENSE.txt loses its extension, testing.dart is skipped without services.
```

- **Swift** emits per element: one file per message, enum, service and
  stub via `GenerateElement`, then walks `templates/package` for the
  package-level files (`internal/sidekick/swift/generate.go`).

Rules for the templates themselves are in
`doc/styleguide/go-template-style-guide.md`: align `{{- range }}` and
`{{- if }}` with the generated code, emit blank lines with `{{- "\n" }}`,
pipe included partials through `| indent N`, and start each file with a
`{{/* ... */}}` license comment. `parsedTemplates.Validate()` runs in each
package's tests and rejects banned constructs.

## Rung 5: the formatter
<!-- step runs="Language repository checkout" code="external tool" -->

Whitespace and layout are not the templates' job. Every language runs the
ecosystem formatter over the output, and that is where style changes go,
usually by changing the formatter version pinned in the consumer's
`librarian.yaml` `tools:` section rather than Librarian itself.

| Language | Formatter | Where it is invoked |
|---|---|---|
| Rust | `taplo fmt Cargo.toml`, then `cargo --frozen fmt -p <crate>` | `internal/librarian/rust/generate.go` `Format` |
| Dart | `dart format <output>`, twice | inside the codec unless `skip-format`, then again from `internal/librarian/dart/generate.go` |
| Swift | `swift-format format --in-place --recursive <dirs>` | `internal/librarian/swift/generate.go` `Format`, with the `tools.swift` environment |

Dart is the odd one out because the codec formats its own output:

```excerpt
file: internal/sidekick/dart/generate.go
symbol: Generate
end: "+15"
caption: Tests pass skip-format to stay hermetic; librarian's Format runs dart format again afterwards.
```

```excerpt
file: internal/librarian/rust/generate.go
symbol: Format
end: "+8"
caption: --frozen lets formatting run in parallel across crates without fighting over the cargo lock.
```

A formatter change shows up as a diff across every generated library in
the consumer's next regeneration, which is normal and is why regeneration
pull requests are reviewed per language.

## Rung 6: post-processing in `internal/librarian/<lang>`
<!-- step runs="Language repository checkout" code="librarian" -->

Everything after the codec returns is plain Go in the glue package, and it
is the place for files that are *about* the library rather than generated
from the API: crate scaffolding, repository metadata, documentation
indexes, workspace bookkeeping.

Rust does the most:

```excerpt
file: internal/librarian/rust/generate.go
start: "exists, err := crateExists(library.Output)"
end: "+29"
caption: Scaffold a new crate, render, add the prost hybrid for gRPC root types, write .repo-metadata.json, validate new crates.
```

After all crates are generated, `generateLibraries` also writes the doc
index and runs `cargo update --workspace`.

Swift's post-processing is organized around module types. A package can
list modules in `librarian.yaml`, and each `module_type` selects a
different generator, including one that is not a template at all:

```excerpt
file: internal/librarian/swift/generate_module.go
symbol: generateModule
end: "case \"storage\":"
caption: swift-protobuf modules run protoc --swift_out; convert-swift modules render conversion code from a second model.
```

Dart has no post-processing beyond formatting; `pubspec.yaml` handling
lives in `bump` and `publish`.

The generic `postprocess:` block in `librarian.yaml` (`replace`,
`copy_file`, `remove_file`) is defined in `internal/config` but is not
referenced by the Rust, Dart or Swift glue packages read for this page; it
should be treated as unavailable for sidekick languages unless you verify
otherwise.

## The verification loop
<!-- step runs="Developer machine; GitHub Actions" code="librarian" -->

For any rung from 3 upward:

1. Unit tests for the two packages of the language:

   ```sh
   go test -short ./internal/sidekick/<lang>/... ./internal/librarian/<lang>/...
   ```

   Tests that parse real protos need `protoc` on `PATH` and skip through
   `requireProtoc(t)` otherwise. Expected output is inline raw strings in
   the `_test.go` files; there is no golden directory to regenerate, so a
   deliberate output change means editing the expectation next to the test
   that covers it.

2. The CI workflow for the language. Rust and Swift share
   `.github/workflows/sidekick.yaml`; Dart has `.github/workflows/dart.yaml`.
   Both install the toolchain the formatter needs and run the coverage tool
   over the language's packages.

```excerpt
file: .github/workflows/sidekick.yaml
start: "- name: Run tests and check coverage"
end: "+1"
caption: Rust and Swift unit tests. The same workflow's integration job regenerates the Rust language repository on pushes to main.
```

3. Regenerate in the language repository and read the diff. Point the
   consumer's `librarian.yaml` at your Librarian build (`go run` from a
   local checkout, or `version:` at a pre-release tag), run
   `librarian generate <library>` for one representative library and then
   `--all`, and inspect `git diff`. This is the only check that shows the
   change against real APIs rather than fixtures; the Rust integration job
   automates it for `main`, Dart and Swift have no equivalent job in this
   repository.

## Contrast: languages generated by an external tool
<!-- step runs="Language repository checkout" code="external tool" -->

For Go, Java, Node.js, PHP, Python and Ruby the generator is a `protoc`
plugin or CLI maintained in its own repository. Librarian owns three things
for them: the command line, the version pins, and the post-processing. It
owns none of the templates.

```excerpt
file: internal/librarian/golang/generate.go
symbol: buildGAPICOpts
end: "+25"
caption: sdk.yaml and the service config become plugin options; the plugin decides what code those options produce.
```

The ladder therefore collapses to two rungs:

- **Options and metadata** (which files, which flags, `.repo-metadata.json`,
  version files, `go mod tidy`): change `internal/librarian/<lang>`, same as
  rung 6 above.
- **Generated code**: change the upstream generator, release it, and bump
  its pin under `tools:` in the language repository's `librarian.yaml`,
  which `librarian install` reads. No Librarian release is required for
  that second step.

`sdk.yaml` (rung 2) still applies, because `serviceconfig.Find` is shared;
`skip_rest_numeric_enums`, `transports` and `release_level` all flow into
the plugin options shown above.

## Recap: knobs per language
<!-- step runs="Everywhere" code="summary" -->

| Knob layer | Rust | Dart | Swift |
|---|---|---|---|
| `librarian.yaml` struct | `RustDefault`, `RustCrate`, `RustModule` | `DartPackage` | `SwiftDefault`, `SwiftPackage`, `SwiftModule`, `SwiftDependency` |
| Option transport to codec | string map (`buildCodec` to `newCodec` switch) | string map (`buildCodec` to `annotateModel`) | typed structs passed directly |
| Model overrides | included/skipped IDs, documentation and pagination overrides, module-name overrides, resource-name heuristic, Discovery pollers | name and title override, skipped IDs, include list | included/skipped IDs, include list, Discovery pollers |
| `sdk.yaml` fields read | `release_level`, `title`, `description`, `discovery`, `open_api`, `service_config` | `title`, `description`, `service_config` | `service_config`; Discovery matched by path |
| Annotation files | eight `annotate_*.go` plus helpers in `codec.go` | one `annotate.go` | ten `annotate_*.go` plus about fifteen helper files |
| Template roots | ten roots, `template-override` | one tree with renames | nine directories, per-element emission |
| Formatter | `taplo` and `cargo fmt` | `dart format`, twice | `swift-format` |
| Post-processing | crate scaffold, prost hybrid, repo metadata, validate, `cargo update`, doc index | none | module-type dispatch, `protoc --swift_out`, repo metadata, doc index |
| CI | `sidekick.yaml` (unit, plus Rust integration) | `dart.yaml` (unit) | `sidekick.yaml` (unit) |
| Specification formats | protobuf, discovery, openapi (plus `none` for storage modules) | protobuf | protobuf, discovery |

Rust also has a `rust_prost` codec for modules with `template: prost` or
`template: tonic`; it renders a temporary crate whose `build.rs` runs
`prost-build`, so its knobs are the module fields and its "templates" are
that crate.
