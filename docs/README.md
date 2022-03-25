# Zed Documentation

The Zed documentation is organized as follows:

* [formats](formats/README.md) - the Zed data model and serialization formats
of Zed data
* [zq](zq/README.md) - the `zq` command-line tool and Zed query language
* [zed](zed/README.md) - the `zed` command-line tool for managing Zed data lakes and the
API for interacting with a Zed lake
* [tutorials](tutorials) - tutorials on Zed

## Terminology

"Zed" is an umbrella term that describes
a number of different elements of the system:
* The [Zed data model](formats/zed.md) is the abstract definition of the data types and semantics
that underly the Zed formats.
* The [Zed formats](formats/README.md) are a family of
[sequential (ZNG)](formats/zng.md), [columnar (ZST)](formats/zst.md),
and [human-readable (ZSON)](formats/zson.md) formats that all adhere to the
same abstract Zed data model.
* A [Zed lake](zed/README.md) is a collection of optionally-indexed Zed data stored
across one or more [data pools](#14-data-pools) with ACID commit semantics and
accessed via a Git-like API.
* The [Zed language](zq/language.md) is the system's dataflow language for performing
queries, searches, analytics, transformations, or any of the above combined together.
* A  [Zed query](zq/language.md#1-introduction) is a Zed script that performs
search and/or analytics.
* A [Zed shaper](zq/language.md#9-shaping) is a Zed script that performs
data transformation to _shape_
the input data into the desired set of organizing Zed data types called "shapes",
which are traditionally called _schemas_ in relational systems but are
much more flexible in the Zed system.

## Tooling

The Zed system is managed and queried with the `zed` command,
which is organized into numerous subcommands like the familiar command patterns
of `docker` or `kubectrl`.
Built-in help for the `zed` command and all of its subcommands is always
accessible with the `-h` flag.

The `zq` command offers a convenient slice of the `zed` for running
stand-alone, command-line queries on inputs from files, http URLs, or S3.
`zq` is like `jq` but is easier and faster, utilizes the richer
Zed data model, and interoperates with a number of other formats beyond JSON.
If you don't need a Zed lake, you can install just the
slimmer `zq` command which omits lake support and dev tools.

`zq` is always installed alongside `zed`.  You might find yourself mixing and
matching `zed` lake queries with `zq` local queries and stitching them
all together with Unix pipelines.

The [Zed language documentation](zq/language.md)
is the best way to learn about `zq`.
All of its examples use `zq` commands run on the command line.
Run `zq -h` for a list of command options and online help.

The [Zed Lake documentation](zed/README.md)
is the best way to learn about `zed`.
All of its examples use `zed` commands run on the command line.
Run `zed -h` or `-h` with any subcommand for a list of command options
and online help.  The same language query that works for `zq` operating
on local files or streams also works for `zed query` operating on a lake.

For installation instructions, see the [Quick Start](../README.md#quick-start).

### Design Philosophy

The design philosophy for Zed is based on composable building blocks
built from self-describing data structures.  Everything in a Zed lake
is built from Zed data and each system component can be run and tested in isolation.

Since Zed data is self-describing, this approach makes stream composition
very easy.  Data from a Zed query can be trivially be piped to a local
instance of `zq` by feeding the resulting Zed stream to stdin of `zq`, for example,
```
zed query "from pool | ... remote query..." | zq "...local query..." -
```
There is no need to configure the Zed entities with schema information
like [proto configs](https://developers.google.com/protocol-buffers/docs/proto3)
or connections to
[schema registries](https://docs.confluent.io/platform/current/schema-registry/index.html).

A Zed lake is completely self contained requiring no auxiliary databases
(like the [Hive metastore](https://cwiki.apache.org/confluence/display/hive/design))
or other third-party services to interpret the lake data.
Once copied, a new service can be instantiated by pointing a `zed service`
at the copy of the lake.

Functionality like indexing, data compaction, and retention are all
API-driven.

Bite-sized components are unified by the Zed data, usually in the ZNG format:
* All lake meta-data is available via metaqueries.
* All like operations available through the service API are also available
directly via the `zed` command.
* Everything available to the client is
* Search indexes and aggregate partials are all just ZNG files and you can
learn about the Zed lake by simply running `zq` on the various ZNG files
in cloud store.
* Lake management is agent-driven through the API.  Instead of complex policies
like data compaction being implemented in the core with some fixed set of
algorithms and policies, an agent can simply hit the API to obtain the meta-data
of the objects in the lake, analyze the objects (e.g., looking for too much
key space overlap) and issue API commands to say merge overlapping objects
and deleted the old fragmented objects all with the transactional consistency
of the commit log.
* Components are easily tested and debugged in isolation.
