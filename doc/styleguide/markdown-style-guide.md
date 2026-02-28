# Markdown Style Guide

_This document was copied from
https://google.github.io/styleguide/docguide/style.html for use by Gemini._

---

## Overview

This Google style guide emphasizes writing clear, maintainable Markdown that
balances three key goals: readable source text, maintainable documentation over
time, and simple, memorable syntax.

## Core Principles

### Minimum Viable Documentation

"A small set of fresh and accurate docs is better than a sprawling, loose
assembly of documentation in various states of disrepair." Focus on essential
documentation and regularly remove outdated material.

### The Better/Best Rule

Documentation review standards differ from code reviews. Prioritize fast
iteration and author productivity. Reviewers should trust authors to fix issues
and suggest alternatives rather than vague critiques.

## Formatting Standards

### Capitalization

Preserve original capitalization for products, tools, and binaries. Use proper
capitalization in code examples and technical references.

### Document Layout

Recommended structure:

- H1 title (matches filename when possible)
- Brief 1-3 sentence introduction
- `[TOC]` directive (if supported)
- H2+ headings for content sections
- "See also" section at bottom

### Table of Contents

Place `[TOC]` after the introduction but before the first H2 heading. This
ensures accessibility for screen readers and keyboard navigation.

### Character Line Limit

"Markdown content follows the residual convention of an 80-character line limit"
for consistency with coding practices. Exceptions include links, tables,
headings, and code blocks.

### Trailing Whitespace

Avoid trailing spaces. Use backslashes for line breaks instead of double spaces.

## Headings

**Use ATX-style headings** (`#`, `##`, etc.) rather than underlined headings.
Provide unique, complete names for each heading to create intuitive anchor
links. Include spacing after `#` and blank lines before/after headings. Restrict
yourself to one H1 per document.

Follow Google's
[capitalization guidance](https://developers.google.com/style/capitalization)
for titles and headers.

## Lists

### Numbering

Use lazy numbering for long, changing lists (all items marked `1.`). For short,
stable lists, use sequential numbering for clarity in source.

### Nesting

Use consistent 4-space indentation for nested items. Place 2 spaces after list
numbers and 3 spaces after bullets.

## Code

### Inline Code

Use backticks for short code quotations, field names, and generic file types:

```
Pay attention to the `foo_bar_whammy` field.
```

### Code Blocks

Use fenced code blocks with explicit language declarations. Always prefer fenced
blocks over indented ones. When nesting code blocks within lists, indent
appropriately to maintain list structure. Escape newlines in command-line
snippets using trailing backslashes.

## Links

### Best Practices

Use explicit paths for internal Markdown links: `[...](/path/to/page.md)` rather
than full URLs. Avoid relative paths with `../`. Write naturally and wrap the
most relevant phrase with link text.

### Reference Links

Reserve reference links for lengthy URLs that would disrupt readability. Define
reference links just before the next heading in their section, or at the
document end if used across multiple sections.

## Images

Use images sparingly to show rather than describe. Provide descriptive alt text
for accessibility. Screenshots work best when they clarify navigation or complex
visual concepts.

## Tables

Use tables for uniform, tabular data that needs quick scanning. Avoid tables for
data better suited to lists. Keep cells concise -- use reference links to manage
length. Ensure good data distribution; unbalanced dimensions signal a list
format would work better.

## HTML Usage

"Strongly prefer Markdown to HTML hacks." Standard Markdown handles nearly all
documentation needs. HTML reduces readability and portability. Note: Gitiles
does not render HTML.

---

**Philosophy:** This guide prioritizes plain text clarity, consistent
formatting, and team maintainability over aesthetic perfection or feature
maximization.
