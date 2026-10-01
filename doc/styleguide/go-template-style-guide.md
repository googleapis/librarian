# Go Template Style Guide

## Overview

This guide defines conventions for authoring and formatting Go templates
(`.gotmpl`) across Librarian sidekick generators.

In the Go ecosystem, these conventions are based on the
[Helm Template Formatting Guide](https://helm.sh/docs/chart_best_practices/templates/#formatting-templates)
and Hugo's template practices, adapted for target source code generation.

## Core Principles

1. **Visual Alignment:** Template control flow must visually reflect the structure
   and indentation of the generated code.
2. **Zero Trailing Whitespace:** Blank lines and code lines must never contain
   trailing spaces or tabs.
3. **Explicit Indentation:** Use `| indent N` explicitly for nested partials.
4. **Predictable Whitespace Trimming:** Use whitespace trimming flags (`{{-` and
   `-}}`) deliberately to avoid runaway vertical whitespace or accidental code
   squashing.

---

## Formatting Standards

### 1. Visual Alignment with Target Code

Indent template control actions (`{{- if }}`, `{{- range }}`, `{{- end }}`) so
that they align with the logical block level of the surrounding target code.
Never pull template tags all the way to column 0 if the generated code is
indented within a class, function, or struct.

#### Good:
```gotmpl
public class {{ .Name }} {
    {{- range .Codec.Methods }}
    {{- "\n" }}
    public func {{ .Name }}() {
        {{- include "templates/common/method_body" . | indent 8 }}
    }
    {{- end }}
}
```

#### Bad:
```gotmpl
public class {{ .Name }} {
{{- range .Codec.Methods }}
{{- "\n" }}
    public func {{ .Name }}() {
{{- include "templates/common/method_body" . | indent 8 }}
    }
{{- end }}
}
```

---

### 2. Emitting Blank Lines

To emit a blank line between generated blocks or methods, use `{{- "\n" }}`
indented to match the surrounding block.

#### Why `{{- "\n" }}` is the Recommended Pattern:
- **Trims preceding template whitespace:** The leading `{{-` trims the newline
  and indent of the previous template line.
- **Emits clean newline:** The `"\n"` emits a single newline character.
- **Zero trailing spaces:** The following literal newline after `}}` completes
  the blank line (`\n\n`) with zero trailing whitespace.
- **Preserves visual indentation:** `{{- "\n" }}` can be indented to match the
  surrounding template code, maintaining clean visual flow.

#### Good:
```gotmpl
    {{- range .Codec.Methods }}
    {{- "\n" }}
    {{- range .DocLines }}
    /// {{ . }}
    {{- end }}
    public func {{ .Name }}() {}
    {{- end }}
```

#### Bad (Column 0 Jump):
```gotmpl
    {{- range .Codec.Methods }}
{{ "" }}
    {{- range .DocLines }}
    /// {{ . }}
    {{- end }}
    public func {{ .Name }}() {}
    {{- end }}
```
*Problem:* Breaking visual indentation to place `{{ "" }}` at column 0 makes
the template jarring to read.

#### Bad (Indented Empty Action):
```gotmpl
    {{- range .Codec.Methods }}
    {{ "" }}
    {{- range .DocLines }}
    /// {{ . }}
    {{- end }}
    public func {{ .Name }}() {}
    {{- end }}
```
*Problem:* Indenting `    {{ "" }}` writes 4 spaces onto the blank line,
producing trailing whitespace that fails linters and git hooks.

---

### 3. Partial Inclusion and Indentation

When calling partial templates, explicitly pipe the output through the `indent`
helper:

```gotmpl
    {{- include "templates/common/method_signature" . | indent 4 }}
```

- Prefer explicit `| indent N` over implicit or "magic" indentation inference.
  Explicit indentation makes the emitted column depth obvious to template
  readers and is robust across whitespace trimming flags.
- When nesting partials within indented blocks, increase the indent accordingly
  (e.g., 4 spaces for class body, 8 spaces for method body).

---

### 4. Whitespace Trimming (`{{-` and `-}}`)

Be disciplined with trim flags:
- Use `{{-` when the preceding indentation or newline in the template is merely
  syntactic formatting for the template itself and should not appear in output.
- Avoid trailing `-}}` unless you deliberately intend to swallow all whitespace
  (including newlines) that follows the tag. Overusing trailing `-}}` often
  accidentally collapses consecutive lines or swallows desired blank lines.

---

### 5. Syntax and Spacing

- Always include a space between curly braces and actions:
  - `{{- if .Condition }}` (correct)
  - `{{-if .Condition}}` (incorrect)
- Use standard Go template operators and pipelines (`eq`, `ne`, `not`, `and`,
  `or`).

---

### 6. Comments and License Headers

- Use Go template comment syntax `{{/* ... */}}` for template comments.
  These comments are evaluated at parse time and are completely omitted from
  generated output.
- **Template License Header:** Every `.gotmpl` file must begin with an
  Apache 2.0 license header enclosed in template comments:
  ```gotmpl
  {{/*
  Copyright 2026 Google LLC

  Licensed under the Apache License, Version 2.0 (the "License");
  ...
  */}}
  ```
