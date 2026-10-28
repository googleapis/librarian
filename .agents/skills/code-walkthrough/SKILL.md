---
name: code-walkthrough
description: Produces a self-contained interactive HTML architecture walkthrough (dependency map, guided tour, command flows, guides) for this repository at the current commit. Use when asked to "walk me through the code", explain the architecture or a subsystem, or build a guide for modifying or extending a part of the codebase. The narrative is written fresh against the checkout; the facts (packages, imports, line counts) are computed.
---

# Code walkthrough

## What you produce

A self-contained HTML page that opens from `file://` with no server and no
network. It has an architecture map (every Go package in the module, grouped
into layers you assign, with import wires drawn from `go list`), a guided tour,
one flow per CLI command, deep-dive guides, a per-target table and findings. The
HTML is generated; never commit it.

Everything you write goes in one JSON file. Everything that is a fact about the
code is computed by the renderer and cannot be typed by hand:

| You author in JSON | Renderer computes |
|---|---|
| layers, each package's layer and (optionally) a better one-line description | package list, imports, non-test LOC, doc synopses, module path, Go version, repo, commit |
| tour steps, flows, guides, findings, languages table | validation of every cited path and package, links between walkthroughs |

## Where output goes

Every file goes in `.walkthrough/` at the module root, which is gitignored as a
whole. The renderer refuses any `-out` outside it. Each walkthrough is a pair
named after its scope:

- `.walkthrough/architecture.json` and `.walkthrough/architecture.html` for a
  whole-architecture request.
- `.walkthrough/<focus>.json` and `.walkthrough/<focus>.html` for a focused
  request, where `<focus>` is the subsystem in lowercase kebab case, such as
  `rust-bump` or `python-generate`.

Names use only letters, digits, `.`, `_` and `-`. When the user asks for more
on an existing walkthrough, edit its JSON and re-render it; do not start a new
pair. Every page has a "Walkthroughs" bar linking to the other pages in
`.walkthrough/`, and rendering one page updates that bar in the others.

## Workflow

