# ZNG Specification

> ### Note: This specification is in BETA development.
> We hope that no backward incompatible changes will be made during
> the BETA phase.  We plan to
> declare the specification stable in the near future.
>
> [Zed](../../README.md)'s
> implementation of ZNG is tracking this spec and as it changes,
> the zq output format is subject to change.  In this branch,
> zq attempts to implement everything herein excepting:
>
> * Only streams of `record` types (which may consist of any combination of
>   other implemented types) are supported by zq even though a stream of
>   any types may currently be expressed in value messages.
> * ZQL syntax for working with the [`enum` type](#3115-enum-typedef) is not yet implemented. ([#1498](https://github.com/brimdata/zed/issues/1498))
> * [Primitive Types](#3-primitive-types) for `float16`, `float32`, and `decimal` are not yet implemented. ([#1312](https://github.com/brimdata/zed/issues/1312), [#1522](https://github.com/brimdata/zed/issues/1522))

* [1. Introduction](#1-introduction)
* [2. The ZNG Format](#2-the-zng-format)
  + [2.1 Control Messages](#21-control-messages)
    - [2.1.1 Typedefs](#211-typedefs)
      - [2.1.1.1 Record Typedef](#2111-record-typedef)
      - [2.1.1.2 Array Typedef](#2112-array-typedef)
      - [2.1.1.3 Set Typedef](#2113-set-typedef)
      - [2.1.1.4 Union Typedef](#2114-union-typedef)
      - [2.1.1.5 Enum Typedef](#2115-enum-typedef)
      - [2.1.1.6 Map Typedef](#2116-map-typedef)
      - [2.1.1.7 Alias Typedef](#2117-alias-typedef)
    - [2.1.2 Compressed Value Message Block](#212-compressed-value-message-block)
    - [2.1.3 Application-Defined Messages](#213-application-defined-messages)
    - [2.1.4 End-of-Stream Markers](#214-end-of-stream-markers)
  + [2.2 Value Messages](#22-value-messages)
* [3. Primitive Types](#3-primitive-types)
* [4. Type Values](#4-type-values)
* [Appendix A. Related Links](#appendix-a-related-links)
* [Appendix B. Recommended Type Coercion Rules](#appendix-b-recommended-type-coercion-rules)

## 1. Introduction

ZNG implements the [ZSON](zson.md) data model and is an efficient, binary serialization
format for ZSON value sequences.  ZNG is ideally suited for streams
of heterogeneously typed records, e.g., structured logs, where filtering and
analytics may be applied to a stream in parts without having to fully deserialize
every value.

ZNG is richly typed and thinner on the wire than JSON.
ZNG strikes a balance between the narrowly typed but flexible
[newline-delimited JSON (NDJSON)](http://ndjson.org/) format and
a more structured approach like [Apache Avro](https://avro.apache.org).

As it follows the ZSON data model, ZNG embeds all type information
in the stream itself while having a binary serialization format that
allows "lazy parsing" of fields such that
only the fields of interest in a stream need to be deserialized and interpreted.
Unlike Avro, ZNG embeds its "schemas" in the data stream and thereby admits
an efficient multiplexing of heterogeneous data types by prepending to each
data value a simple integer identifier to reference its type.

ZNG requires no external schema definitions as its type system
constructs schemas on the fly from within the stream using composable,
dynamic type definitions.  The state comprising the dynamically constructed
types is called the "type context".
Given a type context, there is no need for
a schema registry service, though ZNG can be readily adapted to systems like
[Apache Kafka](https://kafka.apache.org/) which utilize such registries,
by having a connector translate the schemas implied in the
ZNG stream into registered schemas and vice versa.

Multiple ZNG streams with different type contexts are easily merged because the
serialization of values does not depend on the details of
the type context.  One or more streams can be merged by simply merging the
input contexts into an output context and adjusting the type reference of
each value in the output ZNG sequence.  The values need not be traversed
or otherwise rewritten to be merged in this fashion.

ZNG is more expressive than JSON in that any JSON input
can be mapped onto ZNG and recovered by decoding
that ZNG back into JSON, but the converse is not true.

Likewise, ZNG is a superset of SQL relational tables so any table could potentially
be exported as ZNG data and re-imported from ZNG to SQL (though there is
no notion of constraints like foreign keys in the ZNG format).

The [`zq`](https://github.com/brimdata/zed) command-line tool provides a
reference implementation of ZNG as it's described here, including the type
system, error handling, etc., barring the exceptions
described in the [beta notice](#note-this-specification-is-in-beta-development)
at the top of this specification.

## 2. The ZNG Format

The ZNG binary format is based on machine-readable data types with an
encoding methodology inspired by Avro,
[Parquet](https://en.wikipedia.org/wiki/Apache_Parquet), and
[Protocol Buffers](https://developers.google.com/protocol-buffers).

ZNG follows the ZSON data model and
encodes a sequence of one or more typed data values to comprise a stream.
The stream of values is interleaved with control messages
that provide type definitions and other metadata.  The type of
a particular data value is specified by its "type identifier", or type ID,
which is an integer representing either a "primitive type" or a
"complex type".

A ZNG stream comprises a sequence of interleaved control messages and value messages
that are serialized into a stream of bytes.

Each message is prefixed with a single-byte header code.  Codes `0xf6-0xff`
are allocated as control messages while codes `0x00-0xf5` indicate a value message.

### 2.1 Control Messages

Control codes `0xf6` through `0xff` (in hexadecimal) are defined as follows:

| Code   | Message Type                   |
|--------|--------------------------------|
| `0xf6` | record definition              |
| `0xf7` | array definition               |
| `0xf8` | set definition                 |
| `0xf9` | union definition               |
| `0xfa` | enum definiton                 |
| `0xfb` | map definiton                  |
| `0xfc` | type alias                     |
| `0xfd` | compressed value message block |
| `0xfe` | application-defined message    |
| `0xff` | end-of-stream                  |

The application-defined messages are available to higher-layer protocols and
potential future variations of ZNG.  A ZNG implementation that
merely skips over all of the application-defined messages is guaranteed by
this specification to decode all of the data as described herein even if such
messages provide additional semantics on top of the base ZNG format.

Any such application-defined messages not known by
a ZNG data receiver shall be ignored.

The body of a application-defined control message is typically a structured
message in JSON, ZSON, or ZNG.
These messages are guaranteed to be preserved
in order within the stream and presented to higher layer components through
any ZNG streaming API.  In this way, senders and receivers of ZNG can embed
protocol directives as ZNG control payloads rather than defining additional
encapsulating protocols.

> For example, the [zqd](../../cmd/zed/lake) service uses application-defined
> message `0xf7` to embed search and server stats in the return stream of ZNG
> data, e.g., as a long-running search progresses on the server.

### 2.1.1 Typedefs

Following a header byte of `0xf6-0xfb` is a "typedef".  A typedef binds
"the next available" integer type ID to a type encoding.  As there are
a total of 23 primitive type IDs, the Type IDs for typedefs
begin at the value 23 and increase by one for each typedef. These bindings
are scoped to the stream in which the typedef occurs.

Type IDs for the "primitive types" need not be defined with typedefs and
are predefined with the IDs shown in the [Primitive Types](#-primitive-types) table.

A typedef is encoded as a single byte indicating the complex type ID followed by
the type encoding.  This creates a binding between the implied type ID
(i.e., 23 plus the count of all previous typedefs in the stream) and the new
type definition.

The type ID is encoded as a `uvarint`, an encoding used throughout the ZNG format.

> Inspired by Protocol Buffers,
> a `uvarint` is an unsigned, variable-length integer encoded as a sequence of
> bytes consisting of N-1 bytes with bit 7 clear and the Nth byte with bit 7 set,
> whose value is the base-128 number composed of the digits defined by the lower
> 7 bits of each byte from least-significant digit (byte 0) to
> most-significant digit (byte N-1).

#### 2.1.1.1 Record Typedef

A record typedef creates a new type ID equal to the next stream type ID
with the following structure:
```
---------------------------------------------------------
|0xf6|<ncolumns>|<name1><type-id-1><name2><type-id-2>...|
---------------------------------------------------------
```
Record types consist of an ordered set of columns where each column consists of
a name and its type.  Unlike JSON, the ordering of the columns is significant
and must be preserved through any APIs that consume, process, and emit ZNG records.

A record type is encoded as a count of fields, i.e., `<ncolumns>` from above,
followed by the field definitions,
where a field definition is a field name followed by a type ID, i.e.,
`<name1>` followed by `<type-id-1>` etc. as indicated above.

The field names in a record must be unique.

The `<ncolumns>` is encoded as a `uvarint`.

The field name is encoded as a UTF-8 string defining a "ZNG identifier".
The UTF-8 string
is further encoded as a "counted string", which is the `uvarint` encoding
of the length of the string followed by that many bytes of UTF-8 encoded
string data.

N.B.: As defined by ZSON, a field name can be any valid UTF-8 string much like JSON
objects can be indexed with arbitrary string keys (via index operator)
even if the field names available to the dot operator are restricted
by language syntax for identifiers.

The type ID follows the field name and is encoded as a `uvarint`.

#### 2.1.1.2 Array Typedef

An array type is encoded as simply the type code of the elements of
the array encoded as a `uvarint`:
```
----------------
|0xf7|<type-id>|
----------------
```

#### 2.1.1.3 Set Typedef

A set type is encoded as the type ID of the
elements of the set, encoded as a `uvarint`:
```
----------------
|0xf8|<type-id>|
----------------
```

#### 2.1.1.4 Union Typedef

A union typedef creates a new type ID equal to the next stream type ID
with the following structure:
```
-----------------------------------------
|0xf9|<ntypes>|<type-id-1><type-id-2>...|
-----------------------------------------
```
A union type consists of an ordered set of types
encoded as a count of the number of types, i.e., `<ntypes>` from above,
followed by the type IDs comprising the types of the union.
The type IDs of a union must be unique.

The `<ntypes>` and the type IDs are all encoded as `uvarint`.

`<ntypes>` cannot be 0.

#### 2.1.1.5 Enum Typedef

An enum type is encoded as a `uvarint` representing the number of symbols
in the enumeration followed by the names of each symbol.
```
--------------------------------
|0xfa|<nelem>|<name1><name2>...|
--------------------------------
```
`<nelem>` is encoded as `uvarint`.
The names have the same UTF-8 format as record field names and are encoded
as counted strings following the same convention as record field names.

#### 2.1.1.6 Map Typedef

A map type is encoded as the type code of the key
followed by the type code of the value.
```
--------------------------
|0xfb|<type-id>|<type-id>|
--------------------------
```
Each `<type-id>` is encoded as `uvarint`.


#### 2.1.1.7 Alias Typedef

A type alias defines a new type ID that binds a new type name
to a previously existing type ID.  This is useful for systems like Zeek,
where there are customary type names that are well-known to users of the
Zeek system and are easily mapped onto a ZNG type having a different name.
By encoding the aliases in the format, there is no need to configure mapping
information across different systems using the format, as the type aliases
are communicated to the consumer of a ZNG stream.

Type aliases can also be used like the "logical types" in Avro and Parquet,
where higher-level semantics can be defined for a particular named type,
all outside the scope of the base ZNG specification.

A type alias is encoded as follows:
```
----------------------
|0xfc|<name><type-id>|
----------------------
```
where `<name>` is an identifier representing the new type name with a new type ID
allocated as the next available type ID in the stream that refers to the
existing type ID `<type-id>.  `<type-id> is encoded as a `uvarint` and `<name>`
is encoded as a `uvarint` representing the length of the name in bytes,
followed by that many bytes of UTF-8 string.

It is an error to define an alias that has the same name as a primitive type.
It is permissible to redefine a previously defined alias with a
type that differs from the previous definition.

### 2.1.2 Compressed Value Message Block

Following a header byte of `0xf6` is a compressed value message block.
Such a block comprises a compressed sequence of value messages.  The
sequence must not include control messages.

> The reason control messages are not allowed in compressed blocks is to
> allow for optimizations that discard entire buffers of data based on
> heuristics to know a filtering predicate can't be true based on a
> quick scan of the data (e.g., using the Boyer-Moore algorithm to determine
> that a comparison with a string constant would not work for any record
> value in the buffer).  Since blocks may be dropped without parsing using
> such an optimization, any typedefs should be lifted out into the zng data
> stream in front of the compressed blocks (i.e., the stream is rearranged
> but it's always safe to move typedefs earlier in the stream as long as
> the typedef order is preserved and a zng end-of-stream is not crossed).
> For application-specific messages and end-of-stream, a compressed buffer
> should be terminated and these messages sent as uncompressed data.
>
> Since ZNG streams typically consist of a very sparse
> set of typedefs with very long runs of data, these constraints are not
> a barrier to performance in practice.

A compressed value message block is encoded as follows:
```
-------------------------------------------------------------------------------
|0xfd|<format>|<uncompressed-length>|<compressed-length>|<compressed-messages>|
-------------------------------------------------------------------------------
```
where
* `<format>`, a `uvarint`, identifies the compression algorithm applied to the
  message sequence,
* `<uncompressed-length>`, a `uvarint`, is the length in bytes of the
  uncompressed message sequence, and
* `<compressed-length>`, a `uvarint`, is the length in bytes of `<compressed-messages>`
* `<compressed-messages>` is the compressed value message sequence.

Values for `<format>` are defined in the
[ZNG compression format specification](./compression-spec.md).

### 2.1.3 Application-Defined Messages

An application-defined message has the following form:
```
------------------------------
|0xfe|<encoding>|<len>|<body>|
------------------------------
```
where
* `<encoding>` is a single byte indicating whether the body is encoded
as ZNG (0), JSON (1), ZSON (2), an arbitrary UTF-8 string (3), or arbitrary binary data (4),
* `<len>` is a `uvarint` encoding the length in bytes of the message body
(exclusive of the length 1 encoding byte), and
* `<body>` is a data message whose semantics are outside the scope of
the base ZNG specification.

If the encoding type is ZNG, the embedded ZNG data
starts and ends a single ZNG stream independent of outer the ZNG stream.

### 2.1.4 End-of-Stream Markers

A ZNG stream must be terminated by an end-of-stream marker.
A new ZNG stream may begin immediately after an end-of-stream marker.
Each such stream has its own, independent type context.

In this way, the concatenation of ZNG streams (or ZNG files containing
ZNG streams) results in a valid ZNG data sequence.

For example, a large ZNG file can be arranged into multiple, smaller streams
to facilitate random access at stream boundaries.
This benefit comes at the cost of some additional overhead --
the space consumed by stream boundary markers and repeated type definitions.
Choosing an appropriate stream size that balances this overhead with the
benefit of enabling random access is left up to implementations.

End-of-stream markers are also useful in the context of sending ZNG over Kafka,
as a receiver can easily resynchronize with a live Kafka topic by
discarding incomplete messages until a message is found that is terminated
by an end-of-stream marker (presuming the sender implementation aligns
the ZNG messages on Kafka message boundaries).

A end-of-stream marker is encoded as follows:
```
------
|0xff|
------
```

After this marker, all previously read
typedefs are invalidated and the "next available type ID" is reset to
the initial value of 23.  To represent subsequent records that use a
previously defined type, the appropriate typedef control code must
be re-emitted
(and note that the typedef may now be assigned a different ID).

### 2.2 Value Messages

Following a header byte in the range `0x00-0xf5` is a ZNG value.
The header byte indicates the type ID of the value.  If the type ID
is larger than `0xf4`, then the type ID is "escaped" with the value `0xf5`
and the actual type ID is encoded as a `uvarint` of the difference
of the type ID less the constant `0xf5`.

It is an error for a value to reference a type ID that has not been
previously defined by a typedef scoped to the stream in which the value
appears.

The value is encoded in the subsequent bytes using a "tag-encoding" scheme
that captures the structure of both primitive types and the recursive
nature of complex types.  This structure is encoded
explicitly in every value and the boundaries of each value and its
recursive nesting can be parsed without knowledge of the type or types of
the underlying values.  This admits an efficient implementation
for traversing the values, inclusive of recursive traversal of complex values,
whereby the inner loop need not consult and interpret the type ID of each element.

#### 2.2.1 Tag-Encoding of Values

Each value is prefixed with a "tag" that defines:
* whether it is a primitive or complex value,
* whether it is the null value, and
* its encoded length in bytes.

The collection of sub-values comprising a complex-type value
is called a "container".

To encode the length N of the value, a bit for the complex/primitive type indicator,
and representation for the null value,
The tag for a container of length N is
```
2*N + 1
```
The tag for a primitive of length N is
```
2*N + 2
```
The tag for the null value is 0.

For example, the following tags have the following meanings:

| Tag |    Meaning          |
|-----|---------------------|
|  0  | null                |
|  1  | length 0 container  |
|  2  | length 0 primitive  |
|  3  | length 1 container  |
|  4  | length 1 primitive  |
|  5  | length 2 container  |
|  6  | length 2 primitive  |
| ... | etc                 |

A container recursively contains a list of tagged values.  Since the container
encodes its overall length, there is no need to encode the number of elements
in a container as they are easily discovered by scanning the buffer for each value
until the last tagged value is encountered.

#### 2.2.2 Tag-Encoded Body of Primitive Values

Following the tag encoding is the value encoded in N bytes as described above.
A typed value with a `value` of length `N` is interpreted as described in the
[Primitive Types](#3-primitive-types) table.  The type information needed to
interpret all of the value elements of a complex type are all implied by the
top-level type ID of the value message.  For example, the type ID could indicate
a particular record type, which recursively provides the type information
for all of the elements within that record, including other complex types
embedded within the top-level record.

Note that because the tag indicates the length of the value, there is no need
to use varint encoding of integer values.  Instead, an integer value is encoded
using the full 8 bits of each byte in little-endian order.  For signed values,
before encoding, are shifted left one bit, and the sign bit stored as bit 0.
For negative numbers, the remaining bits are negated so that the upper bytes
tend to be zero-filled for small integers.

#### 2.2.2 Tag-Encoded Body of Complex Values

The body of a length-N container comprises zero or more tag-encoded values,
where the values are encoded as follows:

| Type     |          Value                            |
|----------|-------------------------------------------|
| `array`  | concatenation of elements                 |
| `set`    | normalized concatenation of elements      |
| `record` | concatenation of elements                 |
| `union`  | concatenation of type position and value  |
| `enum`   | position of enum element                  |
| `map`    | concatenation of key and value elements   |

Since N, the byte length of any of these container values, is known,
there is no need to encode a count of the
elements present.  Also, since the type ID is implied by the typedef
of any complex type, each value is encoded without its type ID.

For sets, the concatenation of elements must be normalized so that the
sequence of bytes encoding each element's tag-counted value is
lexicographically greater than that of the preceding element.

A union value is encoded as a container with two elements. The first
element is the `uvarint` encoding of the positional index determining
the type of the value in reference to the union's list of defined types,
and the second element is the value encoded according to that type.

An enumeration value is represented as the `uvarint` encoding of the
positional index of that value's symbol in reference to the enum's
list of defined symbols.

A map value is encoded as a container as a sequence of alternating tag-encoded
key and value encoded as keys and values of the underlying key and value types.
The concatenation of elements must be normalized so that the
sequence of bytes encoding each tag-counted key (of the key/value pair) is
lexicographically greater than that of the preceding key (of the preceding
key/value pair).

> XXX it's not clear that we should require sorted keys and give up
> on a single canonical form for the zng file.  Or make this sort optional.
> Or have the zq/zng writer do the sort but not correct it on input.

## 3. Primitive Types

For each ZNG primitive type, the following table describes:
* The predefined ID, which need not be defined in [ZNG Typedefs](#211-typedefs)
* How a typed `value` of length `N` is interpreted in a [ZNG Value Message](#22-value-messages)

All multi-byte sequences, which are not varints (e.g., float64, ip, etc),
representing machine words are serialized in little-endian format.


| Type       | ID |    N     |       ZNG Value Interpretation                 |
|------------|---:|:--------:|------------------------------------------------|
| `uint8`    |  0 | variable  | unsigned int of length N                       |
| `uint16`   |  1 | variable | unsigned int of length N                       |
| `uint32`   |  2 | variable | unsigned int of length N                       |
| `uint64`   |  3 | variable | unsigned int of length N                       |
| `int8`     |  4 | variable | signed int of length N                         |
| `int16`    |  5 | variable | signed int of length N                         |
| `int32`    |  6 | variable | signed int of length N                         |
| `int64`    |  7 | variable | signed int of length N                         |
| `duration` |  8 | variable | signed int of length N as ns                   |
| `time`     | 9 | variable | signed int of length N as ns since epoch       |
| `float16`  | 10 |    2     | 2 bytes of IEEE 64-bit format                  |
| `float32`  | 11 |    4     | 4 bytes of IEEE 64-bit format                  |
| `float64`  | 12 |    8     | 8 bytes of IEEE 64-bit format                  |
| `decimal`  | 13 |  4,8,16  | N bytes of IEEE decimal format                 |
| `bool`     | 14 |    1     | one byte 0 (false) or 1 (true)                 |
| `bytes`    | 15 | variable | N bytes of value                               |
| `string`   | 16 | variable | UTF-8 byte sequence defined by ZSON            |
| `bstring`  | 17 | variable | UTF-8 byte sequence defined by ZSON           |
| `ip`       | 18 | 4 or 16  | 4 or 16 bytes of IP address                    |
| `net`      | 19 | 8 or 32  | 8 or 32 bytes of IP prefix and subnet mask     |
| `type`     | 20 | variable | type value byte sequence [as defined below](#4-type-values) |
| `error`    | 21 | variable | UTF-8 byte sequence of string of error message |
| `null`     | 22 |    0     | No value, always represents an undefined value |

> TBD: For decimal, ZSON assumes decimal128 and there's not a way to specify
> different widths.  This should be okay since a 128 will hold the others.

## 4. Type Values

As the ZSON data model support first-class types and because the ZNG design goals
require that value serializations cannot change across type contexts, type values
must be encoded in a fashion that is independent of the type context.
Thus, a serialized type value encodes the entire type in a canonical form
according to the recursive definition in this section.

The type value of a primitive type (include type `type`) is its primitive ID,
serialized as a single byte.

The type value of a complex type is serialized recursively according to the
complex type it represents as described below.

#### 4.1 Record Type Value

A record type value has the form:
```
-----------------------------------------------------
|0x19|<ncolumns>|<name1><typeval><name2><typeval>...|
-----------------------------------------------------
```
where `<ncolumns>` is the number of columns in the record encoded as a `uvarint`,
`<name1>` etc. are the field names encoded as in the
record typedef, and each `<typeval>` is a recursive encoding of a type value.

#### 4.2 Array Type Value

An array type value has the form:
```
----------------
|0x20|<typeval>|
----------------
```
where `<typeval>` is a recursive encoding of a type value.

#### 4.3 Set Type Value

An set type value has the form:
```
----------------
|0x21|<typeval>|
----------------
```
where `<typeval>` is a recursive encoding of a type value.

#### 4.4 Union Type Value

A union type value has the form:
```
-------------------------------------
|0x22|<ntypes>|<typeval><typeval>...|
-------------------------------------
```
where `<ntypes>` is the number of types in the union encoded as a `uvarint`
and each `<typeval>` is a recursive definition of a type value.

#### 4.5 Enum Type Value

An enum type value has the form:
```
--------------------------------
|0x23|<nelem>|<name1><name2>...|
--------------------------------
```
where `<nelem>` and each symbol name are encoded as in an enum typedef.

#### 4.6 Map Type Value

A map type value has the form:
```
----------------------------
|0x23|<key-type>|<val-type>|
----------------------------
```
where `<key-type>` and `<val-type>` are recursive encodings of type values.

#### 4.7 Alias Type Value

An alias type value may appear either as a definition or a reference.
When an alias appears in reference form, it must have been previously
defined in the type value in accordance with a left-to-right depth-first-search (DFS)
traversal of the type.

An alias type value definition has the form:
```
----------------------
|0x17|<name><typeval>|
----------------------
```
where `<name>` is encoded as in an alias typedef
and `<typeval>` is a recursive encoding of a type value.  This creates
a binding between the given name and the indicated type value only within the
scope of the encoded value and does not affect the type context.
This binding may be changed by another type alias
definition of the same name in the same type value according to the DFS order.

An alias type value reference has the form:
```
-------------
|0x18|<name>|
-------------
```
It is an error for an alias reference to appear in a type value with a name
that has not been previously defined according to the DFS order.

## Appendix A. Related Links

* [Zeek ASCII logging](https://docs.zeek.org/en/master/log-formats.html#zeek-tsv-format-logs)
* [Binary logging in Zeek](https://old.zeek.org/development/projects/binary-logging.html)
* [Hadoop sequence file](https://cwiki.apache.org/confluence/display/HADOOP2/SequenceFile)
* [Avro](https://avro.apache.org)
* [Parquet](https://en.wikipedia.org/wiki/Apache_Parquet)
* [Protocol Buffers](https://developers.google.com/protocol-buffers)
* [MessagePack](https://msgpack.org/index.html)
* [gNMI](https://github.com/openconfig/reference/tree/master/rpc/gnmi)

## Appendix B. Recommended Type Coercion Rules

> TBD: it might be better to put this in the ZQL docs

While outside the scope of the ZNG format specification, we include here
some suggested rules for converting types when mixed-type operations occur,
e.g., adding an uint32 field to an int32 fields or aggregating a stream of
mixed-type numeric values in a sum operator.

The age old question is "does unsigned(-1) equal unsigned(maxint) and thus
that signed -1 is larger than unsigned 1 in a coerced comparison?"
The standard SQL specification goes so far as to avoid unsigned types altogether
to avoid this confusion.  However, since unsigned types are prevalent
in the real world, and we want ZNG to be a reliable and complete language-independent
model for communicating structured data, ZNG embraces the unsigned type.

Given the dynamic typing nature of ZNG streams (e.g., x in one record might
be a uint8, in another an int64, and in still another, a string), type coercion
is important for ergonomic use, and implementations are thus encouraged
to handle mixed-type operations robustly.

For systems that perform analytics directly on ZNG, the following coercion
patterns are recommend for logical comparisons of numbers, arithmetic operations,
or streaming aggregations over numbers:
* For float32 and float64, the float32 is converted to float64.
* For float32 and any integer type, the integer is converted to float32
and any loss of precision causes no error.
* For float64 and any integer type, the integer is converted to float64
and any loss of precision causes no error.
* For integers of same signed-ness but different widths, the smaller width
type is converted to the wider type.
* For any signed and unsigned integers smaller than 64 bits, the unsigned value
is converted to the corresponding signed type if possible, and otherwise,
both are converted to the widest signed type that will allow conversion of
the unsigned value unless the unsigned value cannot be converted,
in which case an overflow error occurs.   e.g., uint8(255) and int8(-1), are
converted to int16(255) and int16(-1), but uint64(2^32) and any signed value
will result in overflow.
* For a time or duration with a number, automatic coercion is not performed
and casts or conversion functions should be used.
* For a string with number, automatic coercion is not performed
and casts or conversion functions should be used.

Also,
* numeric constants should be int64 or float64 unless cast, which means
comparisons with constants will generally be coerced to these types and results
of mathematical operations with constants will be promoted as well;
* times and durations may be added, resulting in a time;
* times may be subtracted, resulting in a duration; and,
* a "plus" operator applied to two strings, implies concatenation,
but a "plus" applied to a string and a non-string is a type mismatch and casts
or conversion functions should be used.
