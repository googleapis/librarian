# Surfer (POC)

This directory contains the source code for the `gcloud` command generator, now a standalone tool named `Surfer`. This tool parses `Protobuf API` definitions and a `gcloud.yaml` configuration file to generate a complete `gcloud` command surface, including commands, flags, and help text.

## Local Testing

This guide provides a simple, self-contained way to run the generator for quick testing and iteration.

### Setting Up the Test Environment

A helper script is provided to automate the setup of a local test environment.

**From the root of the `librarian` repository**, run the following command:

```bash
bash ./librarian/internal/surfer/scripts/setup_test_env.sh
```

This script will create a `test_env` directory in the `librarian` project root, clone the necessary `googleapis` repository, and create a `test.sh` script inside `test_env` for running the generator.

### Running the Generator

Once the setup is complete, you can easily build and run the generator:

**Run the test script:**

```bash
./test_env/test.sh
```

### Verifying the Output

The `test.sh` script will build the `surfer-dev` binary inside `test_env/bin` and then run it. The generated command surface will be created in a new `parallelstore` directory within the `test_env`.
