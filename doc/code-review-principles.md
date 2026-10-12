# Code Review Principles

This document distills key code review feedback from maintainers to
preemptively address recurring concerns in pull requests. These principles
supplement the foundational guidelines in [How We Write Go](/doc/howwewritego.md)
and [CONTRIBUTING.md](/CONTRIBUTING.md).

## 1. Architectural Boundaries & Cross-Language Consistency

*   **Respect Boundaries:** Place language-specific logic within its dedicated
    package. API-level properties belong in shared service configs, while
    repository-level logic goes to the librarian config. Centralize shared,
    cross-cutting logic into common packages.
*   **Cross-Language Consistency:** If a solution exists for one language
    (e.g., dependency management), adopt that established pattern rather than
    inventing a new one for a different language.

## 2. Refactoring, DRY & State Management

*   **Extract Distinct Phases & Helpers:** Do not repeat yourself (DRY). Break
    down long functions into high-level orchestrators by extracting distinct
    logic blocks and repeated operations into helper functions.
*   **Atomicity in State Mutations:** Never mutate shared state or
    configuration objects before fallible operations. Perform all checks first,
    and only mutate state when success is guaranteed.
*   **Inline Single-Use Variables:** If a variable is used only immediately
    after its declaration (e.g., as arguments to a command), inline it to
    reduce boilerplate.
*   **Minimize Vertical Density:** Group related code tightly. Avoid
    unnecessary blank lines, using them only to signal a shift in logic.

## 3. Error Handling & Resource Management

*   **Explicit and Independent Error Checks:** Do not combine `err == nil` with
    other conditions. Always check `err != nil` explicitly, and gracefully
    handle errors rather than masking them. Do not use redundant nil checks
    before `errors.Is` or `errors.As`.
*   **Combine and Contextualize Errors:** Combine related error branches into a
    single contextual `fmt.Errorf` that wraps the original error.
*   **Proper File Closing and I/O:** Always check errors returned by `Close()`
    inside a `defer` for file writes using named returns. Alternatively, prefer
    `os.WriteFile` when writing complete files. Check if file content was
    actually modified before overwriting to save I/O operations.
*   **Avoid Overly Defensive Code:** Do not add redundant checks (like
    `os.Stat`) for files or data that are fully controlled by the project, or
    if existence is already guaranteed by previous pipeline steps.

## 4. Reproducibility & Integrity

*   **Pin Tool and Dependency Versions:** Never use floating tags like
    `"latest"` for tools or dependencies. Always enforce pinned, stable
    versions for reproducible builds.
*   **Prefer Immutable Identifiers:** Use specific commit SHAs rather than
    resolving branch names when downloading remote tools or checking out
    repositories. Verify file integrity using SHA-256 checksums where
    applicable.
*   **Avoid Outright Breaking Changes:** Break large, potentially breaking
    refactors across several smaller PRs. Maintain backward compatibility
    (e.g., for legacy output paths) during transitions.

## 5. Clean and Intentional Testing

*   **Isolate Test State:** Always initialize mutable test data inside the
    `t.Run` block to prevent subtests from polluting each other's state. Use
    `t.Chdir` for directory changes and `t.Cleanup` for any global
    modifications.
*   **Standardize Test Variables:** Use `got` and `want` for single-value
    comparisons. When checking errors against a wanted error, consistently use
    `gotErr` over `err`.
*   **Assert on Side-Effects:** Do not simply verify that no error occurred.
    Assert on the actual outputs, side-effects, or generated files resulting
    from the function call.
*   **Decouple Functional and Validation Tests:** Avoid duplicating validation
    error modes in functional integration tests. If a validation function has
    thorough error tests, the functional test only needs to verify the happy
    path.
*   **Test Realistic Scenarios:** Avoid writing tests for impossible states
    (e.g., passing Python configurations to Java validators).

## 6. General Code Health

*   **Modern Octal Literals:** Always use the modern Go `0o` prefix (e.g.,
    `0o644`) for octal literals instead of the legacy `0` prefix.
*   **Provide Evidence for Complex Changes:** When making changes to internal
    tooling or generation scripts, attach resulting diffs from a real target
    repository in the PR to prove it works as intended.
*   **Minimize Exported Surface Area:** Keep sentinel errors and helper
    variables unexported if they have no usage outside of their local package.
*   **Maintain Professional Tone:** Ensure code review comments, commit
    messages, and PR descriptions maintain a constructive and professional
    tone.
