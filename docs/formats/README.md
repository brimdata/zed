# Zed Formats

This directory contains specifications for the zed family of data formats,
providing a unified approach to row, columnar, and human-readable formats.
The zed data model underlying the formats
is a superset of both the dataframe/table model of relational systems and the
semi-structured model that is used ubiquitously in development and by NOSQL
data stores.

* [ZSON](zson.md) is a JSON-like, human readable format for zed data.
* [ZNG](zng.md) is a row-based, binary representation of zed data somewhat like
Avro but with zed's more general model for hetereogeneous and self-describing schemas.
* [ZST](zst.md) is a columnar version of ZNG like Parquet or ORC but also
embodies zed's more general model for hetereogeneous and self-describing schemas.
* [ZNG over JSON](zng-over-json.md) defines a JSON format for encapsulating zed data
in JSON for easy transmission and decoding to JSON-based clients as is
implemented by the
[zealot javascript library](https://github.com/brimdata/brim/tree/master/zealot)
and the
[zed python library](https://github.com/brimdata/zq/tree/main/python).
