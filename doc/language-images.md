# Language Images

[`cmd/librarian/Dockerfile`](/cmd/librarian/Dockerfile) builds one image per
language. Each image has `librarian`, the language runtimes, and the
prerequisites that `librarian install` needs.

## Build an image

You need Docker and access to `marketplace.gcr.io`:

```sh
gcloud auth login
gcloud auth configure-docker marketplace.gcr.io
```

From the repository root, run:

```sh
docker build --target <language> -t librarian-<language> -f cmd/librarian/Dockerfile .
```

`<language>` is one of `go`, `java`, `nodejs`, `php`, `python`, `ruby`, `rust`
or `swift`.

## Update a version

These `ARG`s in the Dockerfile track a file in a language repository. When that
file changes, update the `ARG` to match.

| `ARG` | Repository | File | What to look for |
| ----- | ---------- | ---- | ---------------- |
| `GO_RUNTIME_VERSION` | google-cloud-go | [go.mod](https://github.com/googleapis/google-cloud-go/blob/main/go.mod) | `go` directive |
| `GO_GOIMPORTS_VERSION` | google-cloud-go | [librarian.yaml](https://github.com/googleapis/google-cloud-go/blob/main/librarian.yaml) | `version` of `golang.org/x/tools/cmd/goimports` in `tools.go`, without the leading `v` |
| `RUST_VERSION` | google-cloud-rust | [rust-toolchain.toml](https://github.com/googleapis/google-cloud-rust/blob/main/rust-toolchain.toml) | `channel` (minor version) |
| `TAPLO_CLI_VERSION`, `CARGO_*_VERSION` | google-cloud-rust | [librarian.yaml](https://github.com/googleapis/google-cloud-rust/blob/main/librarian.yaml) | `version` of each tool in `tools.cargo` |
| `PROTOC_VERSION` | google-cloud-rust | [librarian.yaml](https://github.com/googleapis/google-cloud-rust/blob/main/librarian.yaml) | `tools.protoc.version` |
| `PYTHON_VERSION` | google-cloud-python | [packages/gapic-generator/setup.py](https://github.com/googleapis/google-cloud-python/blob/main/packages/gapic-generator/setup.py) | `python_requires` (minimum) |
| `PYTHON_PROTOC_VERSION` | google-cloud-python | [librarian.yaml](https://github.com/googleapis/google-cloud-python/blob/main/librarian.yaml) | `tools.protoc.version` |
| `JAVA_VERSION` | google-cloud-java | [librarian_generation_check.yaml](https://github.com/googleapis/google-cloud-java/blob/main/.github/workflows/librarian_generation_check.yaml) | `java-version` of the `actions/setup-java` step (major version) |
| `NODEJS_NODE_VERSION` | google-cloud-node | [package.json](https://github.com/googleapis/google-cloud-node/blob/main/package.json) | `engines.node` (minimum) |
| `NODEJS_PNPM_VERSION` | google-cloud-node | [package.json](https://github.com/googleapis/google-cloud-node/blob/main/package.json) | version after `pnpm@` in `packageManager` |
| `PHP_VERSION` | google-cloud-php | [composer.json](https://github.com/googleapis/google-cloud-php/blob/main/composer.json) | `require.php` (range) |
| `RUBY_VERSION` | google-cloud-ruby | [ci.yml](https://github.com/googleapis/google-cloud-ruby/blob/main/.github/workflows/ci.yml) | `ruby` values in the CI matrix (range) |
| `SWIFT_VERSION` | google-cloud-swift | [Package.swift](https://github.com/googleapis/google-cloud-swift/blob/main/Package.swift) | `swift-tools-version` (minimum) |

For a minimum, a range, a major version or a minor version, set the `ARG` to
any full release version that satisfies it.

`GO_VERSION` matches Librarian's own [`go.mod`](/go.mod). The other `ARG`s are
not set from a file in a language repository.

To update a version:

1.  Change the `ARG`, and its `*_CHECKSUM` if it has one. For Java, also change
    `JAVA_BUILD`. The comment at the top of the Dockerfile says where each
    checksum comes from.
2.  Build every image that uses the `ARG`. The last step of each image runs the
    version command of each runtime, so the build fails if one is missing.
3.  Optionally, run `librarian install` in the image from a checkout of the
    language repository:

    ```sh
    docker run --rm --user "$(id -u):$(id -g)" \
      -v "$(pwd)":/workspace -w /workspace \
      librarian-<language> install
    ```
