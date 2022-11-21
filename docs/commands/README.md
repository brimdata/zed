# Command Tooling

The Zed system is managed and queried with the [`zed` command](zed.md),
which is organized into numerous subcommands like the familiar command patterns
of `docker` or `kubectrl`.
Built-in help for the `zed` command and all of its subcommands is always
accessible with the `-h` flag.

The [`zq` command](zq.md) offers a convenient slice of `zed` for running
stand-alone, command-line queries on inputs from files, HTTP URLs, or [S3](../integrations/amazon-s3.md).
`zq` is like [`jq`](https://stedolan.github.io/jq/) but is easier and faster, utilizes the richer
Zed data model, and interoperates with a number of other formats beyond JSON.
If you don't need a Zed lake, you can install just the
slimmer `zq` command which omits lake support and dev tools.

`zq` is always installed alongside `zed`.  You might find yourself mixing and
matching `zed` lake queries with `zq` local queries and stitching them
all together with Unix pipelines.
