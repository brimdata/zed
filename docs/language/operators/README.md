# Operators

---

Dataflow operators process a sequence of input values to create an output sequence
and appear as the components of a dataflow pipeline.

* [assert](assert.md) - evaluate an assertion
* [combine](combine.md) - combine parallel paths into a single output
* [cut](cut.md) - extract subsets of record fields into new records
* [drop](drop.md) - drop fields from record values
* [file](from.md) - source data from a file
* [from](from.md) - source data from pools, files, or URIs
* [fork](fork.md) - copy values to parallel paths
* [fuse](fuse.md) - coerce all input values into a merged type
* [get](from.md) - source data from a URI
* [head](head.md) - copy leading values of input sequence
* [join](join.md) - combine data from two inputs using a join predicate
* [over](over.md) - traverse nested values as a lateral query
* [put](put.md) - add or modify fields of records
* [rename](rename.md) - change the name of record fields
* [sample](sample.md) - select one value of each shape
* [search](search.md) - select values based on a search expression
* [sort](sort.md) - sort values
* [summarize](summarize.md) -  perform aggregations
* [switch](switch.md) -  route values based on cases
* [tail](tail.md) - copy trailing values of input sequence
* [uniq](uniq.md) - deduplicate adjacent values
* [where](where.md) - select values based on a Boolean expression
* [yield](yield.md) - emit values from expressions
