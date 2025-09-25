# Librarian CLI

[![Go Reference](https://pkg.go.dev/badge/github.com/googleapis/librarian/cmd/librarian.svg)](https://pkg.go.dev/github.com/googleapis/librarian/cmd/librarian)
[![codecov](https://codecov.io/github/googleapis/librarian/graph/badge.svg?token=33d3L7Y0gN)](https://codecov.io/github/googleapis/librarian)

This repository contains code for a unified command line tool for
Google Cloud SDK client library configuration, generation and releasing.

See [CONTRIBUTING.md](CONTRIBUTING.md) for a guide to contributing to this repository,
and [the doc/ folder](doc/) for more detailed project documentation.

The Librarian project supports the Google Cloud SDK ecosystem, and
we do not *expect* it to be of use to external users. That is not
intended to discourage anyone from reading the code and documentation;
it's only to set expectations. (For example, we're unlikely to accept
feature requests for external use cases.)

## Installation

Install [Go](https://go.dev/doc/install) and make sure you have you have the
following in your `PATH`:

```sh
export PATH="$(go env GOPATH)/bin:$PATH"
```

Next install the CLI

```sh
go install github.com/googleapis/librarian/cmd/librarian@latest
```

## Running Librarian

To see the current set of commands available, run:

```sh
librarian -help
```

Use the `-help` or `-h` flag for any individual command to see detailed
documentation for its purpose and associated flags. For example:

```sh
librarian generate -h
```

Alternatively, if you prefer not to have librarian installed you can use the Go
command to run the latest released version:

```sh
go run github.com/googleapis/librarian/cmd/librarian@latest -help
```

## Documentation

- [CLI Documentation](https://pkg.go.dev/github.com/googleapis/librarian/cmd/librarian)
- [Language Onboarding Guide](doc/language-onboarding.md)
- [How We Write Go](doc/howwewritego.md)
- [State Schema](doc/state-schema.md)
- [Config Schema](doc/config-schema.md))
- [Running Tests](doc/testing.md)
- [sidekick](doc/sidekick.md)

## License

Apache 2.0 - See [LICENSE](LICENSE) for more information.
