# Contributor guide

Note: This repository is intended to support Google Cloud SDK in
generating and releasing client libraries for Google Cloud APIs.

We do not expect this repository to be particularly relevant to
developers outside Google, and we don't expect or seek external code
contributions. That doesn't mean it will never happen, but it would
be unusual.

## Team membership

In order to contribute to this repository, please ask a member of
[cloud-sdk-librarian-admin](https://github.com/orgs/googleapis/teams/cloud-sdk-librarian-admin)
to add you to
[cloud-sdk-librarian-team](https://github.com/orgs/googleapis/teams/cloud-sdk-librarian-team).

## Source control conventions

All submissions, including submissions by project members, require
review. We use GitHub pull requests (PRs) for this purpose. If you have a
specific team member in mind, please request their review directly,
otherwise just leave the default group reviewer.

### Multiple commits

When creating a PR, it's sometimes useful to organize the PR into
commits that are worth reviewing individually. If your PR contains
multiple commits, indicate in the PR description whether reviewers should
review one commit at a time, or just the final result. (If you've created a
slew of commits with experiments in, but only the result is relevant, you
don't want to waste reviewer time checking each experiment.)

Creating additional commits to address reviewer feedback is generally preferable to
amending and force-pushing the branch, as this allows the reviewer to quickly
check the changes you've made.

PR are always squashed and merged: if your PR contains multiple
commits when you come to merge it, please review/edit the resulting commit
message as part of merging.

### Commit messages

Please use [conventional commit messages](https://www.conventionalcommits.org).
For commits which affect go code, include the module that is changed in parentheses.
For example, a change adding a feature to the `internal/git` module might have a
commit message of:

```text
feat(internal/git): add interfaces for testability
```

Conventional commits are used by the release tooling within the Librarian CLI,
which is then used to create release notes. We intend to use the Librarian CLI
for its own release process, so using conventional commits will result in useful
release notes, and act as dogfooding for the conventional commit handling.

### Merging

PRs should generally be merged by the author of the PR rather than the reviewer
(partly so that the author can edit the resulting commit message as described
above). This is not a hard and fast rule, however: if a PR unblocks
other development and the author is not working, or if the PR is trivial and is
already in the form of a single commit with a suitable commit message, the reviewer
may merge the PR.

## Issues

When creating an issue, include a "type:"-prefixed label (e.g. "type: docs") rather
than using the GitHub "Type" field, so that the issue mirrors appropriately with
internal bug tracking.

Feel free to assign the issue to an individual if there is a natural owner.
Otherwise, leave it unassigned for a triage rotation to pick up.

To highlight the importance of an issue, use one of the "priority: p0" or
"priority: p1" labels. (Not specifying a priority is equivalent to "priority: p2".)
A p0 issue indicates that the team should drop other work to address the issue
as soon as possible. A p1 issue indicates that the issue should be addressed urgently,
ahead of other work - this will usually block the next release, for example.
