# `zq` log query language (ZQL)

ZQL is a powerful query language for searching and analyzing event data. It is in many ways optimal for working with [Zeek](https://www.zeek.org/) data, though it can be used to query any data in in [ZNG](../../zng/docs/README.md) or [NDJSON](http://ndjson.org/) format.

The language embraces a syntax that should be familiar to those who have worked with UNIX/Linux shells. At a high level, each query consists of a _[search](search-syntax/README.md)_ portion and an optional _pipeline_. Here's a simple example query:

![Simple Example Query](images/simple-example-query.png)

As is typical with pipelines, you can imagine the data flowing left-to-right through this chain of processing elements, such that the output of each element is the input to the next. The search portion first isolates a set of the stored event data, then each element of the pipeline performs additional operations on the data.

The available pipeline elements are broadly categorized into:

* _[Processors](processors/README.md)_, that filter or transform events, and,
* _[Aggregate Functions](aggregate-functions/README.md)_. that carry out running computations based on the values of fields in successive events.

To build effective queries, it is also important to become familiar with ZQL's supported _[Data Types](data-types/README.md)_.

Each of the following sections describes these elements of the query language in more detail. To make effective use of the materials, it is recommended to first review the [Documentation Conventions](conventions/README.md). You will likely want to start out working with the [Sample Data](https://github.com/brimdata/zq-sample-data) so you can reproduce the examples shown.

# Sections

* [Documentation Conventions](conventions/README.md)
* [Search syntax](search-syntax/README.md)
* [Processors](processors/README.md)
* [Aggregate Functions](aggregate-functions/README.md)
* [Grouping](grouping/README.md)
* [Data Types](data-types/README.md)
