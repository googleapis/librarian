# How We Write Code

This document distills foundational, language-agnostic engineering and code
review principles to preemptively address recurring concerns in pull requests.

## 1. Architectural Boundaries & Cross-Language Consistency

*   **Respect Boundaries:** Place language-specific logic within its dedicated
    package. API-level properties belong in shared service configs, while
    repository-level logic goes to the librarian config. Centralize shared,
    cross-cutting logic into common packages.
*   **Cross-Language Consistency:** If a solution exists for one language
    (e.g., dependency management), adopt that established pattern rather than
    inventing a new one for a different language.

## 2. Refactoring, DRY & State Management

*   **Extract Distinct Phases & Helpers:** Break down long functions into
    high-level orchestrators by extracting distinct logic blocks into helper
    functions. Only deduplicate repeated logic when there are 3 or more
    occurrences or the shared dependency is acceptable ("a little copying is
    better than a little dependency").
*   **Atomicity in State Mutations:** Never mutate shared state or
    configuration objects before fallible operations. Perform all checks first,
    and only mutate state when success is guaranteed.
*   **Inline Single-Use Variables:** If a variable is used only immediately
    after its declaration (e.g., as arguments to a command), inline it to
    reduce boilerplate.
*   **Minimize Vertical Density:** Group related code tightly. Avoid
    unnecessary blank lines, using them only to signal a shift in logic.
*   **Avoid Overly Defensive Code:** Do not add redundant checks for files or
    data that are fully controlled by the project, or if existence is already
    guaranteed by previous pipeline steps.

## 3. Reproducibility & Integrity

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

## 4. Clean and Intentional Testing

*   **Assert on Side-Effects:** Do not simply verify that no error occurred.
    Assert on the actual outputs, side-effects, or generated files resulting
    from the function call.
*   **Decouple Functional and Validation Tests:** Avoid duplicating validation
    error modes in functional integration tests. If a validation function has
    thorough error tests, the functional test only needs to verify the happy
    path.
*   **Test Realistic Scenarios:** Avoid writing tests for impossible states
    (e.g., passing Python configurations to Java validators).

## 5. General Code Health

*   **Provide Evidence for Complex Changes:** When making changes to internal
    tooling or generation scripts, attach resulting diffs from a real target
    repository in the PR to prove it works as intended.
*   **Minimize Exported Surface Area:** Keep internal helpers, constants, and
    error definitions unexported if they have no usage outside of their local
    package or module.
*   **Maintain Professional Tone:** Ensure code review comments, commit
    messages, and PR descriptions maintain a constructive and professional
    tone.
