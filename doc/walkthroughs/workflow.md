---
title: The Librarian workflow, command by command
summary: Follow the commands a client-library repository runs over the life of a library, from onboarding an API to tagging a release, and see for each one where it runs and whose code does the work.
audience: For engineers who want to understand the overall workflow
order: 1
nav: Workflow
---

Librarian is one Go binary that a *language repository* (such as
`google-cloud-go` or `google-cloud-rust`) runs against its own checkout.
Everything it does is driven by the `librarian.yaml` at the root of that
repository. This walkthrough follows the commands in the order a repository
typically uses them and answers, for each step:

- **Where does it run?** A developer machine, a GitHub Actions workflow in the
  language repository, or release automation.
- **Whose code does the work?** Librarian's own Go code, an external tool
  that Librarian installs (such as `protoc` or a generator plugin), or files
  and scripts that live in the language repository.

```flow
title: The lifecycle of a library and where each command runs
lanes:
  - id: user
    label: Developer
    kind: user
  - id: ci
    label: Automation in the language repository
    kind: ci
  - id: lib
    label: librarian binary
    kind: lib
  - id: ext
    label: External tools and registries
    kind: ext
nodes:
  - id: install
    lane: lib
    label: install
    step: 3
  - id: add
    lane: user
    label: add <api>
    step: 4
  - id: update
    lane: ci
    label: update sources.googleapis
    step: 5
  - id: generate
    lane: lib
    label: generate --all
    step: 6
  - id: protoc
    lane: ext
    label: protoc, generators, formatters
    step: 8
  - id: pr
    lane: ci
    label: regeneration pull request
    step: 13
  - id: bump
    lane: lib
    label: bump --all
    step: 10
  - id: publish
    lane: ext
    label: publish to crates.io, pub.dev, ...
    step: 11
  - id: tag
    lane: lib
    label: tag
    step: 12
edges:
  - install -> add
  - add -> generate | first generation
  - update -> generate | new pins
  - generate -> protoc
  - protoc -> pr | generated files
  - pr -> bump | after merge
  - bump -> publish
  - publish -> tag
```

Color legend used throughout: blue is automation in the language repository,
green is Librarian's own code, purple is code or configuration owned by the
language repository, orange is an external tool or service.

## Librarian always runs inside the language repository
<!-- step runs="Developer machine or GitHub Actions in the language repository" code="librarian" -->

There is no server. Each command reads `librarian.yaml` from the current
working directory; there is no `--config` flag and no upward search, so the
repository root is wherever you run the command.

```excerpt
file: internal/config/config.go
start: "LibrarianYAML = "
end: "+1"
caption: The configuration file name is a relative path, so the working directory is the repository.
```

The CLI is built with `urfave/cli`. Eleven commands are registered; `bump`,
`publish` and `tag` are hidden from `--help` because they are meant for
release automation rather than people.

```excerpt
file: internal/librarian/librarian.go
start: "Commands: []*cli.Command{"
end: "},"
caption: The complete command registry. There is no separate clean, docindex or release-please command; those are helpers called by add and generate.
```

Language repositories get the binary in one of two ways: developers run
`go run github.com/googleapis/librarian/cmd/librarian@<version>`, and GitHub
workflows use the composite action at the root of this repository. The action
installs Go and `protoc`, then installs the Librarian version pinned in the
consumer's `librarian.yaml`.

```excerpt
file: action.yaml
start: "- name: Install librarian"
end: "+10"
caption: The action reads the version field of the consumer's librarian.yaml, so the pin lives with the consumer.
```

## librarian.yaml is the only state Librarian keeps
<!-- step runs="Language repository" code="language repository configuration" -->

Every command starts from the same data model in `internal/config`, a pure
1:1 mapping of `librarian.yaml`. The top-level sections are:

| Section | What it holds | Who writes it |
|---|---|---|
| `language`, `version` | The repository's language and the pinned Librarian version | `librarian update version`, people |
| `sources` | Pinned commits and checksums of `googleapis` and other input repositories | `librarian update sources.<name>` |
| `tools` | Versions of `protoc`, generator plugins and package-manager tools to install | people |
| `default` | Repository-wide defaults: output directory, tag format, per-language settings | people |
| `libraries` | One entry per library: name, APIs, version, output, keep list, per-language options | `librarian add`, `bump`, people |

