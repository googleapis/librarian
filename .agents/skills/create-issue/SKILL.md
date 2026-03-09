---
name: create-issue
description: Helps create a GitHub issue that strictly follows the conventions in CONTRIBUTING.md. Use this skill when asked to create, draft, or open a GitHub issue or bug report.
---

# Create Issue

## Overview

Use this skill when asked to write or draft a GitHub issue. It ensures issues follow the conventions established in `CONTRIBUTING.md` and matches the user's specific architectural and stylistic preferences.

## Instructions

When the user asks you to create a GitHub issue, follow these steps to draft the issue title, body, assignee, labels, and milestone. **You MUST ALWAYS ask for explicit confirmation before creating the issue using the `gh` CLI.** You should output the drafted issue to the terminal so the user can review it first.

### 1. Issue Title

All issues must have a path prefix indicating the relevant domain, followed by a lowercase summary.

**Path Prefixes:**
- Use package names or tool paths (e.g., `librarian:`, `cli:`, `tool/cmd/migrate:`).
- For language-specific code under `internal/librarian/<language>`, use the lowercase language name (e.g., `java:`, `dotnet:`).
- For systemic proposals, chain prefixes (e.g., `proposal: config:`).
- Aside from proper nouns, the rest of the issue title **must use lowercase**.

### 2. Issue Body Structure

Match the user's specific writing style based on the type of issue:

#### Implementation/Tracking Issues
- **Opening:** Start with "This issue tracks the implementation of..." or "This issue tracks integrating..."
- **Architecture Awareness:** Explicitly call out the exact packages and files that need to be created or modified (e.g., "The implementation in `internal/librarian/dotnet/publish.go` (and wired into `internal/librarian/publish.go`)").
- **Action-Oriented Steps:** Use a numbered list to break down the required logic into precise, ordered steps.

#### Systemic Proposals
- Use a cross-language comparative structure:
    - **Bolded categories** for each language (e.g., **Python**, **Go**).
    - A brief 1-2 sentence explanation of the current state.
    - A markdown code block demonstrating the exact file output or configuration.
- **Conclusion:** A single, decisive concluding sentence that frames the evaluation goal without jumping to a solution.

#### Migration Tasks
- Use a two-sentence format:
    1. An imperative command to update a specific tool.
    2. A reference to an internal design document (e.g., "See go/sdk-librarian-nodejs for details.").

### 3. Assignee & Labels

- **Assignee:** Default to assigning if there's a clear owner; otherwise, leave unassigned.
- **Labels:** 
    - Use `critical` for immediate attention (e.g., broken build on main).
    - Use `needs fix soon` for high-priority items.

### 4. Milestone

- Run `gh milestone list` to retrieve available milestones.
- Ask the user to choose one, or suggest "Unplanned" if not on the roadmap.

## Output Format

Present the drafted issue clearly:

**Title:** `prefix: issue description`
**Assignee:** `@username` (or Unassigned)
**Labels:** `critical`, `needs fix soon`, or None
**Milestone:** `milestone_name`

**Body:**
```markdown
[Issue body here]
```

Ask the user if they would like to refine the issue or create it using `gh issue create`.