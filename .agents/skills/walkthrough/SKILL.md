---
name: walkthrough
description: Builds a self-contained interactive Librarian architecture map and step-by-step developer walkthrough artifact (.html or .zip) directly from the live codebase with zero background server. Use when asked to walk through the code ("walk me through the code"), explore Librarian's architecture or dependencies, compare sidekick vs external micro-generators, modify or add a sidekick language, or generate custom outputs/parsers.
---

# Interactive Architecture Map & Walkthrough Artifact

## Overview

`tool/cmd/walkthrough` analyzes the checked-out Go repository tree using `go/ast` and compiles a **single, self-contained HTML artifact** (or portable `.zip` archive) with zero servers, zero external CDN/runtime dependencies, and zero manual drift:

- **Live SVG Architecture & Dependency Map**: Every Go package across all 7 layers (`Entry points`, `Command orchestration`, `Language integrations`, `Generation engine (sidekick) | Tool installers`, `Domain services`, `Infrastructure`, and `Test support`), non-test Go LOC, and direct/reverse intra-repo dependencies (`imports` in blue, `imported by` in emerald) computed live from the AST.
- **Overall Architecture Guided Tour & Command Timelines**: Step-by-step tour from `cmd/librarian` and `librarian.yaml` through source tarballs, parallel generation, post-processing, and `bump → publish → tag`.
- **Languages & AST-Verified Contract Matrix**: Live comparison of all 9 language packages and every exported lifecycle function verified directly against the Go AST.
- **4 Built-In Generation Deep Dives**:
  1. *Sidekick vs. External Micro-Generators*
  2. *Modifying Sidekick Generation for an Existing Language* (`rust`, `swift`, `dart`)
  3. *Adding a New Language Generator with Sidekick*
  4. *New Input Sources (`sidekick/parser`) & Custom Non-Cloud-SDK Outputs (`templates/`)*

## 1. Open Directly in Jetski / Any Agent Harness or Browser (Zero Server Required)

When a developer asks *"walk me through the code"* (or asks about architecture, layers, dependencies, or code generation):

1. Build the standalone single-file HTML artifact (or `.zip` archive):
   ```sh
   # Self-contained HTML file (opens directly in Jetski artifact preview, Claude Artifacts, or any browser via file://)
   go run ./tool/cmd/walkthrough -out walkthrough.html

   # Optional portable zip package
   go run ./tool/cmd/walkthrough -out walkthrough.zip
   ```
2. **In Jetski / Artifact-Enabled Harnesses**: Write or register `walkthrough.html` in the conversation artifact directory with `UserFacing: true` so it renders directly in the side preview panel with one click!
3. Ask which learning path the developer wants to explore or customize together:
   - Overall architecture & package dependency map (**Map** & **Guided tour**)
   - Generation internals: Sidekick vs. external `protoc` GAPIC micro-generators
   - Modifying Sidekick generation/templates for an existing language (`rust`, `swift`, `dart`)
   - Adding a brand-new language generator with Sidekick
   - Supporting new specification input sources (`internal/sidekick/parser`) or non-Cloud-SDK outputs via custom template trees (`internal/sidekick/language`)

## 2. Adding Custom Local Walkthroughs on Demand

Developers can ask you at any time to explain a specific subsystem or create a brand-new interactive walkthrough (e.g., *"Create a walkthrough showing how Rust workspace dependency resolution and bump work"*).

Drop a JSON file into `.agents/skills/walkthrough/custom/<slug>.json` matching the `Guide` schema in `tool/cmd/walkthrough/model.go`:

```json
{
  "id": "rust-bump-and-deps",
  "title": "Rust Workspace Dependency Resolution & Bump",
  "subtitle": "How internal/librarian/rust resolves Cargo crate dependencies and computes SemVer bumps",
  "audience": "Custom local walkthrough",
  "steps": [
    {
      "t": "Crate dependency resolution during add and tidy",
      "runs": "librarian add / tidy",
      "code": "internal/librarian/rust/",
      "body": "<p>Explain the exact functions and data flow with <code>code</code> and <a class=\"src\" data-src=\"internal/librarian/rust/\">source links</a>.</p>",
      "look": ["internal/librarian/rust/"],
      "pkg": "internal/librarian/rust"
    }
  ]
}
```

Then regenerate `walkthrough.html` and update the preview artifact:
```sh
go run ./tool/cmd/walkthrough -out walkthrough.html
```
The custom walkthrough appears immediately on the **Paths** landing page and inside **Generation guides** alongside the built-in paths.

## 3. Verifying Changes

```sh
go test -race ./tool/cmd/walkthrough/...
```
