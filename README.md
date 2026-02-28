# Librarian

[![Go Reference](https://pkg.go.dev/badge/github.com/googleapis/librarian/cmd/librarian.svg)](https://pkg.go.dev/github.com/googleapis/librarian/cmd/librarian)
[![codecov](https://codecov.io/github/googleapis/librarian/graph/badge.svg?token=33d3L7Y0gN)](https://codecov.io/github/googleapis/librarian)

This repository contains command line tools for managing Google Cloud SDK
client libraries. The primary tool is `librarian`, which handles the full
library lifecycle: onboarding new libraries, generating code from API
specifications, bumping versions, and publishing releases.

## Usage

Run `librarian -help` for a list of commands,
or see the [command reference](https://pkg.go.dev/github.com/googleapis/librarian/cmd/librarian).

To run without installing:

    go run github.com/googleapis/librarian/cmd/librarian@latest -help

See the [doc/](doc/) folder for additional documentation.

## Contributing

This project supports the Google Cloud SDK ecosystem and is not
intended for external use. For contribution guidelines, see
[CONTRIBUTING.md](CONTRIBUTING.md).

## License

Apache 2.0 - See [LICENSE](LICENSE) for more information.
