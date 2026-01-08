# Gemini Code Assist Style Guide

This guide defines the standards for code generation, refactors, and code
reviews for this repository.

When reviewing or generating code, apply the following checks and references:

- **Follow local standards:** Ensure all changes conform to
  [`doc/howwewritego.md`](../doc/howwewritego.md), which defines the project’s
  architectural patterns, design decisions, and testing requirements.
- **Verify testing quality:** Add or update tests as needed, following the
  guidance in [Go Test Comments](https://go.dev/wiki/TestComments).
- **Enforce idiomatic Go:** Flag patterns that conflict with the recommendations
  in [Go Code Review Comments](https://go.dev/wiki/CodeReviewComments).
- **Write proper commit messages:** Follow the conventions in
  [`CONTRIBUTING.md` → Commit Messages](../CONTRIBUTING.md#commit-messages).
  Commit messages should be written in complete sentences and use the
  imperative mood.
- **Avoid excessive blank lines:** Use line breaks only to signal context
  shifts. Avoid unnecessary vertical padding.