```excerpt
file: internal/config/config.go
start: "type Config struct {"
caption: The root of the configuration. Each language has its own optional block in Default and Library.
```

Librarian writes this file through `tidy`, which validates and canonicalizes
it, so hand edits and generated edits converge on the same shape. There is no
other state file: no database, no `.librarian/state`. Release automation can
therefore reconstruct what was released by diffing `librarian.yaml` between
commits, which is exactly what `tag` does later.

## `install`: provision the tools a language needs
<!-- step runs="GitHub Actions in the language repository, before generate" code="librarian + package managers" -->

`librarian install` reads the `tools` section and installs each tool into
`$LIBRARIAN_BIN` (by default under the user cache directory). It never writes
into the repository. The first step is shared by all languages: download the
pinned `protoc` release.

```excerpt
file: internal/librarian/install.go
start: "switch lang {"
end: "+23"
caption: After protoc, each language has its own installer; dart's is a no-op because the dart SDK is expected on PATH.
```

What the per-language installers run:

| Language | Installer runs | Installs |
|---|---|---|
| go | `go install <tool>@<version>` with `GOBIN=$LIBRARIAN_BIN/go_tools` | `protoc-gen-go`, `protoc-gen-go-grpc`, `protoc-gen-go_gapic` |
| java | `mvn` | gapic-generator-java and plugins |
| nodejs | `pnpm` | `gapic-generator-typescript` and helpers |
| php | `composer`, `pip`, `pnpm` | gapic-generator-php and post-processors |
| python | `pip` | gapic-generator-python, synthtool, pandoc; also extracts embedded templates |
| ruby | `gem` | gapic-generator-ruby |
| rust | `cargo install --locked` | extra cargo plugins such as `cargo-workspaces`; `cargo` and `taplo` themselves must be on PATH |
| swift | `git clone` + `swift build` | `swift-format` and friends |
| dart | nothing | `dart` must be on PATH |

Every one of these is an external process spawned through `internal/command`,
which merges `$LIBRARIAN_BIN` into `PATH` so later steps find the tools.

```excerpt
file: internal/command/command.go
start: "func buildCmd("
end: "+30"
caption: All external processes go through buildCmd; with --verbose the full command line is echoed, which is the quickest way to see what a language actually runs.
```

## `add <api>`: onboard an API into a library
<!-- step runs="Developer machine (occasionally an onboarding bot)" code="librarian; edits language-repository files" -->

`librarian add google/cloud/secretmanager/v1` registers an API path from
`googleapis` as a new library, or adds it to an existing one. It needs the
`googleapis` source to validate the path and, for some languages, to read the
service configuration.

```excerpt
file: internal/librarian/add.go
start: "func runAdd("
end: "+30"
caption: Fetch googleapis, validate the API path, add or update the library, resolve dependencies, sync release-please files, then tidy.
```

The language-specific part is how a library name is derived and what else a
new library needs. Each language package exposes an `Add` function:

```excerpt
file: internal/librarian/add.go
start: "func addNewLibrary("
end: "+59"
caption: The per-language Add dispatch. Dart has no case and gets a bare library entry.
```

For Go, Node.js, Python and Ruby, `add` also edits the repository's
release-please manifest and configuration files when they exist. Those JSON
files belong to the language repository; Librarian only knows their names and
the entries it must add.

```excerpt
file: internal/librarian/release_please.go
start: "func releasePleaseFiles("
end: "+28"
caption: Which release-please files each language keeps at its repository root.
```

`add` writes nothing until the end, when `RunTidyOnConfig` validates and
saves `librarian.yaml`. Running it again for the same API fails with "already
exists".

## `update sources.googleapis`: move the input pins
<!-- step runs="Scheduled automation in the language repository" code="librarian" -->

Generated code is a function of the pinned `googleapis` commit. `update`
resolves the branch head through the GitHub API, streams the matching tarball
to compute its SHA-256, and writes both values back into `librarian.yaml`.
Nothing else changes; the next `generate --all` picks up the new protos.

```excerpt
file: internal/librarian/update.go
start: "sourceRepos    = map[string]fetch.RepoRef{"
end: "+6"
caption: The updatable sources and the branch each one tracks.
```

