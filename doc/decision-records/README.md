# Decision Records

## Summary

Decision records (otherwise known as architecture decision records or "ADRs") are records of important decisions made throughout a project. 

They capture the context and rationale for a decision at a point in time, which is invaluable for new joiners and for a system's evolution and maintenance.
By capturing how and why a decision was made, it can be re-assessed much more easily, as decision makers will know how important it was and the non-obvious details of why it was made, when building on or revisiting those decisions.

Googlers can check out go/adr for internal documentation on decision records.

## When to write one

Decision records are typically for documenting when a particular choice must be made among several options, whether it is a particular technology or a practice for the team to follow.
They are not well-suited for larger designs that may involve a variety of intertwined design choices.

The process is intended to be lightweight, and more focused on content and record-keeping with context than formality.

Some examples of good candidates:

1. Choosing a test framework
1. Choosing a programming language
1. Choosing a build system
1. Choosing a database
1. Choosing a vendor for capability X
1. Choosing a code review methodology or workflow (if specific, if a more involved SDLC, may be better fit for a design doc)

Some examples of unsuitable candidates:

1. Designing a build system
1. Designing an API
1. Code generation implementation

When unsure, ask a friend, or err on the side of writing a quick one by using Gemini to help structure your thoughts and context using the template linked below.

> [!NOTE]
> We intentionally use the phrase "decision records" and not "architecture decision records" as these are not limited to just "architecture."

## Structure

See [the template](doc/decision-records/000000-decision-record-template.md) for general structure.
The number of the ADR can be the PR number, or some monotonically increasing number relative to prior decision records.

Feel free to customize the format as best fit for the particular situation, as long as its key components are captured (like the context, considered options, consequences): the template is a guide, not a rule.
