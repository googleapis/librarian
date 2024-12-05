# generator

This repository contains experimental code for SDK generation.

To run the generator:

```
docker build containers/dotnet -t picard
go run ./cmd/generator generate -language dotnet -api secretmanager
```

## License

Apache 2.0 - See [LICENSE] for more information.

[contributing]: CONTRIBUTING.md
[license]: LICENSE