```excerpt
file: internal/librarian/config_value.go
start: "func fetchSourceCommitAndChecksum("
end: "+12"
caption: A pin is a commit plus the checksum of GitHub's tarball for it.
```

`update version` is the same idea for Librarian itself: it asks the Go module
proxy for the latest `github.com/googleapis/librarian` and records it, which
is what the GitHub Action later installs.

## `generate`: fetch inputs and decide what to build
<!-- step runs="GitHub Actions in the language repository; developers locally" code="librarian" -->

`librarian generate <library>` or `generate --all` is the heart of the
workflow. It begins by materializing every configured source. Sources are not
git clones: each pin is downloaded once as a tarball, verified against its
checksum, and extracted under the cache directory keyed by commit.

```excerpt
file: internal/fetch/fetch.go
start: "func Repo(ctx context.Context, repo, commit, expectedSHA256 string)"
end: "+50"
caption: Cache hit on the extracted directory, then on a verified tarball, then download.
```

A `dir:` override in `sources` short-circuits all of that, which is how tests
and local experiments point at a working copy of `googleapis`.

Next, each selected library is normalized by `applyDefaults`: the API path
and output directory are derived from the name where the language allows it,
repository-wide defaults are merged in, and a `preview` variant is resolved
into a separate library to generate.

```excerpt
file: internal/librarian/library.go
start: "func applyDefaults("
end: "+29"
caption: Every library passes through this once per run; the result is never written back to librarian.yaml.
```

## `generate`: clean, generate, format
<!-- step runs="GitHub Actions in the language repository" code="librarian, dispatching to a language package" -->

With the libraries known, `generate` runs three phases. First it deletes the
previous output, keeping only the files listed in each library's `keep`
list, so stale generated files disappear. Then it generates and formats.
Each phase is a `switch` on `cfg.Language`.

```excerpt
file: internal/librarian/generate.go
start: "func runGenerate("
end: "+48"
caption: Selection, preview resolution, clean, then generate.
```

```excerpt
file: internal/librarian/generate.go
start: "func generateLibraries("
end: "+20"
highlight: "switch cfg.Language"
caption: Each case decides its own concurrency and whether formatting is per library or repository-wide.
```

Notable differences between the cases:

- **Java** generates sequentially, formats everything with
  `google-java-format` in one go, and then runs `java.PostGenerate`, which
  rewrites the repository's root `pom.xml` and BOM.
- **Rust** generates and formats in parallel, writes a documentation index,
  and finishes with `cargo update --workspace`.
- **Python** formats inside its own `Generate`, running the synthtool
  post-processor and `nox -s format`.
- **Node.js** has no separate format step.

The language packages under `internal/librarian/<lang>` are all Librarian
code, but what they *run* differs a lot, which is the next step.

## `generate`: two kinds of generators
<!-- step runs="GitHub Actions in the language repository" code="librarian (sidekick) or external generator tools" -->

Three languages are generated in-process by Librarian's own template engine,
*sidekick*. The other six shell out to the established GAPIC generators,
which Librarian installs and drives through `protoc`.

| Language | Generator | How it is invoked | Language-repository files consumed |
|---|---|---|---|
| rust | sidekick (in-process) | `protoc` for descriptors, templates in the binary | workspace `Cargo.toml`, `keep` lists |
| swift | sidekick (in-process) | same, plus `protoc --swift_out` for some modules | `Package.swift` directories |
| dart | sidekick (in-process) | same | `pubspec.yaml` |
| go | `protoc-gen-go_gapic` | `protoc` with plugins from `$LIBRARIAN_BIN`, then `go mod tidy` | `go.mod`, `internal/version.go` |
| java | gapic-generator-java | `protoc` with Maven-installed plugins | `pom.xml` tree |
| nodejs | gapic-generator-typescript | the generator CLI run in the googleapis directory | `packages/<lib>/librarian.js` if present |
| php | gapic-generator-php | `protoc` twice, then a Python post-processor | common resources from `default.php` |
| python | gapic-generator-python | `protoc`, synthtool post-processor, `nox` | `gapic_version.py`, samples |
| ruby | gapic-generator-ruby | `protoc` with gem plugins, then `toys` tasks | `toys_tasks` declared per library |

