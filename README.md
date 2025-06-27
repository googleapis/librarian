# Librarian CLI

This repository contains code for a unified command line tool for
Google Cloud SDK client library configuration, generation and releasing.

See [CONTRIBUTING.md][] for a guide to contributing to this repository,
and [the doc/ folder](doc/) for more detailed project documentation.

The Librarian project supports the Google Cloud SDK ecosystem, and
we do not *expect* it to be of use to external users. That is not
intended to discourage anyone from reading the code and documentation;
it's only to set expectations. (For example, we're unlikely to accept
feature requests for external use cases.)

## Running Librarian

To see the current set of commands available, run:

```sh
go run ./cmd/librarian
```

Use the `-h` flag for any individual command to see detailed
documentation for its purpose and associated flags. For example:

```sh
go run ./cmd/librarian generate -h
```

Most commands require a language-specific image to be available;
there are no such images published at the moment.

See https://pkg.go.dev/github.com/googleapis/librarian/cmd/librarian for
additional documentation.

## License

Apache 2.0 - See [LICENSE] for more information.

[contributing]: CONTRIBUTING.md
[license]: LICENSE
