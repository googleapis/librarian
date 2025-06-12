# Project documentation

This folder containers public documentation for Librarian. While this
folder is public, its primary audience is Googlers who intend to use,
develop, or integrate with Librarian.

## Guidance

All documents should be in [GitHub flavored Markdown](https://github.github.com/gfm/).
There's no need to stick to a strict 80-column line limit, but wrap at
around that mark to make reviewing documentation PRs simpler.

As this repository is public, do not provide any details of Google
internal systems, or refer to such systems unless they have already
been described in public documents such as research papers and cases
studies. The following systems *have* been mentioned publicly, so can
be referred to within public documentation and can be mentioned, but
anything that requires detailed information should be in an internal document:

- go links
- Piper and google3
- Critique
- Kokoro
- Changelists (CLs)
- GAPIC

See the [Google Open Source Glossary](https://opensource.google/documentation/reference/glossary)
for more terms that can be referred to, but the above are the one most
likely to be relevant to Librarian.

Use go links and b/ links to refer to internal documents and bugs.
