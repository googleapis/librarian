# PHP Generator Developer Guide

This directory contains the PHP generator implementation for Librarian. This guide describes the workflow for local development and testing.

## Prerequisites

Before starting, ensure you have the following installed and available on your system `PATH`:
* **Go** (to build the Librarian CLI)
* **PHP** and **Composer** (required to install and run the PHP generator plugin)
  * On Debian/Ubuntu (gLinux): `sudo apt-get install php-cli composer`
* **protoc** (Protocol Buffers compiler, version 33.2 is recommended). If `protoc` is not configured in `librarian.yaml` under `tools.protoc`, Librarian falls back to using the system-installed `protoc`.
  * See the [GitHub Actions install-protoc setup](../../../.github/actions/install-protoc/action.yaml) for installation details.

## Local Workspace Layout

To test changes locally, you should set up a workspace containing the following sibling repositories (typically under a common parent directory, e.g., `repos/`):

* `librarian/`: This repository.
* `google-cloud-php/`: The target PHP monorepo where generated client libraries reside.

## Local Development Workflow

When modifying the PHP generator or adding PHP support, use the following workflow to test and verify your changes:

### Step 1: Run the Migration Tool
Before running generation, you must first generate the `librarian.yaml` configuration file for the `google-cloud-php` repository. This tool will auto-discover PHP libraries and map their API paths by parsing `.OwlBot.yaml` files.

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
> We recommend using `go build` with `-o bin/librarian` rather than `go install` for local development. This keeps the development binary isolated inside the local project folder and prevents it from overwriting any globally installed production version of `librarian` in `~/go/bin`.


### Step 3: Install Generator Tools
Before generating code, you must install the language-specific generator tools and plugins (e.g. `gapic-generator-php` and `protoc`).

Navigate to the `google-cloud-php` repository and run `librarian install`:
```bash
cd ../google-cloud-php
../librarian/bin/librarian install
```
This downloads the PHP generator and writes a wrapper script in your local workspace.

### Step 4: Run Code Generation
Run the compiled `librarian` binary to generate code for a target library (e.g., `Ces`):

```bash
../librarian/bin/librarian generate Ces
```
*(Replace `Ces` with the name of the library you are testing).*

### Step 5: Verify and Format Configuration
If you modified `librarian.yaml`, you can format and validate it using the `tidy` command:

```bash
../librarian/bin/librarian tidy
```