1. **Get the skeleton.** Pick `<name>` as described in
   [Where output goes](#where-output-goes). The skeleton lists every package
   with its doc comment synopsis. Skip this step when `<name>.json` already
   exists; `-init` never overwrites it.

   ```sh
   go run .agents/skills/code-walkthrough/scripts/render.go -init -out .walkthrough/<name>.json
   ```

   `-facts` prints the same computed facts as JSON (`module`, `goVersion`,
   `sha`, `pkgs[]` with `imports` and `loc`). Use it for any claim about who
   imports what ("only X and Y import Z", "nothing imports W") instead of
   reading import blocks by hand.

2. **Read before you write.** Open the packages and files relevant to the
   request. Do not describe code you have not read in this session; do not
   reuse text from an earlier walkthrough. Package doc comments are usually the
   best one-line description; if one is missing or wrong, override `desc` and
   report it as a finding. This task is read-only: do not edit repository
   files.

3. **Decide the scope.** "Walk me through the code / the architecture" or any
   request that does not name one subsystem is a **whole-architecture**
   request and produces every section below. A request about one subsystem
   or one command sets `focus` to that subsystem's name and produces the
   tour (3–6 steps) plus the guide or flow it asked for; the renderer warns
   about missing sections unless `focus` is set.

4. **Edit `.walkthrough/<name>.json`.** For each section, the content comes from the
   checkout, found as described; do not wait for the prompt to name it.
   - `title`, optional `subtitle`, `focus` (see above).
   - `layers`: 4–8 entries, top (entry points) to bottom (leaf infrastructure).
     Every package gets a `layer`; use `side` to split one layer into labelled
     columns (e.g. `"side": "in-process codecs"` vs `"side": "tool installers"`).
   - `steps`: the guided tour, 8–12 steps for a whole-architecture request,
     one per layer boundary from process start to the leaf packages, plus one
     for tests/CI. Each step has `t`, `body` (HTML, see allowed markup below),
     `look` (files to read) and usually `pkg`. Titles are plain phrases; the
     page numbers steps, so a title that starts with its own ordinal is
     rejected.
   - `flows`: one per user-facing command that does real work. Find the
     commands where the CLI registers them (the entry package's command list,
     e.g. the file that builds the `cli.Command`/`cobra.Command` tree), and
     for each follow the call chain from the handler: every stage is a
     function it calls, `where` cites that function's file, `out` says what
     exists afterwards. Each flow has `cmd`, `file`, `inputs`, `stages[]`.
   - `guides`: multi-step deep dives. A whole-architecture request has at
     least these five, each a separate guide with its own id:
     1. how the main command does its job end to end, from the handler to
        the files it writes (in this repository: how `librarian generate`
        produces a library);
     2. how the engine that command delegates to turns its input into output,
        stage by stage with the data structure between stages (here: how
        sidekick turns an API spec into generated code). The engine is the
        subsystem the main command spends its time in, not the most-imported
        package; `-facts` fan-in finds leaf utilities, not engines;
     3. how to add a new instance of the main extension point (whatever the
        `switch`/registry in the orchestration layer dispatches on: a
        language, provider, backend, format);
     4. how to modify an existing instance of it;
     5. when two mechanisms coexist for the same job (in-process vs external,
        old vs new path), how they compare and when to use which.
     A how-to guide shows code: for each step, quote the smallest real example
     in the checkout (a sample or the simplest existing implementation) in
     `<pre><code>`, 5–20 lines, with `<` and `&` escaped, directly after a
     link to the file it came from. Trim with a `// ...` line rather than
     paraphrasing. Never show code that is not in the checkout; if the reader
     must adapt it, say what to rename. Lists of dispatch sites or hooks are
     produced by searching the code (`grep -n 'case config.LanguageX:'`), not
     recalled.
   - Write for someone who has not opened the package. Name the type and field
     ("the `Services`, `Messages` and `Enums` slices of `api.API`"), not a
     shorthand ("walk model.Services"), and say what an operation does the
     first time a term appears ("annotating: storing a per-language struct in
     each node's `Codec` field for templates to read").
   - `findings`: 3–6 short observations a reader should know before changing
     code (more when the user asks for full detail): conventions that are not enforced, duplicated logic, a package that
     every other one imports, a hook only some instances implement. Only
     claims you verified in this session.
   - `langCols` / `langs`: required when the repository has one package per
     target (languages, providers, backends): one row per package in that
     directory. `langCols` names the columns (`{"key": "gen", "title":
     "Generator"}`); each row supplies `cells` keyed by column, plus `pkg` (its
     LOC is computed) and an optional `detail` body. Pick columns that answer
     "how does this target differ"; nothing is built in.
   - `contractCols` / `contractHooks`: with `langs`, the matrix of exported
     hooks per target package. Derive it from the checkout, e.g.
     `grep -lE '^func (Generate|Format|Bump)\(' internal/librarian/*/*.go`,
     and never from memory; a wrong cell here is worse than no table.
   - `defaultPkg`: the package the map selects first.
   - `gitRef`: defaults to the commit, so GitHub links show the code the page
     describes. Set it to a branch only if that commit is not pushed.

5. **Render and fix until clean.** The renderer reports every problem at once.
   It writes `.walkthrough/<name>.html` next to the JSON.

   ```sh
   go run .agents/skills/code-walkthrough/scripts/render.go -in .walkthrough/<name>.json
   ```

   Common errors: a cited path that does not exist, a `pkg` that is not a Go
   package, an unknown field name (typo), a layer id used by a package but not
   declared, a step title that numbers itself. Warnings (a package still on
   the skeleton's `other` layer, a missing section without `focus`) do not
   stop rendering, but `-check` exits non-zero while any remain.

6. **Surface it.** In Jetski, copy `.walkthrough/<name>.html` into the
   artifact directory and register it with `UserFacing: true` so it previews
   in-pane; the "Walkthroughs" links only work when the other pages are
   opened from `.walkthrough/` in a browser. Elsewhere, tell the user to open
   `.walkthrough/<name>.html` in a browser. Offer the next step: a deeper
   guide on one area, which is an edit to the JSON and a re-render.

## JSON reference

One compact example per section. Fields not shown are optional; unknown
fields are errors. `body`, `desc`, `detail`, `where`, `inputs[]` and `cells`
values are HTML (see allowed markup); every other string is plain text.

```jsonc
{
  "title": "…", "subtitle": "…", "focus": "",            // focus only for a single-subsystem request
  "defaultPkg": "internal/foo",                           // map selects this first
  "layers": [{"id": "entry", "title": "Entry points", "hint": "process start"}],
  "pkgs":   [{"id": "cmd/foo", "layer": "entry", "side": "", "desc": "<p>…</p>"}],
  "steps":  [{"t": "One binary, one YAML file", "body": "<p>…</p>",
              "look": ["cmd/foo/main.go"], "pkg": "cmd/foo"}],
  "flows":  [{"id": "generate", "title": "foo generate", "cmd": "foo generate LIB",
              "file": "internal/foo/generate.go", "pkg": "internal/foo",
              "inputs": ["<a data-src=\"foo.yaml\">foo.yaml</a>"],
              "stages": [{"title": "Load config", "where": "<a data-src=\"internal/foo/config.go\">config.go</a>",
                          "body": "<p>…</p>", "out": "*config.Config in memory"}]}],
  "guides": [{"id": "new-target", "title": "Adding a target", "subtitle": "…",
              "audience": "Target authors",                // shown above the title
              "steps": [{"t": "Declare the target", "runs": "Schema",   // shown as "Stage:"
                         "code": "internal/config/",                    // shown as "Code:"
                         "body": "<p>…</p><pre><code>…</code></pre>",
                         "look": ["internal/config/target.go"], "pkg": "internal/config"}]}],
  "findings": [{"t": "Layering is a convention", "b": "<p>…</p>", "f": "AGENTS.md", "pkg": ""}],
  "langCols": [{"key": "gen", "title": "Generator"}],
  "langs":    [{"n": "Go", "pkg": "internal/foo/golang", "cells": {"gen": "protoc plugin"},
                "detail": "<p>…</p>"}],                    // "·" or "" renders as a muted dash
  "contractCols":  ["Go", "Rust"],
  "contractHooks": [{"hook": "Generate", "caller": "generate.go", "cells": ["✓", "✓"]}],
  "gitRef": ""                                             // defaults to the commit
}
```

Computed on output and ignored on input: `sha`, `shortSha`, `repo`, `module`,
`goVersion`, `totalLoc`, `pkgs[].loc`, `pkgs[].imports`, `pkgs[].files`,
`langs[].loc`, `cited`.

## Citing code

Link files and directories from any `body`, `desc`, `where`, `detail` or
language `cells` value with
`<a data-src="internal/foo/bar.go">bar.go</a>` (directories end in `/`). The
renderer fails if the path does not exist, whatever quoting or casing the
attribute uses, and the page itself turns any `data-src` it did not validate
into plain text. The modal shows the path, its owning package, that package's
files and a GitHub link pinned to the commit. Cite the exact file a sentence
is about, not the package root.

## Allowed markup

Bodies are sanitized in the browser. Formatting, list, heading and table tags
survive (`p`, `br`, `hr`, `strong`, `b`, `em`, `i`, `u`, `s`, `code`, `pre`,
`kbd`, `samp`, `var`, `mark`, `small`, `sub`, `sup`, `ul`, `ol`, `li`, `dl`,
`dt`, `dd`, `a`, `span`, `div`, `h3`–`h6`, `blockquote`, `table` and its
parts). Elements that can run script or load resources (`script`, `style`,
`iframe`, `img`, `svg`, `math`, `form`, `input`, …) are removed with their
contents; any other tag is unwrapped so its text still shows. The only
attribute kept is `data-src`; `href`, `class`, `id`, `style` and event handlers
are stripped. Titles (`t`, `title`, `subtitle`, `audience`, `runs`, `code`,
`cmd`, `out`) are plain text: no HTML and no Markdown (backticks will show
literally). The renderer HTML-escapes the JSON payload, so content cannot
break out of the page's data block, and the page's CSP allows only the
template's own script by hash.

## Checks before you hand it over

- `render.go -check` passes with no warnings. A whole-architecture page has
  flows, guides and findings; if it warns that one is missing, write it
  rather than setting `focus`.
- Every step's `look` list points at files you actually read.
- Counts and names in prose (number of languages, package names, function
  names) match what you saw; prefer "the language packages under
  `internal/librarian/`" over a number that will go stale.
- Every file you wrote is under `.walkthrough/`, and `git status` shows none
  of them.

## Files

```text
.agents/skills/code-walkthrough/
├── SKILL.md                this file
└── scripts/
    ├── render.go           go list facts + validation + HTML output
    ├── render_test.go      validation, merge, output-directory and -init/-check round-trip tests
    └── template.html       static page embedded into render.go; fills the {{.JSON}} slot
```

The template is repo-agnostic. Change it only to add a visualization; never
put repository names, package lists or prose in it. `-template PATH` renders
with a different template file for experiments.

## Scope and limits

- Go modules only: facts come from `go list ./...`, which needs the module's
  dependencies resolvable (a populated module cache or network).
- Packages with no non-test Go files are omitted from the map; the module
  root package, if it has code, appears as `.`.
- The GitHub link is derived from a `github.com/ORG/REPO` module path. For
  other hosts set `repo` in the JSON or accept that the link is hidden.
- "Copy prompt for agent" buttons interpolate guide titles and paths into the
  prompt text. Write them as you would write them into a chat yourself.