A typical external invocation, assembled for Go:

```excerpt
file: internal/librarian/golang/generate.go
start: "func generateAPI("
end: "+31"
caption: Librarian builds the protoc command line; the generator plugin does the actual code generation.
```

The important consequence: for the six external languages, generator bugs
and features live upstream in the generator projects, and Librarian's role is
orchestration, versions and post-processing. For rust, swift and dart the
whole generator is in this repository. The
[generate and sidekick walkthrough](generate-and-sidekick.html) follows that
path in detail.

The only pieces that come from the language repository during generation are
configuration and a few hooks: `keep` lists, per-library options in
`librarian.yaml`, Node.js's optional `librarian.js` post-processing script,
Ruby's `toys` tasks, and the Java POM tree.

## `tidy`: keep the configuration canonical
<!-- step runs="Developer machine; implicitly at the end of add and bump" code="librarian" -->

`tidy` validates `librarian.yaml` and strips anything that can be derived:
default outputs, the default `googleapis` root, single API paths that match
the library name. It is the only place validation happens, and the only path
through which `add` and `bump` persist their changes.

```excerpt
file: internal/librarian/tidy.go
start: "func RunTidyOnConfig("
end: "+18"
caption: Validate, normalize, sort, write.
```

Unlike the other dispatch points, the per-language tidiers are a map rather
than a switch:

```excerpt
file: internal/librarian/tidy.go
start: "var languageTidiers = "
end: "+8"
```

## `bump`: compute the next versions
<!-- step runs="Release automation in the language repository" code="librarian + git; edits language-repository version files" -->

After regeneration pull requests merge, release automation runs the hidden
`bump --all`. It requires a clean git tree and then takes one of three paths
depending on the language:

```excerpt
file: internal/librarian/bump.go
start: "func runBump("
end: "+38"
caption: Rust and Swift use a repository-wide last tag; Dart has its own dependency-ordered flow; everyone else is tag-per-library.
```

In the tag-per-library flow, a library is bumped when files under its output
directory changed since the git tag of its current version. The tag name
comes from `default.tag_format`.

```excerpt
file: internal/librarian/bump.go
start: "func findLibrariesToBump("
end: "+29"
caption: Per-library git tags must exist locally; the diff since the tag decides what changed.
```

The next version is always a minor bump unless `--version` overrides it;
Librarian does not analyze conventional commits. The language packages then
edit the files that carry the version in that ecosystem: `internal/version.go`
and snippet metadata for Go, `gapic_version.py` for Python, `Cargo.toml` and
`README.md` for Rust, `pubspec.yaml` and `CHANGELOG.md` for Dart.

```excerpt
file: internal/librarian/bump.go
start: "func bumpLibrary("
end: "+19"
```

Java, Node.js, PHP and Ruby do not use `bump`; release-please owns their
versions, which is why `add` maintained its manifests earlier.

## `publish`: push packages to registries
<!-- step runs="Release automation in the language repository" code="librarian + cargo, dart, swift, git" -->

`publish` exists for Rust, Dart and Swift only. Each implementation checks
that the local branch matches the upstream branch point, works out which
packages changed since the last release, and then drives the ecosystem's
publishing tool.

```excerpt
file: internal/librarian/publish.go
start: "if cfg.Language == config.LanguageRust {"
end: "+9"
caption: An if-chain rather than a switch; other languages get an error.
```

For Rust that means `cargo workspaces plan`, `cargo semver-checks` and
finally `cargo workspaces publish`; `--dry-run` swaps in `cargo publish
--dry-run`.

```excerpt
file: internal/librarian/rust/publish.go
start: "args = []string{\"workspaces\", \"publish\""
end: "+6"
```

Dart publishes packages in dependency order with `dart pub publish`; Swift
splits each package into its own repository with `git subtree` and pushes a
branch and tag per package.

## `tag`: mark the release commit
<!-- step runs="Release automation in the language repository" code="librarian + git" -->

`tag` runs after a successful publish. It does not read the working copy's
`librarian.yaml`; it reads the file from git history, finds the newest commit
whose configuration released at least one library (a version appeared or
increased compared with the parent commit), and creates one local tag per
released library.

