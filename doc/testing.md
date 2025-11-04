# Testing

## Unit Tests

These tests are designed to test the internal librarian golang logic.

Usage:

```bash
go test ./...
```

## End-to-End (e2e) Tests

These tests are designed to test the CLI interface for each supported Librarian command
on a local system. It tests the interface with docker using a fake repo and test docker
image. These tests are run as presubmits and postsubmits via GitHub actions.

Setup:

```bash
DOCKER_BUILDKIT=1 docker build \
  -f ./testdata/e2e-test.Dockerfile \
  -t test-image:latest \
  .
```

Usage:

```bash
go test -tags e2e
```

## Integration Tests

These tests are designed to test interactions with remote systems (e.g. GitHub). These
tests are **NOT** run automatically as they create pull requests and branches.

Usage:

```bash
LIBRARIAN_TEST_GITHUB_TOKEN=<a personal access token> \
  LIBRARIAN_TEST_GITHUB_REPO=<URL of GitHub repo> \
  go test ./...
```

Note: `LIBRARIAN_TEST_GITHUB_TOKEN` must have write access to `LIBRARIAN_TEST_GITHUB_REPO`.

Note: These tests are skipped unless these environment variables are set.
