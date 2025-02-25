## generator image used

This librarian CLI expects container images from repository configured via `LIBRARIAN_REPOSITORY`. By default it searches for google-cloud-{language}-generator, can override with `-image` flag.

## docker run command

Librarian CLI generates a `docker run` command based on input. 

For `generate`, path specified via `api-root` and `output` will mount as volumes (see [code](https://github.com/googleapis/librarian/blob/fef5706239308400f0ebe622704f98950afba680/internal/container/container.go#L82-L90)). Below is an example with `generate`.

In this example command, language and api-path flag is required, others are optional.
```go
go run ./cmd/librarian generate -language=java -api-path=/path/to/api -api-root=/path/to/googleapis --work-root=/path/to/workspace --output=/path/to/output
```
the docker command run under the hood is:

```
docker run \
    --rm \
    -v /path/to/googleapis:/apis \
    -v /path/to/output:/output \
    google-cloud-java-generator:latest \
    generate \
    --api-root=/apis \
    --output=/output \
    --api-path=/path/to/api
```
After the command run successfully, generated code will reside in the folder specified with `output`. There is also a temporary folder created under /tmp in the process (see [code](https://github.com/googleapis/librarian/blob/fef5706239308400f0ebe622704f98950afba680/internal/command/command.go#L192)) that may hold some intermediate files.

