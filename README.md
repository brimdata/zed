# Zed [![Tests][tests-img]][tests] [![GoPkg][gopkg-img]][gopkg]

Zed is a system for search, analytics, and data transformation
designed to work at any scale, from a simple command-line tool called [`zq`](docs/zq/README.md)
on your desktop (kind of like [`jq`](https://stedolan.github.io/jq/))
to a large-scale, distributed cluster running on your servers or in the cloud
(like a search cluster or a data warehouse).

Zed is based on a new type of data called
[_super-structured_ data](docs/formats/README.md#2-zed-a-super-structured-pattern),
which can represent both relational tables and JSON.  Super-structured data provides a new
and easier approach for data introspection, shaping, pipeline management,
data quality, test, and debug, all while intermixing the flexibility of
the JSON document model with the efficiency of warehouse analytics.

For a non-technical user, Zed is as easy to use as web search
while for a technical user, Zed exposes its technical underpinnings
in a gradual slope, providing as much detail as desired,
packaged up in the easy-to-understand
[ZSON data format](docs/formats/zson.md) and the
[Zed Lake API](docs/zed/api.md).

Check out the [Zed FAQ](FAQ.md).

## Why?

We think that you shouldn't have to set up one system
for search and another completely different system for historical analytics.
And the same search/analytics system that works at cloud scale should run easily as
a lightweight command-line tool on your laptop.

And rather than having to set up complex ETL pipelines with brittle
transformation logic, managing your data lake should be as easy as `git`.

And while _schemas_ are a great way to model and organize your data, they often
[get in the way](https://github.com/brimdata/sharkfest-21#schemas-a-double-edged-sword)
when you are just trying to store or transmit your semi-structured data.

Finally, we believe a lightweight data store that provides easy search and analytics
would be a great place to store data sets for data science and
data engineering experiments running in Python and providing easy
integration with your favorite Python libraries.

## How?

Zed solves all these problems with a new format called
[ZSON](docs/formats/zson.md),
which is a superset of JSON and the relational models.
ZSON is syntax-compatible with JSON
but it has a comprehensive type system that you can use as little or as much as you like.
Zed types can be used as schemas.

Zed also has a cloud-based object design that was modeled after
the `git` design pattern.  Commits to the lake are transactional
and consistent.  Search index updates are also transactionally
consistent with any ingested data, and searches can run with or
without indexes.

## Quick Start

_Detailed documentation [is available](docs/README.md)._

The quickest way to get running on macOS, Linux, or Windows
is to download a pre-built release binary.
You can find these binaries on the GitHub
[releases](https://github.com/brimdata/zed/releases) page.

If you have Go installed, you can easily install `zed` and `zq` by running
```
go install github.com/brimdata/zed/cmd/{zed,zq}@latest
```
On macOS and Linux, you can use `brew` to install `zq`:
```
brew install brimdata/tap/zq
```
To install `zed`, use
```
brew install brimdata/tap/zed
```
Once installed, you can run the query engine from the command-line using `zq`:
```
echo '"hello, world"' | zq -
```
Or you can run a Zed lake service, load it with data using `zed load`, and hit the API.
In one shell, run the server:
```
mkdir scratch
zed serve -lake scratch
```
And in another shell, run the client:
```
zed create Demo
zed use Demo@main
echo '{s:"hello, world"}' | zed load -
zed query "from Demo"
```
You can also use `zed` from Python.  After you install the Zed Python:
```
pip3 install "git+https://github.com/brimdata/zed#subdirectory=python/zed"
```
You can hit the Zed service from a Python program:
```python
import zed

# Connect to the default lake at http://localhost:9867.  To use a
# different lake, supply its URL via the ZED_LAKE environment variable
# or as an argument here.
client = zed.Client()

# Begin executing a Zed query for all records in the pool named "Demo".
# This returns an iterator, not a container.
records = client.query('from Demo')

# Stream records from the server.
for record in records:
    print(record)
```
See the [python/zed](python/zed) for more details.

### Brim

You can use the [Brim app](https://github.com/brimdata/brim)
to explore, query, and shape the data in your Zed lake.

We originally developed Brim for security-oriented use cases
(having tight integration with [Zeek](https://zeek.org/),
[Suricata](https://suricata.io/), and
[Wireshark](https://www.wireshark.org/)),
but we are actively extending Brim with UX for handling generic
data sets to support data science, data engineering, and ETL use cases.

### Building from Source

It's also easy to build `zed` from source:
```
git clone https://github.com/brimdata/zed
cd zed
make install
```
This installs binaries in your `$GOPATH/bin`.

> If you don't have Go installed, download and install it from the
> [Go install page](https://golang.org/doc/install). Go version 1.16 or later is
> required.

## Contributing

See the [contributing guide](CONTRIBUTING.md) on how you can help improve Zed!

## Join the Community

Join our [Public Slack](https://www.brimdata.io/join-slack/) workspace for announcements, Q&A, and to trade tips!

## Acknowledgment

We modeled this README after
Philip O'Toole's brilliantly succinct
[description of `rqlite`](https://github.com/rqlite/rqlite).

[tests-img]: https://github.com/brimdata/zed/workflows/Tests/badge.svg
[tests]: https://github.com/brimdata/zed/actions?query=workflow%3ATests
[gopkg-img]: https://pkg.go.dev/badge/github.com/brimdata/zed
[gopkg]: https://pkg.go.dev/github.com/brimdata/zed
