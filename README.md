# `zed` [![Tests][tests-img]][tests]

The zed project is a new, clean-slate design for a data-engineering stack.
At zed's foundation lies a new family of self-describing
data formats based on the "zed data model", which blends the highly structured
approach of dataframes and relational tables with the loosely structured
document model of JSON.

While the zed system is built around its family of data formats, it is also
interoperable with popular data formats like CSV, (ND)JSON, and Parquet.

This repository contains tools and components used to organize, search, analyze,
and store zed data, including:

* The [zq](cmd/zq/README.md) command line tool for searching and analyzing data
* The [zqd](ppl/cmd/zqd/README.md) daemon, which serves a REST API to manage
 and query zed data lakes, and is the backend for the [Brim](https://github.com/brimdata/brim)
 application
* The [zapi](cmd/zapi/README.md) command line tool, for interacting with the
API provided by zqd
* The [zed language](docs/language/README.md) documentation
* The [zed formats](docs/formats/README.md) specifications and documentation

We believe the zed data architecture provides a powerful foundation for the
modern data lake and are actively developing tools and software components
for the emerging "zed data lake".

## Installation

To install `zq` or any other tool from this repo, you can either clone the repo
 and compile from source, or use a pre-compiled
 [release](https://github.com/brimdata/zed/releases), available for Windows, macOS, and Linux.

If you don't have Go installed, download and install it from the
[Go downloads page](https://golang.org/dl/). Go version 1.16 or later is
required.

To install the binaries in `$GOPATH/bin`, clone this repo and
execute `make install`:

```
git clone https://github.com/brimdata/zed
cd zed
make install
```

## Contributing

See the [contributing guide](CONTRIBUTING.md) on how you can help improve `zed`!

## Join the Community

Join our [Public Slack](https://www.brimsecurity.com/join-slack/) workspace for announcements, Q&A, and to trade tips!

[tests-img]: https://github.com/brimdata/zed/workflows/Tests/badge.svg
[tests]: https://github.com/brimdata/zed/actions?query=workflow%3ATests
