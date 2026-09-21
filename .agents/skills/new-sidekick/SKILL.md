---
name: new-sidekick
description: Scaffolds a new language generator in internal/sidekick/<lang> and internal/librarian/<lang> with a decomposed, testable architecture following internal/sidekick/GEMINI.md.
---

# New Sidekick Generator Scaffolding

Use this skill when onboarding or creating a new target language generator in
Librarian. It guides you to bootstrap the complete decomposed directory
structure, annotation models, templates, librarian runner, configuration types,
and test suites conforming strictly to
[`internal/sidekick/GEMINI.md`](/internal/sidekick/GEMINI.md).

## Architectural Principles to Uphold

Before creating any files, review the core invariants:

1. **One File Per Annotation Type:** Every AST node annotation (`model`,
   `service`, `method`, `message`, `field`, `oneof`, `enum`, `enum_value`) must
   have its own dedicated source file (`annotate_<node>.go`) paired 1:1 with a
   unit test file (`annotate_<node>_test.go`).
2. **Decomposed Coordinator:** `annotate_model.go` coordinates the annotation
   passes across all model elements. Never bury model traversal inside service
   or method files.
3. **Template-First Emission:** Mustache templates and partials own all syntax,
   code structure, and formatting. Never emit target-language syntax or
   multi-line doc comments from Go using `strings.Builder` or `fmt.Sprintf`.
4. **Mustache Boilerplate & Partials:** Every `.mustache` file must have an
   Apache 2.0 license comment (`{{! ... }}`) using the current year (see
   [`internal/sidekick/swift/templates/common/service.swift.mustache`](/internal/sidekick/swift/templates/common/service.swift.mustache)
   for reference).
5. **Emit Boilerplate from file-level templates:** The file-level templates
   should emit the copyright boilerplate, using the correct formatting for the
   file type.
   1. Use `license.HeaderBulk()` to populate the boilerplate.
   2. Use a partial for each file type, e.g. one for `.cc` files and one for
      `.md` files.
   3. Any tests should use the `license.HeaderBulk()` too.
   4. New languages should use a `CopyrightYear` configuration to preserve the
      original year when the code was generated. Look at Swift and Rust for
      examples.
6. **Config Immutability & Completeness:** Configuration structs in
   `internal/config/<lang>.go` must be pure data types. Implement complete field
   merging (`merge<Lang>`) and defaulting (`fill<Lang>`) in
   `internal/librarian/library.go`. Treat configuration as read-only.
7. **Hermetic Output:** Emitted file paths must remain strictly under `outdir`
   (no `..` relative path escapes).
8. **Idiomatic Tests:** Use `api.NewTest*` fluent builders for test models. Use
   unnamed table-driven slices `for _, test := range []struct { ... }{ ... }`
   and `cmp.Diff`. Never call test builders on production paths.
9. **Commit Scope:** Keep commits strictly scoped to a single directory. If a
   commit must cross directory boundaries during bootstrapping, ask the user
   for confirmation first.

---

## Scaffolding Workflow

Let `<lang>` be the lowercase name of the language (e.g., `cpp`, `kotlin`,
`ruby`) and `<Lang>` be the PascalCase name (e.g., `Cpp`, `Kotlin`, `Ruby`).

### Step 1: Configuration (`internal/config/<lang>.go`)

Create `internal/config/<lang>.go` defining the language configuration structs.
Include the standard project Apache 2.0 license header with the current year
(see [`internal/config/swift.go`](/internal/config/swift.go) for reference):

```go
package config

// <Lang>Default contains the default configuration shared by all <Lang> libraries.
type <Lang>Default struct {
	DefaultVersion string `yaml:"default_version,omitempty"`
}

// <Lang>Library contains <Lang>-specific configuration for a library.
type <Lang>Library struct {
	<Lang>Default `yaml:",inline"`
}
```

### Step 2: Sidekick Codec & Annotations (`internal/sidekick/<lang>/`)

Copy and adapt the living reference generator in
[`internal/sidekick/codec_sample/`](/internal/sidekick/codec_sample/) into
`internal/sidekick/<lang>/`. It contains the complete, tested, and decomposed
structure:

- `codec.go` & `codec_test.go` — Codec initialization (adapt `newCodec` to
  accept `*config.<Lang>Library`).
- `generate.go` & `generate_test.go` — Orchestration pipeline, embedded
  templates, and hermetic tests using `api.NewTestAPI`.
- `annotate_model.go` & `annotate_model_test.go` — Root coordinator iterating
  over messages, enums, and services.
- `annotate_service.go` & `annotate_service_test.go` — `serviceAnnotations` and
  method traversal.
- `annotate_method.go` & `annotate_method_test.go` — `methodAnnotations`.
- `annotate_message.go` & `annotate_message_test.go` — `messageAnnotations`,
  field traversal, and oneof traversal.
- `annotate_field.go` & `annotate_field_test.go` — `fieldAnnotations`.
- `annotate_oneof.go` & `annotate_oneof_test.go` — `oneOfAnnotations`.
- `annotate_enum.go` & `annotate_enum_test.go` — `enumAnnotations` and enum
  value traversal.
