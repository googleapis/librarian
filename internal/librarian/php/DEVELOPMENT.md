# PHP Generator Developer Guide

This directory contains the PHP generator implementation for Librarian. This
guide describes the workflow for local development and testing.

## Prerequisites

Before starting, ensure you have the following installed and available on your
system `PATH`:
*   **Go** (to build the Librarian CLI)
*   **PHP** and **Composer** (required to install and run the PHP generator
    plugin) and **bcmath** to run mathematical operations on any size number in the generator
    *   On Debian/Ubuntu (gLinux): `sudo apt-get install php-cli php-bcmath composer`
    plugin)
*   **Node.js** and **npm** (required for `prettier` formatting during post-processing)
    *   Follow oncall guide for Node to install
*   **Python 3** and **venv** (required to run `synthtool` and `owlbot.py` post-processing in an isolated environment)
*   **protoc** (Protocol Buffers compiler, version 33.2 is recommended). If
    `protoc` is not configured in `librarian.yaml` under `tools.protoc`, Librarian
    falls back to using the system-installed `protoc`.
    *   See [GitHub Actions install-protoc setup](../../../.github/actions/install-protoc/action.yaml)
        for installation details.

## Local Workspace Layout

To test changes locally, you should set up a workspace containing the following
sibling repositories (typically under a common parent directory, e.g.,
`repos/`):

*   `librarian/`: This repository.
*   `google-cloud-php/`: The target PHP monorepo where generated client
    libraries reside.

## Local Development Workflow

When modifying the PHP generator or adding PHP support, use the following
workflow to test and verify your changes:

### Step 0: Check out a specific commit (Optional, for consistency testing)

If you want to verify the generator output against a known baseline, check out a specific commit of `google-cloud-php` before running `migrate`:

```bash
cd ../google-cloud-php
git checkout 1831905d
cd ../librarian
```

### Step 1: Run the Migration Tool
Before running generation, you must first generate the `librarian.yaml`
configuration file for the `google-cloud-php` repository. This tool will
auto-discover PHP libraries and map their API paths by parsing `.OwlBot.yaml`
files.

From the `librarian/` repository root:
```bash
go run ./tool/cmd/migrate ../google-cloud-php
```
This writes or updates the `google-cloud-php/librarian.yaml` config.

### Step 2: Build the Librarian CLI
Compile the `librarian` binary locally:
```bash
go build -o bin/librarian ./cmd/librarian
```

> [!NOTE]
> We recommend using `go build` with `-o bin/librarian` rather than `go
> install` for local development. This keeps the development binary isolated
> inside the local project folder and prevents it from overwriting any
> globally installed production version of `librarian` in `~/go/bin`.

### Step 3: Install Generator Tools
Before generating code, you must install the language-specific generator
tools and plugins (e.g. `gapic-generator-php` and `protoc`).

Since `synthtool` is installed via `pip`, you should use a Python virtual environment to avoid PEP 668 system package restrictions.

Navigate to the `google-cloud-php` repository, set up the virtual environment, and run `librarian install`:
```bash
cd ../google-cloud-php
python3 -m venv .venv
source .venv/bin/activate
../librarian/bin/librarian install
```
> [!IMPORTANT]
> Keep the virtual environment activated when running code generation in the next step, so that `owlbot.py` can import `synthtool`.

This downloads the PHP generator, writes a wrapper script, and installs required formatting and synthesis tools in your local workspace.

### Step 4: Run Code Generation
Run the compiled `librarian` binary to generate code for a target library
(e.g., `Ces`):

```bash
../librarian/bin/librarian generate Ces
```
*(Replace `Ces` with the name of the library you are testing).*

### Step 5: Verify and Format Configuration
If you modified `librarian.yaml`, you can format and validate it using the
`tidy` command:

```bash
../librarian/bin/librarian tidy
```
