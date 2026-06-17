# Java Package Developer Guide

This guide describes how to handle changes in the `librarian` repository that
are expected to temporarily break client library generation in
`google-cloud-java`.

## Handling Breaking Changes in `google-cloud-java`

If you are making changes in `librarian` that are expected to cause code
generation failure or other breakages in the `google-cloud-java` repository
(such as in the integration tests):

1. **Disable the Java Workflow:**
   Temporarily disable the Java integration workflow by modifying
   [java.yaml](https://github.com/googleapis/librarian/blob/main/.github/workflows/java.yaml).
   You can disable the jobs or the trigger (e.g. by adding `if: false` or
   commenting it out).
2. **Add a TODO:**
   Add a `TODO` comment in
   [java.yaml](https://github.com/googleapis/librarian/blob/main/.github/workflows/java.yaml)
   linking to the GitHub issue or pull request you are working on to track the
   reinstate task.
3. **Merge Librarian Changes:**
   Merge your changes into the `librarian` repository.
4. **Update `google-cloud-java`:**
   After the `librarian` changes are merged, update the `google-cloud-java`
   repository to use the pseudo-version containing your changes, and update the
   librarian.yaml accordingly and run `generate -all`.
5. **Reinstate the Java Workflow:**
   Once `google-cloud-java` is updated and working with the new changes, remove
   the `TODO` and reinstate the
   [java.yaml](file:///usr/local/google/home/zhumin/repos/testjava2/librarian/.github/workflows/java.yaml)
   workflow.
