# How to Generate Client Libraries with Librarian
The configuration for librarian is stored in `librarian.yaml` at the root of the repository.


## Running the Generator with Local Changes
If you want to test any changes to local tools:

1.  Follow instructions in your languages’ playbook section for
    prerequisites (note: this list is not exhaustive).
    *   [Rust](https://docs.google.com/document/d/1moP7zq3Qy7xqjdKSpTJhHbhpX7dSG9Wo38f9nlEpcLQ/edit?resourcekey=0-p1YPSwMTMGFYRgr4jegcOw&tab=t.0#heading=h.pmn6qeywaa4u)
    *   [Go](https://docs.google.com/document/d/1moP7zq3Qy7xqjdKSpTJhHbhpX7dSG9Wo38f9nlEpcLQ/edit?resourcekey=0-p1YPSwMTMGFYRgr4jegcOw&tab=t.0#heading=h.dsch23ap4l2b)
    *   [Java](https://docs.google.com/document/d/1moP7zq3Qy7xqjdKSpTJhHbhpX7dSG9Wo38f9nlEpcLQ/edit?resourcekey=0-p1YPSwMTMGFYRgr4jegcOw&tab=t.gxrue4sew3io#heading=h.c6mszfhtw541)
2. **Modify `librarian.yaml` to point to new or local versions.**
    - For example, if you want to bump the protoc-gen-java_grpc, you can change
      its `version` property.
    - If you want to test a version of the go generation, you can change its
      `version` property to your branch or commit-hash.
    - For Java, librarian.yaml is configured to build the gapic generator from
      local path. So you don’t need modifications to test local changes.
3.  **Re-install the local tools:**
    Run the following command from the root of the `google-cloud-java`
    repository to compile your local generator changes and update the wrappers:
    ```sh
    # Retrieve the configured librarian version
    V=$(go run github.com/googleapis/librarian/cmd/librarian@latest config get version)
    # Re-install the local tools
    go run github.com/googleapis/librarian/cmd/librarian@${V} install
    ```
4.  **Regenerate the client library:**

    To regenerate all libraries:
    ```sh
    go run github.com/googleapis/librarian/cmd/librarian@${V} generate --all
    ```
    Or to regenerate a single library (e.g., `accessapproval`):
    ```sh
    go run github.com/googleapis/librarian/cmd/librarian@${V} generate accessapproval
    ```