- `annotate_enum_value.go` & `annotate_enum_value_test.go` —
  `enumValueAnnotations`.
- `templates/partials/prologue.mustache` — Reusable output file license header.
- Target templates (e.g. `templates/readme/README.md.mustache`) — Rendering
  logic including `{{> /templates/partials/prologue}}`.

When copying from `internal/sidekick/codec_sample/`:
1. Change package declarations from `package codec_sample` to `package <lang>`.
2. Adapt `newCodec` and `Generate` to accept `*config.<Lang>Library`.
3. Keep all 1:1 test files and table-driven test conventions intact.

### Step 3: Librarian Integration (`internal/librarian/<lang>/`)

Create the librarian orchestration package under `internal/librarian/<lang>/`:

#### 1. `generate.go` & `generate_test.go`
- `generate.go`:
  ```go
  // Package <lang> orchestrates <Lang> client library generation in Librarian.
  package <lang>

  import (
  	"context"
  	"fmt"

  	"github.com/googleapis/librarian/internal/config"
  	sidekick "github.com/googleapis/librarian/internal/sidekick/<lang>"
  	"github.com/googleapis/librarian/internal/sidekick/api"
    "github.com/googleapis/librarian/internal/sidekick/parser"
    "github.com/googleapis/librarian/internal/sources"
  )

  // Generate runs the <Lang> generator pipeline.
  func Generate(ctx context.Context, cfg *config.Config, lib *config.Library, srcs *sources.Sources) error {
    if lib.<Lang> == nil {
      return fmt.Errorf("library %s is missing <lang> configuration", lib.Name)
    }
    modelConfig, err := libraryToModelConfig(library, library.APIs[0], sources, pc)
    if err != nil {
      return err
    }
    model, err := parser.CreateModel(modelConfig)
    if err != nil {
      return err
    }
  	return sidekick.Generate(ctx, model, lib.Output, lib.<Lang>)
  }
  ```
- `generate_test.go`: Unit tests for Librarian `<lang>.Generate`.

#### 2. `format.go` & `format_test.go`
- `format.go`: Runs the target language formatter (e.g. `clang-format`,
  `rustfmt`, `swift-format`) or performs no-op/validation. Must fail loudly if
  formatting fails on validly generated code.
  ```go
  func Format(ctx context.Context, library *config.Library) error {
    return nil
  }
  ```
- `format_test.go`: Unit tests for `Format`.

### Step 4: Hook into Librarian Core

Wire the new language into the central Librarian registry:

1. **`internal/config/language.go`:**
   Add `Language<Lang> = "<lang>"` to the language constants.
2. **`internal/config/config.go`:**
   - Add `<Lang> *<Lang>Library` to `type Library struct { ... }` with
     `yaml:"<lang>,omitempty"`.
   - Add `<Lang> *<Lang>Default` to `type Default struct { ... }` with
     `yaml:"<lang>,omitempty"`.
3. **`internal/librarian/library.go`:**
   - Implement `merge<Lang>(base, override *config.<Lang>Library)`
     `*config.<Lang>Library` ensuring **all fields** are merged without silent
     drops.
   - Implement `fill<Lang>(lib *config.Library, def *config.Default)` applying
     defaults.
   - Wire `merge<Lang>` into `resolvePreview` and `fill<Lang>` into
     `fillDefaults`.
4. **`internal/librarian/generate.go`:**
   Add dispatch case for `config.Language<Lang>` in `Generate(ctx, cfg, ...)`.

---

## Verification & Self-Review Checklist

After scaffolding, run these commands:

```bash
gofmt -s -w .
go tool goimports -w .
go tool golangci-lint run ./internal/sidekick/<lang>/... ./internal/librarian/<lang>/... ./internal/config/...
go test -short ./internal/sidekick/<lang>/... ./internal/librarian/<lang>/... ./internal/config/...
go test ./...
```

### Invariants Checklist:
- [ ] Every AST node type has its own `annotate_<node>.go` and
  `annotate_<node>_test.go`.
- [ ] `annotate_model.go` drives all model traversals; no buried loops in
  service annotators.
- [ ] Zero target-language syntax generation in Go source
  (`strings.Builder`/`fmt.Sprintf`).
- [ ] Every `.mustache` file has a license comment enclosed in `{{! ... }}`.
- [ ] Standalone templates include `{{> /templates/partials/prologue}}` for
  output file headers.
- [ ] `merge<Lang>` merges every field of `<Lang>Library`; no fields dropped.
- [ ] `fill<Lang>` applies defaults from `<Lang>Default`.
- [ ] All table-driven tests use `for _, test := range []struct { ... }{ ... }`
  and `cmp.Diff`.
- [ ] All tests use fluent `api.NewTest*` builders; no test builders called on
  production paths.
- [ ] Generated files are constrained strictly to `outdir`.
- [ ] Commits are scoped to single directories.
