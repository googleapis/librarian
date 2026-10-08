---
title: Supporting a different input format
summary: How sidekick chooses a parser, what the protobuf, OpenAPI and Discovery parsers can and cannot populate today, what a new parser must produce, and which Google Cloud conventions you would hit when generating a non-GAPIC SDK from such inputs.
audience: For engineers considering OpenAPI, Discovery or another specification as an input to sidekick
order: 4
nav: New input format
---

Sidekick's model, `api.API`, is independent of where the API definition came
from, and three parsers already feed it: protobuf descriptors, OpenAPI v3
documents and Discovery documents. That makes "support a new input" sound
like "write a parser". The parser is the small part. This walkthrough shows
how the format is selected, what each parser actually produces, the contract
a new one has to meet, and then the places downstream that quietly assume a
Google Cloud API described by protos in `googleapis`.

Read [Inside generate](generate-and-sidekick.html) first if the pipeline
stages are unfamiliar; this page starts at the format switch.

## Where the format is chosen
<!-- step runs="librarian.yaml, then internal/librarian/<lang>, then the parser" code="librarian" -->

A library's `specification_format` is one of four strings. The default is
`protobuf`; `none` exists for one Rust crate that only re-exports others.

```excerpt
file: internal/config/specs.go
start: "const ("
end: "+19"
```

The value travels through three hands. The per-language builder in
`internal/librarian/<lang>` copies it into `parser.ModelConfig` and uses it
to pick the source path (Rust reads the Discovery or OpenAPI path from
`sdk.yaml`; Swift expects the API path to *be* the Discovery file; Dart
rejects anything but protobuf). `parser.CreateModel` switches on it to pick
the parser. Finally the codecs read it again to adjust defaults, because a
Discovery-described service encodes enums and bytes differently on the wire.

```excerpt
file: internal/sidekick/swift/codec.go
start: "responseEncoding := defaultResponseEncoding"
end: "+16"
caption: Codec-side branches on the format; a new format may need the same treatment in every language.
```

So adding a format means touching, at minimum, the constants, each
builder's path resolution, the parser switch and possibly each codec.

## What each parser populates
<!-- step runs="In memory" code="librarian (internal/sidekick/parser)" -->

The matrix below is the heart of the assessment. A blank cell is not a bug;
it is a field of `api.API` that the format has no information for, and that
templates will treat as absent.

| Capability (model field) | protobuf | OpenAPI v3 | Discovery |
|---|---|---|---|
| Messages, enums, nested types, maps | full | `components.schemas`; no oneofs; `allOf` partial | `schemas` (objects only); inline objects; enums as strings |
| Services | from `service` blocks plus mixins | **one synthetic service** | one per `resources` node with methods |
| HTTP bindings (`PathInfo`) | `google.api.http` plus `additional_bindings` | from `paths` and parameters | `servicePath` plus `path` and parameters |
| Long-running operations | `OperationInfo` plus Operations mixin | no | only through `librarian.yaml` `discovery.pollers` |
| Mixins (Locations, IAM, Operations) | yes | no | no |
| Pagination | name-based, shared pass | same | same |
| Routing headers | `google.api.routing` | no | no |
| Field behavior | full `field_behavior` | `REQUIRED` only | required flag only |
| Auto-populated request IDs | `field_info` plus service config | `format: uuid` plus service config | service config only |
| Resource names and references | `google.api.resource*` | no | no |
| Method signatures | `google.api.method_signature` | no | `parameterOrder` plus body |
| Streaming | yes | no | no |
| API version header | `google.api.api_version` | no | `apiVersion` |
| OAuth scopes | not modeled | not modeled | parsed, then dropped |
| Languages that accept it | rust, rust_prost, dart, swift | rust | rust, swift |

Pagination deserves a note because it is the one "GAPIC feature" that works
everywhere: detection is by field name, not by annotation.

```excerpt
file: internal/sidekick/parser/protobuf_annotations.go
symbol: parseOperationInfo
end: "+12"
caption: LRO detection reads a protobuf extension; neither OpenAPI nor Discovery has an equivalent.
```

## OpenAPI today
<!-- step runs="In memory" code="librarian (internal/sidekick/parser/openapi.go)" -->