```excerpt
file: internal/librarian/tag.go
start: "func tag(ctx context.Context, releaseCommit string) error {"
end: "+53"
caption: Release discovery from history; tags are local, pushing is left to the caller.
```

```excerpt
file: internal/librarian/bump.go
start: "func findReleasedLibraries("
end: "+32"
caption: The before/after diff that defines "released".
```

Because `bump` later looks these tags up to decide what changed, the intended
order is strict: bump, commit, publish, tag.

## How language repositories wire it together
<!-- step runs="Language repository automation" code="language repository workflows" -->

Librarian never opens pull requests, never runs on a schedule and never
pushes anything except, for Swift, the split package repositories. Every
language repository therefore supplies its own automation around the binary,
and a survey of the nine repositories shows three patterns.

**Pattern 1: the repository drives the whole loop.** `google-cloud-java` has
a daily workflow that reads the pinned version with `config get version`,
runs `update sources.googleapis` and `tidy`, and if `librarian.yaml` changed
goes on to `install` and `generate --all` before opening a pull request with
a bot token.

```excerpt
file: .github/workflows/update_librarian_googleapis.yaml
repo: googleapis/google-cloud-java
ref: 1d92aa592c98a9cc8cb0d4465a95a3aa17cb223d
line: 77
caption: The only in-repository cron that runs update, tidy, install and generate end to end.
code: |
  version=$(go run github.com/googleapis/librarian/cmd/librarian@latest config get version)
  echo "version=${version}" >> $GITHUB_OUTPUT
  - name: Run librarian update & tidy
    run: |
      go run "github.com/googleapis/librarian/cmd/librarian@${STEPS_LIBRARIAN_OUTPUTS_VERSION}" update sources.googleapis
      go run "github.com/googleapis/librarian/cmd/librarian@${STEPS_LIBRARIAN_OUTPUTS_VERSION}" tidy
```

**Pattern 2: the repository only checks for drift.** `google-cloud-go`,
`google-cloud-python`, `google-cloud-node`, `google-cloud-php`,
`google-cloud-dart` and `google-cloud-swift` run `generate --all` (or a
subset of touched packages) on pushes and pull requests and fail if
`git diff` is not empty. `google-cloud-rust` does the same from Cloud Build
rather than GitHub Actions. The regeneration pull requests themselves arrive
from automation that is not in the public repositories; the Go repository's
`apidiff` workflow simply recognizes branches named `librarian-<timestamp>`.

```excerpt
file: .gcb/scripts/install-librarian.sh
repo: googleapis/google-cloud-rust
ref: 90c8b17a48575f9d199cc89dd74463650ff51938
line: 21
caption: Cloud Build reads the pin straight out of librarian.yaml to avoid downloading Librarian twice.
code: |
  # Normally we recommend `librarian config get version` but that requires
  # downloading two copies of librarian.
  version=$(sed -n 's/^version: *//p' /workspace/librarian.yaml)
  # Make multiple download attempts to avoid download-induced flakes.
  go install github.com/googleapis/librarian/cmd/librarian@${version} ||
  (sleep 5 && go install github.com/googleapis/librarian/cmd/librarian@${version}) ||
  (sleep 10 && go install github.com/googleapis/librarian/cmd/librarian@${version})
```

**Pattern 3: nothing in the repository.** `google-cloud-ruby` has no
workflow that runs Librarian at all; its `.toys` tooling and release-please
handle releases, and regeneration pull requests come from outside.

How the version is pinned also varies, and it is worth knowing because the
`uses: googleapis/librarian@<ref>` line in a workflow does **not** pin the
CLI; the composite action always installs whatever `librarian.yaml: version`
says. Node pins the action at a v0.31.1 commit while its `librarian.yaml`
says v0.47.0, and both are correct.

```excerpt
file: .github/workflows/generation_check.yaml
repo: googleapis/google-cloud-go
ref: 00f8c4132a1f056a8d8b03ea7624ea2b61f7ad10
line: 26
caption: The action reference and the CLI version are independent.
code: |
  # Version of librarian is pulled from librarian.yaml,
  # not the version of the action.
  - uses: googleapis/librarian@main # zizmor: ignore[unpinned-uses]
    if: steps.changes.outputs.librarian == 'true' || (github.event_name == 'push' && github.ref == 'refs/heads/main')
    with:
      protoc-version: '33.2'
  - name: Install librarian tools
    run: librarian install
  - name: Run librarian generate --all
    run: |
      librarian generate --all
      if [ -n "$(git status --porcelain)" ]; then
```

