---
name: walkthrough-basic
description:
  Generates a self-contained interactive HTML walkthrough (architecture map
  with import wires, guided tours, command flows, findings, and an in-page
  source inspector) by analyzing the current Go code. Use when asked to walk
  through the codebase, explain the architecture, onboard someone, or produce
  an interactive code tour.
---

# Interactive Code Walkthrough (basic)

This skill produces one HTML file that works offline with no server. Nothing
about the repository is hardcoded: mechanical facts come from
[`scripts/walkthrough.go`](scripts/walkthrough.go), and the explanatory text is
written by you after reading the code. The render step rejects any narrative
that references packages or files that do not exist.

| Piece                    | Source                                 | Written by          |
| ------------------------ | -------------------------------------- | ------------------- |
| Packages, LOC, imports   | `go/build` + `go/ast`                  | `analyze`           |
| Exported symbols         | `go/ast`                               | `analyze`           |
| CLI commands             | `*.Command{Name/Use: ...}` literals    | `analyze`           |
| Commit, ref, Go version  | `git`, `go.mod`                        | `analyze`           |
| Layers, descriptions     | Reading the code and the import graph  | You (narrative)     |
| Tours, flows, findings   | Reading the code                       | You (narrative)     |
| Inlined source previews  | Every file the narrative references    | `render`            |

## Workflow

Run all commands from the module root. Keep intermediate files out of the
repository, for example in a scratch directory (`$SCRATCH` below).

1.  **Extract facts.**

    ```sh
    go run .agents/skills/walkthrough-basic/scripts/walkthrough.go analyze -root . -out $SCRATCH/facts.json
    ```

    Read `facts.json`. It lists every package with its doc synopsis, LOC,
    files, internal `imports` and `importedBy`, exported symbols, and
    `testOnly`; every detected CLI command with its file, line, and action; and
    the top-level docs.

2.  **Read the code.** Do not write anything yet. At minimum, read:
    - `README.md`, `AGENTS.md`, and the docs listed under `docs` that describe
      architecture.
    - Every `main` package and the root CLI command.
    - The `action` function of each detected command you plan to trace.
    - The most-imported packages (high `importedBy`) and the packages with the
      most imports (orchestrators).

3.  **Design layers from the import graph.** Group packages into 4–8 layers,
    ordered top (entry points) to bottom (leaf infrastructure), so that imports
    point downward. Packages with `testOnly: true` and other fixtures go in a
    final layer with `"id": "test"`, which is exempt from layering checks.
    `render` warns about every other upward import; fix the grouping or
    mention the exception in a finding.

4.  **Write `$SCRATCH/narrative.json`** following the
    [schema](#narrative-schema). Rules:
    - Only describe behavior you verified by opening the file. Cite it with
      `<a class="src" data-src="path/to/file.go">label</a>` or a `look` entry.
    - Never type counts. Use tokens, which `render` resolves from facts:
      `{{packages}}`, `{{totalLoc}}`, `{{commands}}`, `{{sha}}`,
      `{{goVersion}}`, `{{loc:PKG}}`, `{{files:PKG}}`, `{{imports:PKG}}`,
      `{{importedBy:PKG}}`, `{{lines:FILE}}`.
    - Directory references end with `/` (for example `internal/config/`).
    - Link packages with `<a class="pkgref" data-pkg="internal/x">…</a>`.
    - HTML fields allow only `p`, `code`, `strong`, `em`, `ul`, `li`, and the
      two link forms above. Plain-text fields are escaped.

5.  **Render and fix until clean.**

    ```sh
    go run .agents/skills/walkthrough-basic/scripts/walkthrough.go render -root . \
      -facts $SCRATCH/facts.json -narrative $SCRATCH/narrative.json \
      -out $OUT/walkthrough.html
    ```

    `render` fails on unknown packages, missing files, unassigned or duplicated
    packages, and unknown tokens. Fix the narrative, not the script. Add
    `-inline-all` to inline every package file (larger output, every file chip
    on the map becomes viewable).

6.  **Verify and deliver.** Extract the second `<script>` block and run
    `node --check` on it if Node is available. Open the HTML (or attach it as an
    artifact) and report the path. Do not commit generated files.

## Narrative schema

```json
{
  "title": "Project architecture & walkthroughs",
  "summary": "<p>HTML: what the project is and how it is structured.</p>",
  "layers": [
    {"id": "entry", "title": "Entry points", "hint": "process start", "packages": ["cmd/app"]}
  ],
  "descriptions": {
    "cmd/app": "HTML, one or two sentences per package."
  },
  "tours": [
    {
      "id": "core",
      "title": "Core architecture",
      "subtitle": "From process start to output",
      "group": "Overall architecture",
      "steps": [
        {
          "t": "Step title",
          "runs": "app generate (optional command this step explains)",
          "body": "<p>HTML.</p>",
          "look": ["cmd/app/main.go"],
          "pkg": "cmd/app"
        }
      ]
    }
  ],
  "flows": [
    {
      "id": "generate",
      "title": "generate",
      "cmd": "app generate NAME",
      "file": "internal/app/generate.go",
      "steps": [["<a class=\"src\" data-src=\"internal/app/generate.go\">generate.go</a>", "HTML: what happens."]]
    }
  ],
  "findings": [
    {"t": "Title", "b": "HTML observation.", "f": "internal/app/generate.go", "pkg": "internal/app"}
  ]
}
```

Aim for:

- **Descriptions** for every package (missing ones fall back to the doc
  comment, with a warning).
- **Tours:** one core tour of 6–12 steps that follows a request from `main`
  to output, plus 1–3 deep-dive tours for the most complex subsystems. Tours
  sharing a `group` appear together on the Paths tab.
- **Flows:** 2–4 of the most important commands, traced call by call from
  the command's `action`, one step per hop.
- **Findings:** 4–8 non-obvious observations (design decisions, invariants,
  enforced rules, risks), each anchored to a file.

## Principles

- **Generated, not remembered:** if a fact can be computed, compute it or use
  a token. Re-run the skill after code changes instead of editing the HTML.
- **Cite everything:** every step and finding links to a file you read.
- **Simple output:** one HTML file, no network, no external assets.
