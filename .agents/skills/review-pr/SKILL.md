---
name: review-pr
description: Helps review your own pull request or local changes before opening a PR. Analyzes diffs against project conventions and checks out branches locally to ensure you can apply changes. Requires the `gh` and `git` CLI tools.
---

# Review PR

## Overview

Use this skill to review your own pull requests or to get a pre-review of your local changes before opening a PR. It helps you fetch the details, check out the PR locally, and analyze the diff against project standards.

Checking out the PR locally first ensures you can apply and test changes directly during the review.

## Workflow

### 1. Gather Context
Determine if you are reviewing an open PR or local changes.

**If reviewing an existing PR:**
- Run `gh pr checkout <pr-number>` FIRST to ensure you have the changes locally.
- Run `gh pr view --json title,body,author,headRefName,baseRefName` to understand the intent.
- Run `gh pr diff` or `git diff` to review the actual code changes.

**If reviewing local changes (Pre-PR Review):**
- Run `git status` to see what is being worked on.
- Run `git diff HEAD` to get the full diff of all tracked changes (or `git diff --staged` if reviewing a specific staged commit).
- Run `git log -n 1` to understand the context if a commit has already been made locally.

**For both:**
- Cross-reference the changes with project conventions in `CONTRIBUTING.md` and `doc/howwewritego.md` if necessary.

### 2. Analyze the Changes
Check the diff rigorously against the project's standards defined in `doc/howwewritego.md` and `CONTRIBUTING.md`:

**General & Go Idioms:**
- "Clear is better than clever." "Write simple, boring, readable code."
- **Naming:** Name length corresponds to scope size. Use singular form for package/folder names (e.g., `image/`, not `images/`).
- **Comments:** Do not explain standard Go concepts or comment on obvious logic.
- **Vertical Density:** Use line breaks only to signal a shift in logic. Avoid unnecessary vertical padding. Group related lines tightly.
- **Idiomatic Go:** Flag patterns that conflict with `doc/styleguide/go-code-review-comments.md` (e.g., use `errors.Is`/`errors.As`, early returns).

**Project-Specific Rules:**
- `internal/command`: Ensure `command.Run` is used for execution.
- `internal/testhelper`: Ensure existing utilities are used before creating new test tools.
- `internal/yaml`: Ensure this package is used instead of `gopkg.in/yaml.v3`.

**Testing & Documentation:**
- **Testing Quality:** Ensure tests are added or updated for new logic, following `doc/styleguide/go-test-comments.md`.
- **Markdown:** Ensure Markdown changes follow the Google Markdown Style Guide (`doc/styleguide/markdown-style-guide.md`).
- **Commits and PR Body:** If reviewing an existing PR, evaluate the PR body (which will become the commit message) using the conventions from the `/commit-message` skill. It must follow `CONTRIBUTING.md`, be written in complete sentences, and use the imperative mood.

**CRITICAL RULE:** NEVER automatically push commits to GitHub. Always create commits or apply changes locally. ALWAYS use the `ask_user` tool at the end to allow the user to select which specific changes they want applied locally.

## Output Format

Present the findings from the analysis clearly. **ONLY include findings that suggest or require specific changes.** Do NOT include complimentary comments or general positive feedback.

**Overall Assessment:** [Brief summary of the PR or local change quality]

**Findings:**
*(Indicate the file and line number for each change suggested)*

```markdown
1. **File/Line:** [Description of actionable change]
2. **File/Line:** [Description of actionable change]
```

After presenting the findings, use the `ask_user` tool. Provide a multiple-choice question (`type: 'choice'`, `multiSelect: true`) listing each of the numbered findings as an option. Ask the user: "Which of these changes would you like me to apply locally?"