| Repository | Regeneration check | Pin style | Release flow |
|---|---|---|---|
| google-cloud-go | GitHub Actions, push and PRs touching `librarian.yaml` | action reads `librarian.yaml` | release-please app, three config pairs |
| google-cloud-java | daily cron does the full loop; post-merge drift check | `config get version` | release-please app, `java-yoshi-mono-repo` |
| google-cloud-python | GitHub Actions, sharded by touched package on PRs | action reads `librarian.yaml` | release-please app, bulk and individual manifests |
| google-cloud-node | GitHub Actions, at most three touched packages on PRs | action reads `librarian.yaml` | release-please app |
| google-cloud-php | GitHub Actions on push to main only | action reads `librarian.yaml` | release-please `php-librarian`; `dev/google-cloud split` |
| google-cloud-ruby | none | n/a | release-please `ruby-librarian`; `toys release` |
| google-cloud-rust | Cloud Build on every PR and merge | `sed` on `librarian.yaml` | manual `bump --all`, `publish`, `tag` |
| google-cloud-dart | GitHub Actions, `generate -all` plus `tidy` | workflow env var **and** `librarian.yaml` | manual, per `generated/RELEASING.md` |
| google-cloud-swift | GitHub Actions, `install` plus `generate --all` | `sed` on `librarian.yaml` | manual release branch; `publish` splits repositories |

The split between the two kinds of generators from step 8 shows up here too:
the four repositories whose generator is an upstream `protoc` plugin or a
tool built from the repository itself (Java, Node.js, PHP, Python) all depend
on in-repository hooks that Librarian executes: `sdk-platform-java` built
with Maven, per-package `librarian.js`, per-component `owlbot.py`, and
`.librarian/generator-input/client-post-processing`. The sidekick
repositories (Rust, Dart, Swift) need nothing from the repository besides
`librarian.yaml`, `keep` lists and the ecosystem formatters.

Librarian's own CI closes the loop from the other side: its language
workflows check out the real consumer repositories and run `install` and
`generate` against them as smoke tests, so a change here that breaks a
consumer fails before it is released.

```excerpt
file: .github/workflows/go.yaml
start: "- name: Generate a single library (smoke test)"
end: "+3"
caption: Librarian's CI generating a real library in a real consumer repository.
```

## Recap: where each command runs and whose code it needs
<!-- step runs="Everywhere" code="summary" -->

| Command | Typical runner | Librarian code | External tools | Language-repository files |
|---|---|---|---|---|
| `install` | CI before generate | orchestration | `go`, `mvn`, `pnpm`, `composer`, `pip`, `gem`, `cargo`, `swift` | `tools:` section |
| `add` | developer | all | none | release-please JSON (go, nodejs, python, ruby) |
| `update` | scheduled automation | all | GitHub API over HTTPS | none |
| `generate` | CI, developers | orchestration; full generator for rust, swift, dart | `protoc`, GAPIC generators, formatters | `keep` lists, `librarian.js`, `toys` tasks, `pom.xml`, `go.mod` |
| `tidy` | developer, add, bump | all | none | none |
| `bump` | release automation | all | `git`, `cargo`, `dart` | version files, git tags |
| `publish` | release automation | rust, dart, swift | `cargo`, `dart pub`, `git` | manifests |
| `tag` | release automation | all | `git` | `librarian.yaml` history |

Three rules of thumb fall out of this:

1. If a step touches the network, it is `update` (GitHub), `generate`
   (source tarballs) or `publish` (registries); everything else is local.
2. If you cannot find a behavior in `internal/librarian`, it is probably in
   an upstream generator for go, java, nodejs, php, python or ruby, or in the
   language repository's own workflow.
3. `librarian.yaml` plus git history is the complete state; there is nothing
   else to inspect or repair.

To go deeper on generation itself, continue with
[Inside generate: how sidekick produces code](generate-and-sidekick.html).
