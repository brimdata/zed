# The Zed Language

**This doc needs to be updated per issue 3604**

## Regular Expressions

TBD: this is linked from below

## Globs

TBD: this is linked from below

## Record Mutations

TBD: this is linked from below

## Type Fusion

TBD: this is linked from below

## Record Literal

TBD: this is linked from below

## Search Expressions

TBD: this is linked from below

## Expressions

TBD: this is linked from below

## Implied Operators

TBD: this is linked from below

## Data Types

TBD: this is linked from below

Explain casting in this section, e.g., `int64(x)`

## TL;DR

Zed is a pipeline-style search and analytics language for querying
data in files, over HTTP, in S3 storage, or in a
[Zed data lake](../zed/README.md).

Zed processes data in [point-free style](https://en.wikipedia.org/wiki/Tacit_programming)
where a sequence of operators takes input, processes it in a sequence of zero or more
Zed operators, and emits the output.

form [PRQL ref]()...

===
XXX canonical form and -C
===
* data model and types
    * sequence of values
* computational model
    * filter, transform, aggregate over sequence
    * order undefined unless explicitly specified with a sort operator
* lateral subsequence operations
* search syntax
* expression syntax
===
    * Zed data is a sequence of Zed values
        * call a stream
        * a lateral stream is derived from a
    * pipeline
    * Zed operators applied to a stream
        * filters (search, matching)
        * transformation
        * aggregation
===

This simple example shows typical Zed query structure:

![Example Zed 1](images/example-zed.png)

As is typical with pipelines, you can imagine the data flowing left-to-right
through this chain of processing elements, such that the output of each element
is the input to the next.  While Zed follows the common pattern seen in
other query languages where the pipeline begins with a search and further
processing is then performed on the isolated data, one of Zed's
strengths is that searches and expressions can appear in any order in the
pipeline.

![Example Zed 2](images/example-zed-operator-search.png)

You can skip ahead to learn about the [search syntax](#search-expressions)
or browse the complete list of [Zed operators](#operators) below. However, we
recommend first continuing on to read about what makes Zed unique and how it
relates to other common data languages such as SQL.

## Background

An ambitious goal of the Zed project is to offer a language
&mdash; the _Zed language_ &mdash;
that provides an easy learning curve and a gentle slope from simple keyword search
to log-search-style processing and ultimately to sophisticated,
warehouse-scale queries.  The language also embraces a rich set of type operators
based on the [Zed data model](../formats/zed.md) for data shaping
to provide flexible and easy ETL.

The simplest Zed program is perhaps a single word search, e.g.,
```
widget
```
This program searches the implied input for Zed records that
contain the string "widget".

> **Note:** The [string-searching algorithm](https://en.wikipedia.org/wiki/String-searching_algorithm)
> implemented in the Zed tools is currently a brute force search based on
> [Boyer-Moore](https://en.wikipedia.org/wiki/Boyer%E2%80%93Moore_string-search_algorithm)
> substring match. In the future, Zed will also offer an approach for locating
> delimited words within text-based fields, which allows for accelerated
> searches with the benefit of an index.

As with the Unix shell and legacy log search systems,
the Zed language embraces a _pipeline_ model where a source of data
is treated as a stream and one or more operators concatenated with
the `|` symbol transform, filter, and aggregate the stream, e.g.,
```
widget | price > 1000
```

That said, the Zed language is
[declarative](https://en.wikipedia.org/wiki/Declarative_programming)
and the Zed compiler optimizes the data flow computation
&mdash; e.g., often implementing a Zed program differently than
the flow implied by the pipeline yet reaching the same result &mdash;
much as a modern SQL engine optimizes a declarative SQL query.

For example, the query above is more efficiently implemented as
a Boolean AND operation instead of two pipeline stages,
so the compiler is free to transform it to
```
widget and price > 1000
```
And since the "AND" syntax is optional (Boolean AND can be expressed as
concatenation), this query can also be expressed as
```
widget price > 1000
```

To facilitate both a programming-like model as well as an ad hoc search
experience, Zed has a canonical, long form that can be abbreviated
using syntax that supports an agile, interactive query workflow.
For example, the canonical form of an aggregation uses the `summarize`
reserved word, as in
```
summarize count() by color
```
but this can be abbreviated by dropping `summarize` whereby the compiler then
uses the name of the aggregation function to resolve the ambiguity, e.g.,
as in the shorter form
```
count() by color
```
Similarly, the canonical form of a search expression includes a `filter`
operator and the searched word enclosed in a `match()` function (also, the "AND"
operator is explicit in canonical form). Therefore the example from above would
be written canonically as
```
filter match("widget") and price > 1000
```
Unlike typical log search systems, the Zed language operators are uniform:
you can specify an operator including keyword search terms, Boolean predicates,
etc. using the same syntax at any point in the pipeline.  For example,
the predicate `count >= 10` can simply be tacked onto the output of a
count aggregation using the filter from above and perhaps sorting
the final output by `count` in a way that's simple to type and edit:
```
widget price > 1000 | count() by color | count >= 10 | sort count
```
The canonical form of this more complex query is:
```
filter match("widget") and price > 1000
| summarize count() by color
| filter count >= 10
| sort count
```

## SQL Compatibility

To encourage adoption by the vast audience of users who know and love SQL,
a key goal of Zed is to support a superset of SQL's SELECT syntax.
For example, the above query can also be written in Zed as
```
SELECT count(), color
WHERE widget AND price > 1000
GROUP BY color
HAVING count >= 10
ORDER BY count
```
i.e., this SQL expression is a subset of the Zed language, and consequently,
the SQL and Zed forms can be mixed and matched:
```
SELECT count(), color
WHERE widget AND price > 1000
GROUP BY color
| count >= 10 | sort count
```
While this hybrid capability of Zed may seem questionable, our goal here
is to have the best of both worlds: the easy interactive workflow of Zed
combined with the ubiquity and familiarity of SQL.

And because the Zed data model
is based on a heterogeneous sequence of arbitrarily typed semi-structured records,
the Zed language is often a better fit here compared to SQL.  For example, an aggregation
that operates on heterogeneous data might look like this:
```
not srcip in 192.168.0.0/16
| summarize
    bytes := sum(src_bytes + dst_bytes),
    maxdur := max(duration),
    valid := and(status == "ok")
      by srcip, dstip
```
This query filters out records with `srcip` in network 192.168
and computes three aggregations over all such records that have the `srcip` and `dstip`
fields where some record have a `status` field, other records
have a `duration` field, and yet other records have
`src_bytes` and `dst_bytes` fields.  Because Zed is more relaxed than SQL,
you can intermix a bunch of related data of different types into a "data pool"
without having to define any upfront schemas
&mdash; let alone a schema per table &mdash;
thereby enabling easy-to-write queries over heterogeneous pools of data.
Writing an equivalent SQL query for the different record types implied above
would require complicated table references, nested selects, and multi-way joins.

> **Note:** The SQL expression implementation is currently in prototype stage.
> If you try it out, you may run into problems and we'd love your
> feedback.
> Feel free to [open an issue](https://github.com/brimdata/zed/issues/new)
> or talk to us on our [public Slack](https://www.brimdata.io/join-slack/)
> about where you've seen it break or where you think it can be improved.

## Data Sources

In the examples above, the data source is implied.  For example, the
`zed query` command takes a list of files and the concatenated files
are the implied input.
Likewise, in the [Brim app](https://github.com/brimdata/brim),
the UI allows for the selection of a data source and key range.

Data sources can also be explicitly specified using the `from` keyword.
Depending on the operating context, `from` may take a file system path,
an HTTP URL, an S3 URL, or in the
context of a Zed lake, the name of a data pool.

## Directed Acyclic Flow Graphs

While the examples above all illustrate a linear sequence of operations,
Zed programs can include multiple data sources and splitting operations
where multiple paths run in parallel and paths can be combined (in an
undefined order), merged (in a defined order) by one or more sort keys,
or joined using relational join logic (currently only merge-based equijoin
is supported).

Generally speaking, a [flow graph](https://en.wikipedia.org/wiki/Directed_acyclic_graph)
defines a directed acyclic graph (DAG) composed
of data sources and operator nodes.  The Zed syntax leverages "fat arrows",
i.e., `=>`, to indicate the start of a parallel path and terminates each
parallel path with a semicolon.

A data path can be split with the `split` operator as in
```
from PoolOne | split (
  => op1 | op2 | ... ;
  => op1 | op2 | ... ;
) | merge ts | ...
```

> **Note:** Adding `merge` to the Zed language is still a work in progress
> ([zed/2906](https://github.com/brimdata/zed/issues/2906)).

Or multiple pools can be accessed and, for example, joined:
```
from (
  pool PoolOne => op1 | op2 | ...
  pool PoolTwo => op1 | op2 | ...
) | join on key=key | ...
```
Similarly, data can be routed to different paths with replication
using `switch`:
```
from ... | switch color (
  case "red" => op1 | op2 | ...
  case "blue" => op1 | op2 | ...
  default => op1 | op2 | ...
) | ...
```

## Operators

Each operator is identified by name and performs a specific operation
on a stream of records.  The entire list of operators is documented
in the [Zed Operator Reference](reference.md#operators).

For three important and commonly used operators, the operator name
is optional, as the compiler can determine from syntax and context which operator
is intended.  This promotes an easy-to-type, interactive UX
for these common use cases.  They include:
* `filter` - select only the records that match a specified [search expression](#search-expressions)
* `summarize` - perform zero or more aggregations with optional [group-by](operators/summarize.md) keys
* `put` - add or modify fields in records

For example, the canonical form of
```
filter match("widget")
| summarize count() by color
| put COLOR := upper(color)
```
can be abbreviated as
```
widget | count() by color | COLOR := upper(color)
```
as the compiler can tell from syntax and context that the three operators
are `filter`, `summarize`, and `put`.

All other operators are explicitly named.

See the references document for a [list of operators](reference.md#operators).

### Operators not yet fully documented

* `from`
* `merge`
* `split`
* `switch`

## Conventions

To build effective queries, it is also important to become familiar with the
Zed _[Data Types](#data-types)_.

Each of the sections hyperlinked above describes these elements of the language
in more detail. To make effective use of the materials, it is recommended to
first review the XXX.  Just put conventions here.  Coming in subsequent PR.
