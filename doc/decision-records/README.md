# Decision Records

## Summary

Decision records (otherwise known as architecture decision records or "ADRs") are records of important decisions made throughout a project. 

They capture the context and rationale for a decision at a point in time, which is invaluable for new joiners and for evolution of a system.
By capturing how and why a decision was made, it can be re-assessed much more easily, as decision makers will know how important it was and the non-obvious details of why it was made, when building on or revisiting those decisions.

## When to write one

Decision records are typically for documenting when a particular choice must be made among several options, whether it is a particular technology or a practice for the team to follow.
They are not well-suited for larger designs that may involve a variety of intertwined design choices.

Some examples of good candidates:

1. Choosing a test framework
2. Choosing a programming language
3. Choosing a build system
4. Choosing a database
5. Choosing a vendor for capability X
6. Choosing a code review methodology or workflow (if specific, if a more involved SDLC, may be better fit for a design doc)

Some examples of unsuitable candidates:

1. Designing a build system
2. Designing an API
3. Code generation implementation

## Structure

See 0000-decision-record-template.md for a template to use.
Feel free to customize the format as best fit for the particular situation, as long as its key components are captured (like the context, considered options, consequences): the template is a guide, not a rule.
