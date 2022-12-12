---
sidebar_position: 1
sidebar_label: Introduction
---

# The Zed Project

Zed offers a new approach to data that makes it easier to manipulate and manage
your data.

With Zed's new [super-structured data model](formats/README.md#2-zed-a-super-structured-pattern),
messy JSON data can easily be given the fully-typed precision of relational tables
without giving up JSON's uncanny ability to represent eclectic data.

## Getting Started

Trying out Zed is easy: just [install](install.md) the command-line tool
[`zq`](commands/zq.md) and run through the [zq tutorial](tutorials/zq.md).

`zq` is a lot like [`jq`](https://stedolan.github.io/jq/)
but is built from the ground up as a search and analytics engine based
on the [Zed data model](formats/zed.md).  Since Zed data is a
proper superset of JSON, `zq` also works natively with JSON.

While `zq` and the Zed data formats are production quality, the Zed project's
[Zed data lake](commands/zed.md) is a bit [earlier in development](commands/zed.md#status).

For a non-technical user, Zed is as easy to use as web search
while for a technical user, Zed exposes its technical underpinnings
in a gradual slope, providing as much detail as desired,
packaged up in the easy-to-understand
[ZSON data format](formats/zson.md) and
[Zed language](language/README.md).

## Terminology

"Zed" is an umbrella term that describes
a number of different elements of the system:
* The [Zed data model](formats/zed.md) is the abstract definition of the data types and semantics
that underlie the Zed formats.
* The [Zed formats](formats/README.md) are a family of
[sequential (ZNG)](formats/zng.md), [columnar (VNG)](formats/vng.md),
and [human-readable (ZSON)](formats/zson.md) formats that all adhere to the
same abstract Zed data model.
* A [Zed lake](commands/zed.md) is a collection of optionally-indexed Zed data stored
across one or more [data pools](commands/zed.md#14-data-pools) with ACID commit semantics and
accessed via a [Git](https://git-scm.com/)-like API.
* The [Zed language](language/README.md) is the system's dataflow language for performing
queries, searches, analytics, transformations, or any of the above combined together.
* A  [Zed query](language/overview.md#1-introduction) is a Zed script that performs
search and/or analytics.
* A [Zed shaper](language/overview.md#9-shaping) is a Zed script that performs
data transformation to _shape_
the input data into the desired set of organizing Zed data types called "shapes",
which are traditionally called _schemas_ in relational systems but are
much more flexible in the Zed system.

## Digging Deeper

The [Zed language documentation](language/README.md)
is the best way to learn about `zq` in depth.
All of its examples use `zq` commands run on the command line.
Run `zq -h` for a list of command options and online help.

The [Zed Lake documentation](commands/zed.md)
is the best way to learn about `zed`.
All of its examples use `zed` commands run on the command line.
Run `zed -h` or `-h` with any subcommand for a list of command options
and online help.  The same language query that works for `zq` operating
on local files or streams also works for `zed query` operating on a lake.

## Design Philosophy

The design philosophy for Zed is based on composable building blocks
built from self-describing data structures.  Everything in a Zed lake
is built from Zed data and each system component can be run and tested in isolation.

Since Zed data is self-describing, this approach makes stream composition
very easy.  Data from a Zed query can trivially be piped to a local
instance of `zq` by feeding the resulting Zed stream to stdin of `zq`, for example,
```
zed query "from pool | ...remote query..." | zq "...local query..." -
```
There is no need to configure the Zed entities with schema information
like [protobuf configs](https://developers.google.com/protocol-buffers/docs/proto3)
or connections to
[schema registries](https://docs.confluent.io/platform/current/schema-registry/index.html).

A Zed lake is completely self-contained, requiring no auxiliary databases
(like the [Hive metastore](https://cwiki.apache.org/confluence/display/hive/design))
or other third-party services to interpret the lake data.
Once copied, a new service can be instantiated by pointing a `zed serve`
at the copy of the lake.

Functionality like indexing, data compaction, and retention are all
API-driven.

Bite-sized components are unified by the Zed data, usually in the ZNG format:
* All lake meta-data is available via meta-queries.
* All like operations available through the service API are also available
directly via the `zed` command.
* Search indexes and aggregate partials are all just ZNG files and you can
learn about the Zed lake by simply running `zq` on the various ZNG files
in a cloud store.
* Lake management is agent-driven through the API.  For example, instead of complex policies
like data compaction being implemented in the core with some fixed set of
algorithms and policies, an agent can simply hit the API to obtain the meta-data
of the objects in the lake, analyze the objects (e.g., looking for too much
key space overlap) and issue API commands to merge overlapping objects
and delete the old fragmented objects, all with the transactional consistency
of the commit log.
* Components are easily tested and debugged in isolation.
