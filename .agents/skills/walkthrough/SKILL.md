---
name: walkthrough
description: Produces a self-contained interactive HTML architecture walkthrough (dependency map, guided tour, command flows, guides) for this repository at the current commit. Use when asked to "walk me through the code", explain the architecture or a subsystem, or build a guide for modifying or extending a part of the codebase. The narrative is written fresh against the checkout; the facts (packages, imports, line counts) are computed.
---

# Walkthrough

## What you produce

A single `walkthrough.html` that opens from `file://` with no server and no
network. It has an architecture map (every Go package in the module, grouped
into layers you assign, with import wires drawn from `go list`), a guided tour,
optional command flows, deep-dive guides, a languages table and findings. The
HTML is generated; never commit it.

Everything you write goes in one JSON file. Everything that is a fact about the
code is computed by the renderer and cannot be typed by hand:

| You author in JSON | Renderer computes |
|---|---|
| layers, each package's layer and (optionally) a better one-line description | package list, imports, non-test LOC, doc synopses, module path, repo, commit |
| tour steps, flows, guides, findings, languages table | validation of every cited path and package |

## Workflow

1. **Get the skeleton.** It lists every package with its doc comment synopsis.

   ```sh
   go run .agents/skills/walkthrough/scripts/render.go -init -out walkthrough.json
   ```

2. **Read before you write.** Open the packages and files relevant to the
   request. Do not describe code you have not read in this session; do not
   reuse text from an earlier walkthrough. Package doc comments are usually the
   best one-line description; if one is missing or wrong, fix the comment in
   the package (that helps godoc too) rather than only overriding `desc`.

3. **Edit `walkthrough.json`.**
   - `title`, optional `subtitle`.
   - `layers`: 4–8 entries, top (entry points) to bottom (leaf infrastructure).
     Every package gets a `layer`; use `side` to split one layer into labelled
     columns (e.g. `"side": "in-process codecs"` vs `"side": "tool installers"`).
   - `steps`: the guided tour, 6–12 steps for a whole-architecture request, or
     3–6 focused on the subsystem the user asked about. Each step has `t`,
     `body` (HTML, see allowed markup below), `look` (files to read) and
     usually `pkg`.
   - `flows`: only when the request is about a command's execution; each has
     `cmd`, `file`, `inputs`, `stages[]` with `where`/`body`/`out`.
   - `guides`: multi-step deep dives ("how to add X", "how Y works"). This is
     where a request like "walk me through LRO handling" belongs.
   - `findings`: short observations a reader should know before changing code.
     Only claims you verified in this session.
   - `langs` / `contractCols` / `contractHooks`: optional per-language table.
     Derive hook presence with
     `grep -lE '^func (Generate|Format|Bump)\(' internal/librarian/*/*.go`
     style commands, not from memory.
   - `defaultPkg`: the package the map selects first.

4. **Render and fix until clean.** The renderer reports every problem at once.

   ```sh
   go run .agents/skills/walkthrough/scripts/render.go -in walkthrough.json -out walkthrough.html
   ```

   Common errors: a cited path that does not exist, a `pkg` that is not a Go
   package, an unknown field name (typo), a layer id used by a package but not
   declared. Packages missing from the JSON are not an error; they appear
   under "Unclassified" with a warning, so assign them.

5. **Surface it.** In Jetski, copy `walkthrough.html` into the artifact
   directory and register it with `UserFacing: true` so it previews in-pane.
   Elsewhere, tell the user to open it in a browser. Offer the next step: a
   deeper guide on one area, which is an edit to the JSON and a re-render.

## Citing code

Link files and directories from any `body`, `desc`, `where` or `detail` with
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

- `render.go -check` passes with no warnings.
- Every step's `look` list points at files you actually read.
- Counts and names in prose (number of languages, package names, function
  names) match what you saw; prefer "the language packages under
  `internal/librarian/`" over a number that will go stale.
- `walkthrough.json` and `walkthrough.html` are not staged for commit.

## Files

```text
.agents/skills/walkthrough/
├── SKILL.md                 this file
├── resources/template.html  static page; the renderer fills the {{.JSON}} slot
└── scripts/render.go        go list facts + validation + HTML output
```

The template is repo-agnostic. Change it only to add a visualization; never
put repository names, package lists or prose in it.
