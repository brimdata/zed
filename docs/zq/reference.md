---
sidebar_position: 2
sidebar_label: Language Reference
---

# Reference

Below are links to documentation for the various operators and functions
in the Zed language:
* [Operators](#operators) process a sequence of input values to create an output sequence
and appear as the components of a dataflow pipeline,
* [Functions](#functions) appear in expression context and
take Zed values are arguments and produce a value as a result, and
* [Aggregate Functions](#aggregate-functions) appear in either summarization
or expression context and produce an aggregate value for a sequence of inputs values.

Arguments to function and input values to operators are all dynamically type,
yet certain functions expect certain data types or classes of data types.
To this end, the function and operator prototypes include a number
of type classes as follows:
* _any_ - any Zed data type
* _float_ - any floating point Zed type
* _int_ - any signd or ungigned Zed integer type
* _number_ - either float or int

Note that there is no "any" type in Zed as all super-structured data is
comprehensively type; "any" here simply refers to a value that is allowed
to take on any Zed type.

## Operators

* [combine](operators/combine.md) - combine parallel paths into a single output
* [cut](operators/cut.md) - extract subsets of record fields into new records
* [drop](operators/drop.md) - drop fields from record values
* [from](operators/from.md) - source data from pools, URIs, or connectors
* [fork](operators/fork.md) - copy values to parallel paths
* [fuse](operators/fuse.md) - coerce all input values into a merged type
* [head](operators/head.md) - copy leading values of input sequence
* [join](operators/join.md) - combine data from two inputs using a join predicate
* [over](operators/over.md) - traverse nested values as a lateral query
* [put](operators/put.md) - add or modify fields of records
* [rename](operators/rename.md) - change the name of record fields
* [sample](operators/sample.md) - select one value of each shape
* [search](operators/search.md) - select values based on a search expression
* [sort](operators/sort.md) - sort values
* [summarize](operators/summarize.md) -  perform aggregations
* [switch](operators/switch.md) -  route values based on cases
* [tail](operators/tail.md) - copy trailing values of input sequence
* [uniq](operators/uniq.md) - deduplicate adjacent values
* [where](operators/where.md) - select values based on a Boolean expression
* [yield](operators/yield.md) - emit values from expressions

## Functions

* [abs](functions/abs.md) - absolute value of a number
* [base64](functions/base64.md) - encode/decode base64 strings
* [bucket](functions/bucket.md) - quantize a time or duration value into buckets of equal widths
* [cast](functions/cast.md) - coerce a value to a different type
* [ceil](functions/ceil.md) - ceiling of a number
* [cidr_match](functions/cidr_match.md) - test if IP is in a network
* [crop](functions/crop.md) - remove fields from a value that are missing in a specified type
* [error](functions/error.md) - wrap a value as an error
* [every](functions/every.md) - bucket `ts` using a duration
* [fields](functions/fields.md) - return the flattened path names of a record
* [fill](functions/fill.md) - add null values for missing record fields
* [flatten](functions/flatten.md) - transform a record into a flattened map
* [floor](functions/floor.md) - floor of a number
* [grep](functions/grep.md) - search strings inside of values
* [has](functions/has.md) - test existence of values
* [has_error](functions/has_error.md) - test if a value has an error
* [is](functions/is.md) - test a value's type
* [is_error](functions/is_error.md) - test if a value is an error
* [join](functions/join.md) - concatenate array of strings with a separator
* [kind](functions/kind.md) - return a value's type category
* [ksuid](functions/ksuid.md) - encode/decode KSUID-style unique identifiers
* [len](functions/len.md) - the type-dependent length of a value
* [log](functions/log.md) - natural logarithm
* [lower](functions/lower.md) - convert a string to lower case
* [missing](functions/missing.md) - test for the "missing" error
* [nameof](functions/nameof.md) - the name of a named type
* [network_of](functions/network_of.md) - the network of an IP
* [now](functions/now.md) - the current time
* [order](functions/order.md) - reorder record fields
* [parse_uri](functions/parse_uri.md) - parse a string URI into a structured record
* [parse_zson](functions/parse_zson.md) - parse ZSON text into a Zed value
* [pow](functions/pow.md) - exponential function of any base
* [quiet](functions/quiet.md) - quiet "missing" errors
* [replace](functions/replace.md) - replace one string for another
* [round](functions/round.md) - round a number
* [rune_len](functions/rune_len.md) - length of a string in Unicode code points
* [shape](functions/shape.md) - apply cast, fill, and order
* [split](functions/split.md) - slice a string into an array of strings
* [sqrt](functions/sqrt.md) - square root of a number
* [trim](functions/trim.md) - strip leading and trailing whitespace
* [typename](functions/typename.md) - look up and return a named type
* [typeof](functions/typeof.md) - the type of a value
* [typeunder](functions/typeunder.md) - the underlying type of a value
* [under](functions/under.md) - the underlying value
* [unflatten](functions/unflatten.md) - transform a record with dotted names to a nested record
* [upper](functions/upper.md) - convert a string to upper case

## Aggregate Functions

- [and](aggregates/and.md) - logical AND of input values
- [any](aggregates/any.md) - select an arbitrary value from its input
- [avg](aggregates/avg.md) - average value
- [collect](aggregates/collect.md) - aggregate values into array
- [count](aggregates/count.md) - count input values
- [countdistinct](aggregates/count.md) - count distinct input values
- [fuse](aggregates/fuse.md) - compute a fused type of input values
- [max](aggregates/max.md) - maximum value of input values
- [min](aggregates/min.md) - minimum value of input values
- [or](aggregates/or.md) - logical OR of input values
- [sum](aggregates/sum.md) - sum of input values
- [union](aggregates/union.md) - set union of input values
