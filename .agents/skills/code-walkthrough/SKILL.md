---
name: code-walkthrough
description: Produces a self-contained interactive HTML architecture walkthrough (dependency map, guided tour, command flows, guides) for this repository at the current commit. Use when asked to "walk me through the code", explain the architecture or a subsystem, or build a guide for modifying or extending a part of the codebase. The narrative is written fresh against the checkout; the facts (packages, imports, line counts) are computed.
---

# Code walkthrough

## What you produce

A single `walkthrough.html` that opens from `file://` with no server and no
network. It has an architecture map (every Go package in the module, grouped
into layers you assign, with import wires drawn from `go list`), a guided tour,
one flow per CLI command, deep-dive guides, a per-target table and findings. The
HTML is generated; never commit it.

Everything you write goes in one JSON file. Everything that is a fact about the
code is computed by the renderer and cannot be typed by hand:

| You author in JSON | Renderer computes |
|---|---|
| layers, each package's layer and (optionally) a better one-line description | package list, imports, non-test LOC, doc synopses, module path, Go version, repo, commit |
| tour steps, flows, guides, findings, languages table | validation of every cited path and package |

## Workflow

1. **Get the skeleton.** It lists every package with its doc comment synopsis.

   ```sh
   go run .agents/skills/code-walkthrough/scripts/render.go -init -out walkthrough.json
   ```

   `-facts` prints the same computed facts as JSON (`module`, `goVersion`,
   `sha`, `pkgs[]` with `imports` and `loc`). Use it for any claim about who
   imports what ("only X and Y import Z", "nothing imports W") instead of
   reading import blocks by hand.

2. **Read before you write.** Open the packages and files relevant to the
   request. Do not describe code you have not read in this session; do not
   reuse text from an earlier walkthrough. Package doc comments are usually the
   best one-line description; if one is missing or wrong, fix the comment in
   the package (that helps godoc too) rather than only overriding `desc`.

3. **Decide the scope.** "Walk me through the code / the architecture" or any
   request that does not name one subsystem is a **whole-architecture**
   request and produces every section below. A request about one subsystem
   or one command sets `focus` to that subsystem's name and produces the
   tour (3–6 steps) plus the guide or flow it asked for; the renderer warns
   about missing sections unless `focus` is set.

4. **Edit `walkthrough.json`.** For each section, the content comes from the
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
   - `guides`: multi-step deep dives, at least these for a whole-architecture
     request:
     1. how the core engine works end to end (the subsystem most packages
        serve; pick it by fan-in in `-facts`);
     2. how to add a new instance of the main extension point (whatever the
        `switch`/registry in the orchestration layer dispatches on: a
        language, provider, backend, format);
     3. how to modify an existing instance of it;
     4. when two mechanisms coexist for the same job (in-process vs external,
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
     code: conventions that are not enforced, duplicated logic, a package that
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

   ```sh
   go run .agents/skills/code-walkthrough/scripts/render.go -in walkthrough.json -out walkthrough.html
   ```

   Common errors: a cited path that does not exist, a `pkg` that is not a Go
   package, an unknown field name (typo), a layer id used by a package but not
   declared. Packages missing from the JSON, or still on the skeleton's
   `other` layer, are not an error; they appear under "Unclassified" with a
   warning each, so assign them.

6. **Surface it.** In Jetski, copy `walkthrough.html` into the artifact
   directory and register it with `UserFacing: true` so it previews in-pane.
   Elsewhere, tell the user to open it in a browser. Offer the next step: a
   deeper guide on one area, which is an edit to the JSON and a re-render.

## Citing code

Link files and directories from any `body`, `desc`, `where`, `detail` or
language `cells` value with
`<a data-src="internal/foo/bar.go">bar.go</a>` (directories end in `/`). The
renderer fails if the path does not exist; the page shows the path, its owning
package and a GitHub link. Cite the exact file a sentence is about, not the
package root.

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
break out of the page's data block.

## Checks before you hand it over

- `render.go -check` passes with no warnings. A whole-architecture page has
  flows, guides and findings; if it warns that one is missing, write it
  rather than setting `focus`.
- Every step's `look` list points at files you actually read.
- Counts and names in prose (number of languages, package names, function
  names) match what you saw; prefer "the language packages under
  `internal/librarian/`" over a number that will go stale.
- `walkthrough.json` and `walkthrough.html` are not staged for commit.

## Files

```text
.agents/skills/code-walkthrough/
├── SKILL.md                this file
└── scripts/
    ├── render.go           go list facts + validation + HTML output
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
