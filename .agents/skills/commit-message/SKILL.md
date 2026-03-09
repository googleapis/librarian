---
name: commit-message
description:
  Use this skill when asked to write or draft a commit message. It ensures
  commit messages follow the conventions in CONTRIBUTING.md.
---

# Commit Message Writer

This skill drafts commit messages that follow the conventions in
[CONTRIBUTING.md](../../../CONTRIBUTING.md#commit-messages).

## Workflow

Follow these steps to draft a commit message:

1.  **Gather Context**: Understand what changed and why.
    - Run `git diff --staged` to see staged changes.
    - If nothing is staged, run `git diff` to see unstaged changes.
    - Run `git log --oneline -10` to see recent commit style for reference.

2.  **Determine the Type**: Choose the conventional commit type based on the
    nature of the change:
    - `feat`: A new feature
    - `fix`: A bug fix
    - `docs`: Documentation only changes
    - `test`: Adding or updating tests
    - `refactor`: Code change that neither fixes a bug nor adds a feature
    - `chore`: Changes to the build process or auxiliary tools
    - See https://www.conventionalcommits.org/en/v1.0.0/#summary for the full
      list.

3.  **Determine the Package**: Identify the Go package most affected by the
    change. Use the package path relative to the module root (for example,
    `internal/librarian` or `cli`).

4.  **Draft the First Line**: Write the first line following the format
    `<type>(<package>): <description>`.
    - The description completes the sentence "This change modifies Librarian
      to ..."
    - It does not start with a capital letter.
    - It is not a complete sentence.
    - It has no trailing period.
    - The verb after the colon is lowercase.
    - Keep the entire first line under 76 characters.

5.  **Draft the Main Content**: After a blank line, write the body of the
    commit message in plain prose paragraphs. Do NOT create a commit — only
    display the message.
    - First paragraph: describe what the change does and why. Be specific and
      concrete.
    - Second paragraph: add context if needed, such as what the previous
      behavior was or how the new approach works.
    - Additional paragraphs: only if needed for extra context, benchmark data,
      or links.
    - Write in complete sentences with correct punctuation.
    - Do not use HTML, Markdown, or any other markup language.
    - Do not use markdown headers (##) in the body.
    - Do not use bullet lists unless absolutely necessary.
    - Do not add sections like "What", "Why", "How".
    - Write prose, not a template.

6.  **Reference Issues**: If the change fixes an issue, add a blank line
    followed by:
    - `Fixes https://github.com/googleapis/librarian/issues/<number>` if the
      change fully resolves the issue.
    - `For https://github.com/googleapis/librarian/issues/<number>` if the
      change is a partial step towards resolving the issue.
    - Do not use alternate aliases like Close or Resolves.

7.  **Present the Message**: Print the full commit message to the terminal so
    the user can review it before committing. Do NOT create a commit — only
    display the message.

## Example

```
feat(internal/librarian): add version subcommand

A version subcommand is added to librarian, which prints the current version of
the tool.

The version follows the versioning conventions described at
https://go.dev/ref/mod#versions.

Fixes https://github.com/googleapis/librarian/issues/238
```

## Principles

- **Follow the Convention**: Every commit message must match the format in
  CONTRIBUTING.md. Do not deviate.
- **Explain Why**: The body should explain the motivation for the change, not
  just restate what the diff shows.
- **Be Concise**: Keep the first line short. Keep the body focused.
- **No Markup**: The body is plain text only. No Markdown, no HTML.