The OpenAPI parser is real but narrow. It reads the document directly from
the path it is given (not through the source roots), turns every component
schema into a message, and turns every operation into a method on a single
synthetic service with a synthetic `<OperationId>Request` message.

```excerpt
file: internal/sidekick/parser/openapi.go
symbol: makeMethods
end: "+30"
caption: Operations become methods on one service; path and query parameters become request fields.
```

Its constraints reflect the one document it was written against: responses
must be declared under `responses.default` as an `application/json` `$ref`,
request bodies must `$ref` a component schema, and `body` is a reserved
parameter name. Only Rust wires it up, and the only `open_api:` entry in
`sdk.yaml` points at a test fixture inside this repository.

```excerpt
file: internal/serviceconfig/sdk.yaml
start: "- path: google/cloud/secretmanager/v1"
end: "+2"
caption: The sole OpenAPI source known to sdk.yaml is a fixture kept to validate the parser.
```

No CI job generates a library from OpenAPI end to end; coverage comes from
`internal/sidekick/parser/openapi_test.go` and a Rust generation test.

## Discovery today
<!-- step runs="In memory" code="librarian (internal/sidekick/parser/discovery)" -->

The Discovery parser is more complete, because Compute-style APIs are
published only this way. Each `resources` node with methods becomes a
service, schemas become messages, methods get synthetic requests, HTTP
bindings from URI templates, method signatures from `parameterOrder`, and
the API version header. Media upload methods are rejected outright.

```excerpt
file: internal/sidekick/parser/discovery/methods.go
start: "// Methods without path parameters don't get an overload."
end: "+12"
caption: Discovery has an ordering hint that protobuf expresses with method_signature.
```

Discovery documents say nothing about long-running operations, so sidekick
takes that from `librarian.yaml`: a `discovery` block names the operation
type and the poller method per service, and the parser synthesizes the
annotations the LRO templates expect.

```excerpt
file: internal/sidekick/parser/discovery/lro.go
symbol: lroAnnotations
end: "+25"
```

Rust and Swift accept Discovery input; Dart does not. The document is
resolved through the source roots, so the library must list `discovery` in
`roots` alongside `googleapis` (the service config YAML still comes from
`googleapis`). `sdk.yaml` carries the two production entries, `compute` and
`dns`.

## What a new parser must produce
<!-- step runs="In memory" code="librarian (internal/sidekick/api)" -->

The contract is implicit in the passes that run after the switch and in the
templates. Concretely, a `Parse<Format>(cfg *ModelConfig) (*api.API, error)`
must:

1. Set `Name`, `Title`, `Description` and `PackageName`, and register every
   element in the symbol tables (`AddMessage`, `AddEnum`, `AddService`,
   `AddMethod`) **and** in the `Messages`, `Enums`, `Services` slices.
2. Use dotted, fully-qualified IDs with a leading dot: `.pkg.Msg`,
   `.pkg.Msg.field`, `.pkg.Service.Method`. Cross-referencing looks IDs up
   verbatim and fails on anything missing.
3. Give every method a request *message*, synthesizing one if the format has
   none, and an output message that exists (`.google.protobuf.Empty` after
   `LoadWellKnownTypes()`).
4. Provide `PathInfo` with at least one binding if an HTTP client should be
   generated; Rust silently drops methods without bindings.
5. Keep all top-level elements in one package, or extend the exception list
   in `api.Validate`.
6. Name only top-level request fields in `Method.Signatures`.

```excerpt
file: internal/sidekick/api/xref.go
symbol: CrossReference
end: "+40"
caption: The IDs a parser must get right; this pass turns them into pointers or fails.
```

The Discovery method builder is the best template for synthesizing a request
message and bindings from a non-proto source.

```excerpt
file: internal/sidekick/parser/discovery/methods.go
symbol: makeMethod
end: "+30"
```

## Registering the format
<!-- step runs="Code in this repository" code="librarian" -->

With a parser written, the registration list is short but spread out:

