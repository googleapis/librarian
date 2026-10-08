---
name: walkthrough
description: Runs the interactive Librarian walkthroughs locally and answers questions about them. Use this skill when asked to open, run or serve the walkthroughs, the architecture tour or the guides, or when asked a question about how Librarian works that a walkthrough covers.
---

# Walkthrough

## Overview

`doc/walkthroughs/*.md` are step-by-step guides to Librarian. The
`tool/cmd/walkthrough` command compiles them into an interactive site whose
code excerpts and tables are read from the tree at build time. The site is
not published; it runs on the developer's machine. This skill starts it and
answers the questions a reader asks along the way.

## Starting the site

Run the server as a background process from the repository root and tell the
user the URL:

```sh
go run ./tool/cmd/walkthrough -serve 127.0.0.1:8080
```

The address must be loopback. Pages are rebuilt on every load, so the user can
edit a walkthrough or the code and refresh. If the port is taken, pick another.

Each step has an **Ask about this step** panel. With the command above it
copies a prompt to the clipboard, which the user pastes to you. To route
questions to a command instead, pass `-ask`. The command runs in the
repository root through `sh -c`, reads the prompt on stdin and must print the
answer on stdout. Examples (check the tool is installed first):

```sh
# Any CLI that answers a prompt from stdin.
go run ./tool/cmd/walkthrough -serve 127.0.0.1:8080 -ask gemini
# In an editor with agentapi: open the question as a new conversation.
go run ./tool/cmd/walkthrough -serve 127.0.0.1:8080 \
  -ask 'agentapi new-conversation --title="Walkthrough question" "$(cat)"'
```

With `agentapi` the answer appears in the editor, not in the page; the page
shows the conversation id.

## Answering questions

Every prompt from the panel names the page, the step, the commit and the files
excerpted in that step, followed by the question. Answer from those sources:

1. Read the step in `doc/walkthroughs/<page>.md` and the files it excerpts
   at the stated commit. Prefer the code over the prose when they disagree,
   and say so.
2. Cite files and line ranges. Keep answers short; the reader is mid-tour.
3. If the question is about a different step or page, point to it by name.
4. If the prose is wrong or stale, offer to fix the markdown. After editing,
   run `go run ./tool/cmd/walkthrough -check`; it fails if an excerpt anchor
   no longer resolves. See `doc/walkthroughs/README.md` for the format.

## Verifying changes

```sh
go run ./tool/cmd/walkthrough -check
go test ./tool/cmd/walkthrough/
```

Do not add anything to the build that depends on time, network or a language
model: the site must be a pure function of the commit.
