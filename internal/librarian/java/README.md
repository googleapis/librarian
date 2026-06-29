# Java Package Developer Guide

This guide describes how to handle changes in the `librarian` repository
that affect client library generation in `google-cloud-java`. It covers
two scenarios:

1.  **Breaking Changes:** Changes that cause code generation failures,
    compilation errors, or integration test failures in
    `google-cloud-java` (see
    [Handling Breaking Changes](#handling-breaking-changes-in-google-cloud-java)).
2.  **Non-Breaking Diffs:** Changes that introduce diffs in the
    generated code but do not break the build or tests (see
    [Handling Changes That Cause Generation Diffs](#handling-changes-that-cause-generation-diffs)).

## Handling Breaking Changes in `google-cloud-java`

If you are making changes in `librarian` that are expected to cause code
generation failure or other breakages in the `google-cloud-java` repository
(such as in the integration tests; see
[Example](#example-of-a-breaking-change)):

1. **Disable the Java Workflow:**
   Temporarily disable the Java integration workflow by modifying
   [java.yaml](/.github/workflows/java.yaml).
   You can do this by prepending `false && ` to the `if` condition of the
   `integration` job:

   ```yaml
   integration:
     runs-on: ubuntu-24.04
     if: false && github.event_name == 'push' && (github.ref == 'refs/heads/main')
   ```
2. **Add a TODO:**
   Add a `TODO` comment in [java.yaml](/.github/workflows/java.yaml) linking to
   the GitHub issue or pull request you are working on to track the reinstate
   task:

   ```yaml
   integration:
     runs-on: ubuntu-24.04
     # TODO(https://github.com/googleapis/librarian/issues/XXXX): Reinstate this job
     if: false && github.event_name == 'push' && (github.ref == 'refs/heads/main')
   ```
3. **Merge Librarian Changes:**
   Merge your changes into the `librarian` repository.
4. **Update `google-cloud-java`:**
   After the `librarian` changes are merged, update the `google-cloud-java`
   repository to use the pseudo-version containing your changes.

   You can update the version in `librarian.yaml` by running the following
   commands in the `google-cloud-java` repository:

   ```bash
   # Get the latest pseudo-version of librarian from main
   PSEUDO=$(GOPROXY=direct go list -m -f '{{.Version}}' github.com/googleapis/librarian@main)

   # Get the current librarian version used in the repo
   V=$(go run github.com/googleapis/librarian/cmd/librarian@latest config get version)

   # Update the version in librarian.yaml using the current tool version
   go run github.com/googleapis/librarian/cmd/librarian@${V} config set version $PSEUDO
   ```

   After updating the version, run `generate -all` to apply the changes.
5. **Reinstate the Java Workflow:**
   Once `google-cloud-java` is updated and working with the new changes, remove
   the `TODO` and reinstate the
   [java.yaml](/.github/workflows/java.yaml)
   workflow.

### Example of a Breaking Change

[PR #6432](https://github.com/googleapis/librarian/pull/6432) updated
`pom.xml` templates. It passed local tests but broke
`librarian generate --all` in `google-cloud-java`
([Issue #6446](https://github.com/googleapis/librarian/issues/6446)).
Because the integration test only runs in postsubmit, the failure
wasn't caught before merge, requiring a revert
([PR #6449](https://github.com/googleapis/librarian/pull/6449)). If
anticipated, the author should have disabled the workflow
beforehand.

## Handling Changes That Cause Generation Diffs

If you are making changes in `librarian` that do not cause generation failure in
`google-cloud-java` but will introduce a diff in the generated code:

1. **Librarian CI Stays Green:**
   The [java.yaml](/.github/workflows/java.yaml) integration check in the
   `librarian` repository will not fail on such changes.
2. **Submit `google-cloud-java` PR:**
   It is good practice to immediately open a pull request in the
   `google-cloud-java` repository. This PR should update the `librarian`
   dependency to **the new pseudo-version** containing your changes (using the
   commands described in [Handling Breaking Changes](#handling-breaking-changes-in-google-cloud-java))
   and run `generate -all` to apply the generated diff.
3. **Prevent Weekly Update Diffs:**
   Proactively applying these diffs prevents them from being introduced
   abruptly during the weekly automated `librarian` updates.
