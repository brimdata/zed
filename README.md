# Zed [![Tests][tests-img]][tests] [![GoPkg][gopkg-img]][gopkg]

Zed offers a new approach to data that makes it easier to manipulate and manage
your data.

With Zed's new
[super-structured data model](https://zed.brimdata.io/docs/formats/#2-zed-a-super-structured-pattern),
messy JSON data can easily be given the fully-typed precision of relational tables
without giving up JSON's uncanny ability to represent eclectic data.

Trying out Zed is easy: just [install](https://zed.brimdata.io/docs/#getting-started)
the command-line tool [`zq`](https://zed.brimdata.io/docs/commands/zq/).

`zq` is a lot like [`jq`](https://stedolan.github.io/jq/)
but is built from the ground up as a search and analytics engine based
on the [Zed data model](https://zed.brimdata.io/docs/formats/zed).
Since Zed data is a proper superset of JSON, `zq` also works natively with JSON.

While `zq` and the Zed data formats are production quality, the Zed project's
[Zed data lake](https://zed.brimdata.io/docs/commands/zed/#1-the-lake-model)
is a bit [earlier in development](https://zed.brimdata.io/docs/commands/zed/#status).

For a non-technical user, Zed is as easy to use as web search
while for a technical user, Zed exposes its technical underpinnings
in a gradual slope, providing as much detail as desired,
packaged up in the easy-to-understand
[ZSON data format](https://zed.brimdata.io/docs/formats/zson) and
[Zed language](https://zed.brimdata.io/docs/language).

## Why?

We think data is hard and it should be much, much easier.

While _schemas_ are a great way to model and organize your data, they often
[get in the way](https://github.com/brimdata/sharkfest-21#schemas-a-double-edged-sword)
when you are just trying to store or transmit your semi-structured data.

Also, why should you have to set up one system
for search and another completely different system for historical analytics?
And the same unified search/analytics system that works at cloud scale should run easily as
a lightweight command-line tool on your laptop.

And rather than having to set up complex ETL pipelines with brittle
transformation logic, managing your data lake should be as easy as
[`git`](https://git-scm.com/).

Finally, we believe a lightweight data store that provides easy search and analytics
would be a great place to store data sets for data science and
data engineering experiments running in Python and providing easy
integration with your favorite Python libraries.

## How?

Zed solves all these problems with a new foundational data format called
[ZSON](https://zed.brimdata.io/docs/formats/zson),
which is a superset of JSON and the relational models.
ZSON is syntax-compatible with JSON
but it has a comprehensive type system that you can use as little or as much as you like.
Zed types can be used as schemas.

The [Zed language](https://zed.brimdata.io/docs/language) offers a gentle learning curve,
which spans the gamut from simple
[keyword search](https://zed.brimdata.io/docs/language/#7-search-expressions)
to powerful data-transformation operators like
[lateral sub-queries](https://zed.brimdata.io/docs/language/#8-lateral-subqueries)
and [shaping](https://zed.brimdata.io/docs/language/#9-shaping).

Zed also has a cloud-based object design that was modeled after
the `git` design pattern.  Commits to the lake are transactional
and consistent.  Search index updates are also transactionally
consistent with any ingested data, and searches can run with or
without indexes.

## Quick Start

Check out the [installation page](https://zed.brimdata.io/docs/install.md)
for a quick and easy intall.

Detailed documentation is available on the [Zed docs site](https://zed.brimdata.io/docs).

### Brim

The [Brim app](https://github.com/brimdata/brim) is an electron-based
desktop app to explore, query, and shape data in your Zed lake.

We originally developed Brim for security-oriented use cases
(having tight integration with [Zeek](https://zeek.org/),
[Suricata](https://suricata.io/), and
[Wireshark](https://www.wireshark.org/)),
but we are actively extending Brim with UX for handling generic
data sets to support data science, data engineering, and ETL use cases.

## Contributing

See the [contributing guide](CONTRIBUTING.md) on how you can help improve Zed!

## Join the Community

Join our [public Slack](https://www.brimdata.io/join-slack/) workspace for announcements, Q&A, and to trade tips!

## Acknowledgment

We modeled this README after
Philip O'Toole's brilliantly succinct
[description of `rqlite`](https://github.com/rqlite/rqlite).

[tests-img]: https://github.com/brimdata/zed/workflows/Tests/badge.svg
[tests]: https://github.com/brimdata/zed/actions?query=workflow%3ATests
[gopkg-img]: https://pkg.go.dev/badge/github.com/brimdata/zed
[gopkg]: https://pkg.go.dev/github.com/brimdata/zed