| Where | What |
|---|---|
| `internal/config/specs.go` | a new `Spec<Format>` constant and the doc comment on `specification_format` |
| `internal/sidekick/parser/parser.go` | a `case` in `CreateModel`; call `loadServiceConfig` and `updateAutoPopulatedFields` like the others |
| `internal/librarian/rust/codec.go`, `swift/generate.go`, `dart/codec.go` | how to find the document for the format (today Rust reads `sdk.yaml`, Swift the API path) |
| `internal/serviceconfig/api.go` and `sdk.yaml` | a field such as `open_api:` if documents live outside the API directory |
| `internal/sources/sources.go`, `internal/librarian/source.go`, `internal/config` | a new source root if documents come from a new repository |
| `internal/sidekick/<lang>/codec.go` | wire-format defaults per language, as Swift does for Discovery |

```excerpt
file: internal/librarian/source.go
symbol: LoadSources
end: "+20"
caption: Sources are fetched by name; a new input repository needs a name here and in config.Sources.
```

## What assumes Google Cloud
<!-- step runs="Across the pipeline" code="librarian" -->

Generating a GAPIC-style client for a Cloud API from a new format is
bounded work. Generating a *different kind* of SDK, or an SDK for an API that
is not published in `googleapis`, meets assumptions in every stage:

1. **The service config YAML is the naming authority.** Package name,
   service name and title come from `apis[0].name` in a
   `type: google.api.Service` file, and `Find` only looks for it under
   `<googleapis>/<api path>`. Without one, OpenAPI and Discovery produce an
   empty package name and a service called `Service`.
2. **`sdk.yaml` is embedded and keyed by `googleapis` path.** Discovery and
   OpenAPI sources are only discoverable through it (Rust) or by naming the
   document as the API path (Swift). Release levels are inferred from
   `v1alpha` and `v1beta` path segments.
3. **Source roots are an enumeration** of five Google repositories, and
   `googleapis` is mandatory.
4. **No authentication model.** OAuth scopes are parsed from Discovery and
   dropped; `DefaultHost` is the only endpoint datum; Rust and Swift append
   the `$alt` system parameter; the mixin package list and the single-package
   validation exceptions are Google packages.
5. **Package naming** derives `google-cloud-<x>`, `google_cloud_<x>` and the
   Swift equivalents from the protobuf package.
6. **The templates emit gax-based clients** with retry, tracing, LRO and
   pagination helpers, depending on `google-cloud-gax`, `-wkt` and `-auth`
   crates and their Dart and Swift counterparts.
7. **Repository metadata** such as `.repo-metadata.json`, the documentation
   index and README quickstart heuristics assume Cloud documentation URLs.

```excerpt
file: internal/serviceconfig/serviceconfig.go
symbol: findServiceConfig
end: "+20"
caption: The naming authority is found by scanning the API directory in googleapis.
```

## A realistic path for a Discovery-described API
<!-- step runs="Planning" code="assessment" -->

Putting the pieces together, the shortest path to *some* generated client
for a Discovery-described API that is not in `googleapis` today is:

1. Write a small `type: google.api.Service` YAML by hand for the name,
   title and `apis[].name`, and place it where `Find` will look, or extend
   `Find` to accept an explicit path.
2. Add the document to `sdk.yaml` (`discovery:`) or, for Swift, make the
   library's API path the document path, and set `roots: [discovery,
   googleapis]`.
3. Accept Google-flavored output from the existing Rust or Swift templates,
   or point Rust at a new template root with `template-override` (Swift
   would need a new module type).

Beyond that, the effort is in order of size: a new template root and
annotations that do not assume gax, auth and LRO helpers; an authentication
and endpoint model in `api.API` and the codecs; a way to name packages and
services without a service config; and finally whatever the new format
itself lacks, such as streaming or routing. The parser is a week; the rest
is the project.

## Recap
<!-- step runs="Everywhere" code="summary" -->

- Format selection is a string in `librarian.yaml`, honored by the builders,
  the parser switch and the codecs.
- Protobuf is complete; OpenAPI is Rust-only and narrow; Discovery is Rust
  and Swift and relies on `librarian.yaml` for LROs.
- A new parser must satisfy the cross-reference and validation passes and
  synthesize request messages and bindings itself.
- The hard-coded parts are not in the parser: naming authority, `sdk.yaml`,
  source roots, authentication, package naming, gax templates and metadata
  all assume a Google Cloud API in `googleapis`.

See also [Adding a new language to sidekick](extend-new-language.html) for
the language side of the same engine.
