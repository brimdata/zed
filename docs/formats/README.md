# Zed Formats

This directory contains specifications for the Zed family of
["superstructured"](#zdm.md-2-zed-a-super-structured-design-pattern)
data formats, providing a unified approach to row, columnar, and human-readable formats.
The [Zed data model](zdm.md) underlying the formats
is a superset of both the dataframe/table model of relational systems and the
semi-structured model that is used ubiquitously in development and by NOSQL
data stores.

* [ZSON](zson.md) is a JSON-like, human readable format for Zed data.
* [ZNG](zng.md) is a row-based, binary representation of Zed data somewhat like
Avro but with Zed's more general model for hetereogeneous and self-describing schemas.
* [ZST](zst.md) is a columnar version of ZNG like Parquet or ORC but also
embodies Zed's more general model for hetereogeneous and self-describing schemas.
* [Zed over JSON](zjson.md) defines a JSON format for encapsulating Zed data
in JSON for easy transmission and decoding to JSON-based clients as is
implemented by the [Zealot JavaScript library](https://github.com/brimdata/zealot)
and the [Zed Python library](../../python).
