# librarian

This repository contains code for a unified command line tool for
SDK client library configuration, generation and releasing.

Sample command lines coming soon, when we have public containers.

## License

Apache 2.0 - See [LICENSE] for more information.

[contributing]: CONTRIBUTING.md
[license]: LICENSE


## Usages

Available commands:
- "configure": Configure a new API in a given language
- "generate": Generate client library code for an API
- "update-apis": Update a language repo by regenerating configured APIs

This librarian CLI expects container images from repository configured via `LIBRARIAN_REPOSITORY`

### generate

Run `go run ./cmd/librarian generate -h` to get helper messages for defined flags and their descriptions. You will see output as below.

```
Usage:

  librarian generate [arguments]

Flags:

  -api-path string
        (Required) path api-root to the API to be generated (e.g., google/cloud/functions/v2)
  -api-root string
        location of googleapis repository. If undefined, googleapis will be cloned to /tmp
  -build
        whether to build the generated code
  -image string
        language-specific container to run for subcommands. Defaults to google-cloud-{language}-generator
  -language string
        (Required) language to generate code for
  -output string
        directory where generated code will be written
  -work-root string
        Working directory root. When this is not specified, a working directory will be created in /tmp.
```