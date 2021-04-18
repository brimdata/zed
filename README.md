# Zed [![Tests][tests-img]][tests]

The Zed system provides an open-source, cloud-native, and searchable data lake for
semi-structured data.

Zed lakes utilize a superset of the relational and JSON document data models
yet require no up-front schema definitions to insert data.  They also provide
transactional views and time travel by leveraging a `git`-like design pattern
based on a commit journal.  Using this mechanism, a lake's (optional) search indexes
are transactionally consistent with its data.

At Zed's foundation lies a new family of self-describing data formats based on the
[Zed data model](docs/formats/zson.md#1-introduction),
which unifies the highly structured approach of dataframes and relational tables
with the loosely structured document model of JSON.

While the Zed system is built around its family of data formats, it is also
interoperable with popular data formats like CSV, (ND)JSON, and Parquet.

This repository contains tools and components used to organize, search, analyze,
and store Zed data, including:

* The [`zed`](cmd/zed/README.md) command line tool for managing, searching, and querying a Zed lake
* The [Zed language](docs/language/README.md) documentation
* The [Zed formats](docs/formats/README.md) specifications and documentation

The previously released [`zq`](cmd/zed/README.md#zq) tool is now packaged as
a command-line shortcut for the `zed query` command.

## Installation

To install `zed` or any other tool from this repo, you can either clone the repo
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

See the [contributing guide](CONTRIBUTING.md) on how you can help improve Zed!

## Join the Community

Join our [Public Slack](https://www.brimsecurity.com/join-slack/) workspace for announcements, Q&A, and to trade tips!

[tests-img]: https://github.com/brimdata/zed/workflows/Tests/badge.svg
[tests]: https://github.com/brimdata/zed/actions?query=workflow%3ATests
