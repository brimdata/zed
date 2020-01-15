# ZNG Specification

> Note: This specification is ALPHA and a work in progress.
> Zq's implementation of ZNG is not yet up to date with this latest description.

ZNG is a format for structured data values, ideally suited for streams
of heterogeneously typed records, e.g., structured logs, where filtering and
analytics may be applied to a stream in parts without having to fully deserialize
every value.

ZNG has both a text form simply called "ZNG",
comprised of a sequence of newline-delimited UTF-8 strings,
as well as a binary form called "BZNG".

ZNG is richly typed and thinner on the wire than JSON.
Like [newline-delimited JSON (NDJSON)](http://ndjson.org/),
the ZNG text format represents a sequence of data objects
that can be parsed line by line.
ZNG strikes a balance between the narrowly typed but flexible NDJSON format and
a more structured approach like
[Apache Avro](https://avro.apache.org).

ZNG is type rich and embeds all type information in the stream while having a
binary serialization format that allows "lazy parsing" of fields such that
only the fields of interest in a stream need to be deserialized and interpreted.
Unlike Avro, ZNG embeds type information in the data stream and thereby admits
an efficient multiplexing of heterogeneous data types by prepending to each
data value a simple integer identifier to reference its type.   

ZNG requires no external schema definitions as its type system
constructs schemas on the fly from within the stream using composable,
dynamic type definitions.  Given this, there is no need for
a schema registry service, though ZNG can be readily adapted to systems like
[Apache Kafka](https://kafka.apache.org/) which utilize such registries,
by having a connector translate the schemas implied in the
ZNG stream into registered schemas and vice versa.

ZNG is a superset of JSON in that any JSON input
can be mapped onto ZNG and recovered by decoding
that ZNG back into JSON.

> TBD: ZNG is also interchangeable with Avro...

The ZNG design [was motivated by](./zeek-compat.md)
and is compatible with the
[Zeek log format](https://docs.zeek.org/en/stable/examples/logs/).
As far as we know, the Zeek log format pioneered the concept of
embedding the schema of the log lines within the log file itself using
meta-records, and ZNG merely modernizes this original approach.

## 1. The ZNG data model

ZNG encodes a sequence of one or more typed data values to comprise a stream.
The stream of values is interleaved with control messages
that provide type definitions and other metadata.  The type of
a particular data value is specified by its "type code", which is an
integer identifier representing either a built-in type or
a dynamic type definition that occurred previously in the stream.

The ZNG type system comprises the standard set of scalar types like integers,
floating point, strings, byte arrays, etc. as well as composite types
like records, arrays, and sets arranged from the scalar types.

For example, a ZNG stream representing the single string "hello world"
looks like this:
```
9:hello, world
```
Here, the type code is the integer "9" representing the string type
(defined in [Typedefs](#typedefs)) and the data value "hello, world"
is an instance of string.

ZNG gets more interesting when different data types are interleaved in the stream.
For example,
```
9:hello, world
4:42
9:there's a fly in my soup!
9:no, there isn't.
4:3
```
where type code 4 represents an integer.  This encoding represents the sequence of
values:
```
"hello, world"
42
"there's a fly in my soup!"
"no, there isn't."
3
```
ZNG streams are often comprised as a sequence of records, which works well to provide
an efficient representation of structured logs.  In this case, a new type code is
needed to define the schema for each distinct record.  To define a new
type, the "#" syntax is used.  For example,
logs from the open-source Zeek system might look like this
```
#alias:addr=ip
#24:record[_path:string,ts:time,uid:string,id:record[orig_h:addr,orig_p:port,resp_h:addr,resp_p:port]...
#25:record[_path:string,ts:time,fuid:string,tx_hosts:set[addr]...
24:[conn;1425565514.419939;CogZFI3py5JsFZGik;[192.168.1.1:;80/tcp;192.168.1.2;8080;]...
25:[files;1425565514.419987;Fj8sRF1gdneMHN700d;[52.218.49.89;52.218.48.169;]...
```
Note that the value encoding need not refer to the field names and types as that is
completely captured by the type code.  Values merely encode the value
information consistent with the referenced type code.

## 2. ZNG Binary Format (BZNG)

The BZNG binary format is based on machine-readable data types with an
encoding methodology inspired by
[Protocol Buffers](https://developers.google.com/protocol-buffers).

A BZNG stream comprises a sequence of interleaved control messages and value messages
that are serialized into a stream of bytes.

Each message is prefixed with a single-byte header code.  The upper bit of
the header code indicates whether the message is a control message (1)
or a value message (0).

### 2.1 Control Messages

The lower 7 bits of a control header byte define the control code.
Control codes 0 through 4 are reserved for BZNG:

| Code | Message Type      |
|------|-------------------|
|  `0` | record definition |
|  `1` | array definition  |
|  `2` | set definition    |
|  `3` | type alias        |
|  `4` | ordering hint     |

All other control codes are available to higher-layer protocols to carry
application-specific payloads embedded in the ZNG stream.

Any such application-specific payloads not known by
a ZNG data receiver shall be ignored.

The body of an application-specific control message is any UTF-8 string.
These payloads are guaranteed to be preserved
in order within the stream and presented to higher layer components through
any ZNG streaming API.  In this way, senders and receivers of ZNG can embed
protocol directives as ZNG control payloads rather than defining additional
encapsulating protocols.  See the
[zng-over-http](zng-over-http.md) protocol for an example.

### <a name="typedefs"> 2.1.1 Typedefs

Following a header byte of 0x80-0x83 is a "typedef".  A typedef binds
"the next available" integer type code to a type encoding.  Type codes
begin at the value 23 and increase by one for each typedef. These bindings
are scoped to the stream in which the typedef occurs.

Type codes for the "scalar types" need not be defined with typedefs and
are predefined as follows:

<table>
<tr><td>

| Type       | Code |
|------------|------|
| `bool`     |   0  |
| `byte`     |   1  |
| `int16`    |   2  |
| `uint16`   |   3  |
| `int32`    |   4  |
| `uint32`   |   5  |
| `int64`    |   6  |
| `uint64`   |   7  |
| `float64`  |   8  |
| `string`   |   9  |

</td><td>

| Type       | Code |
|------------|------|
| `bytes`    |  10  |
| `bstring`  |  11  |
| `enum`     |  12  |
| `ip`       |  13  |
| `port`     |  14  |
| `net`      |  15  |
| `time`     |  16  |
| `duration` |  17  |
| `any`      |  18  |
| &nbsp;     |      |

</td></tr> </table>

A typedef is encoded as a single byte indicating the composite type code following by
the type encoding.  This creates a binding between the implied type code
(i.e., 23 plus the count of all previous typedefs in the stream) and the new
type definition.

The type code is encoded as a `uvarint`, an encoding used throughout the BZNG format.

> Inspired by Protocol Buffers,
> a `uvarint` is an unsigned, variable-length integer encoded as a sequence of
> bytes consisting of N-1 bytes with bit 7 clear and the Nth byte with bit 7 set,
> whose value is the base-128 number composed of the digits defined by the lower
> 7 bits of each byte from least-significant digit (byte 0) to
> most-significant digit (byte N-1).

#### 2.1.1.1 Record Typedef

A record typedef creates a new type code equal to the next stream type code
with the following structure:
```
----------------------------------------------------------
|0x80|<nfields>|<field1><typecode1><field2><typecode2>...|
----------------------------------------------------------
```
Record types consist of an ordered set of columns where each column consists of
a name and a typed value.  Unlike JSON, the ordering of the columns is significant
and must be preserved through any APIs that consume, process, and emit ZNG records.

A record type is encoded as a count of fields, i.e., `<nfields>` from above,
followed by the field definitions,
where a field definition is a field name followed by a type code, i.e.,
`<field1>` followed by `<typecode1>` etc. as indicated above.

The field names in a record must be unique.

The `<nfields>` is encoded as a `uvarint`.

The field name is encoded as a UTF-8 string defining a "ZNG identifier"
The UTF-8 string
is further encoded as a "counted string", which is `uvarint` encoding
of the length of the string followed by that many bytes of UTF-8 encoded
string data.

N.B.: The rules for ZNG identifiers follow the same rules as
[JavaScript identifiers](https://tc39.es/ecma262/#prod-IdentifierName).

The type code follows the field name and is encoded as a `uvarint`.

A record may contain zero columns.

#### 2.1.1.2 Array Typedef

An array type is encoded as simply the type code of the elements of
the array encoded as a `uvarint`:
```
------------------
|0x81|<type-code>|
------------------
```

#### 2.1.1.3 Set Typedef

A set type is encoded as a concatenation of the type codes that comprise
the elements of the set where each type code is encoded as a `uvarint`:
```
----------------------------------
|0x82|<type-code1><type-code2>...|
----------------------------------
```

#### 2.1.1.4 Alias Typedef

A type alias defines a new type code that binds a new type name
to a previously existing type code.  This is useful for systems like Zeek,
where there are customary type names that are well-known to users of the
Zeek system and are easily mapped onto a BZNG type having a different name.
By encoding the aliases in the format, there is no need to configure mapping
information across different systems using the format, as the type aliases
are communicated to the consumer of a BZNG stream.

A type alias is encoded as follows:
```
------------------------
|0x83|<name><type-code>|
------------------------
```
where `<name>` is an identifier representing the new type name with a new type code
allocated as the next available type code in the stream that refers to the
existing type code ``<type-code>``.  ``<type-code>`` is encoded as a `uvarint` and `<name>`
is encoded as a `uvarint` representing the length of the name in bytes,
followed by that many bytes of UTF-8 string.

### 2.1.2 Ordering Hint

An ordering hint provides a means to indicate that data in the stream
is sorted a certain way.

The hint is encoded as follows:
```
---------------------------------------
|0x84|<len>|[+-]<field>,[+-]<field>,...
---------------------------------------
```
where the payload of the message is a length-counted UTF-8 string.
`<len>` is a `uvarint` indicating the length in bytes of the UTF-8 string
describing the ordering hint.

In the hint string, `[+-]` indicates either `+` or `-` and `<field>` refers
to the top-level field name in a record of any subsequent record value encountered
from thereon in the stream with the field names specified.
The hint guarantees that all subsequent value lines will
appear sorted in the file or stream, in ascending order in the case of `+` and
descending order in the case of `-`, according to the field provided.
If more than one sort
field is provided, then the values are guaranteed to be sorted by each
subsequent key for values that have previous keys of equal value.

It is an error for any such values to appear that contradicts the most
recent ordering directives.

### 2.2 BZNG Value Messages

Following a header byte with bit 7 zero is a `typed value`
with a `uvarint7` encoding its length.

> A `uvarint7` is the same as a `uvarint` except only 7 bits instead of 8
> are available in the first byte.  Its value is equal to the lower 6-bits if bit 6
> of the first byte is 1; otherwise it is that value plus the value of the
> subsequent `uvarint` times 64.

A `typed value` is encoded as either a `uvarint7` (in a top-level value message)
or `uvarint` (for any other values)
encoding the length in bytes of the type code and value followed by
the body of the typed value comprising that many bytes.
Within the body of the typed value,
the type code is encoded as a `uvarint` and the value is encoded
as a byte array whose length is equal to the body length less the
length in bytes of the type code.
```
--------------------------
|uvarint7|type-code|value|
--------------------------
```

A typed value with a `value` of length N and the type indicated
is interpreted as follows:

| Type       | N        |              Value                           |
|------------|----------|----------------------------------------------|
| `bool`     | 1        |  one byte 0 (false) or 1 (true)              |
| `byte`     | 1        |  the byte                                    |
| `int16`    | variable |  signed int of length N                      |
| `uint16`   | variable |  unsigned int of length N                    |
| `int32`    | variable |  signed int of length N                      |
| `uint32`   | variable |  unsigned int of length N                    |
| `int64`    | variable |  signed int of length N                      |
| `uint64`   | variable |  unsigned int of length N                    |
| `float64`  | 8        |  8 bytes of IEEE 64-bit format               |
| `string`   | variable |  UTF-8 byte sequence of string               |
| `bytes`    | variable |  bytes of value                              |
| `bstring`  | variable |  UTF-8 byte sequence with `\x` escapes       |
| `enum `    | variable |  UTF-8 bytes of enum string                  |
| `ip`       | 4 or 16  |  4 or 16 bytes of IP address                 |
| `net`      | 8 or 32  |  8 or 32 bytes of IP prefix and subnet mask  |
| `time`     | 8        |  8 bytes of signed nanoseconds from epoch    |
| `duration` | 8        |  8 bytes of signed nanoseconds duration      |
| `any`      | variable |  <uvarint type code><value as defined here>  |

All multi-byte sequences representing machine words are serialized in
little-endian format.

> Note: The bstring type is an unusual type representing a hybrid type
> mixing a UTF-8 string with embedded binary data.   This type is
> useful in systems like Zeek where data is pulled off the network
> while expecting a string, but there can be embedded binary data due to
> bugs, malicious attacks, etc.  It is up to the receiver to determine
with out-of-band information or inference whether the data is ultimately
> arbitrary binary data or a valid UTF-8 string.

Array, set, and record types are variable length and are encoded
as a sequence of elements:

| Type     |          Value            |
|----------|---------------------------|
| `array`  | concatenation of elements |
| `set`    | concatenation of elements |
| `record` | concatenation of elements |

Since N, the byte length of
this sequence is known, there is no need to encode a count of the
elements present.  Also, since the type code is implied by the typedef
of any composite type, each value is encoded without its type code,
except for elements corresponding to type "any", which are encoded
as a "typed value" (as defined above).

The concatenation of elements is encoded as a sequence of "tag-counted" values.
A tag carries both the length information of the corresponding value as well
a "composite bit" to differentiate between scalar values and composite values
without having to refer to the implied type.  This admits an efficient implementation
for traversing the values, inclusive of recursive traversal of composite values,
whereby the inner loop need not consult and interpret the type code of each element.

The tag encodes the length N of the value and indicates whether
it is a scalar value or a composite value.
The length is offset by 1 whereby length of 0 represents an unset value
analogous to null in JSON.
The composite bit is 1 for composite values and 0 for scalar values.
The tag is defined as
```
2*(N+1) + the composite bit
```
and is encoded as a `uvarint`.

For example, tag value 0 is an unset scalar value and tag value 1
is an unset composite value.  Tag value 2 is a length zero scalar value,
e.g., it could represent empty string.  Tag 3 is a length 1 scalar value,
e.g., it would represent the boolean "true" if followed by byte value 1
in the context of type code 0.

Following the tag encoding is the value encoded in N bytes as described above.

## 3. ZNG Text Format

The ZNG text format is a human-readable form that follows directly from the BZNG
binary format.  A stream of control messages and values messages is represented
as a sequence of UTF-8 lines each terminated by a newline.  Any newlines embedded
in values must be escaped, i.e., via `\n`.

A line that begins with `#` is a control message.

Any line that begins with a decimal integer and has the form
```
<integer>:<value text>
```
is a value.

### 3.1 ZNG Control Messages

Except for typedefs, all control messages have the form
```
#!<control code>:<payload>
```
where `<control code>` is a decimal integer in the range 5-127 and `<payload>`
is any UTF-8 string with escaped newlines.

Any line beginning that does not conform with the syntax described here
is an error.
When errors are encountered parsing ZNG, an implementation should return a
corresponding error and allow ZNG parsing to proceed if desired.

### 3.1.1 ZNG Typedefs

A typedef control message has the form
```
#<type encoding>
```
i.e., there is a single `#` character followed by a type encoding establishing
a binding between the "next available" type code and the type defined by the
encoding.  For readability, the type code may be included in the typedef
with the following syntax:
```
#<integer>:<type encoding>
```
In this case, the integer type code must match the next implied type code.
This form is useful for human-readable display, test cases, debugging etc.

#### 3.1.1.1 Record Typedef

A record type has the following syntax:
```
#record[name1:code1,name2:code2,...]
```
or
```
#<type-code>:record[name1:code1,name2:code2,...]
```
where `name1`, `name2`, ... are field names as defined by the BZNG record type definitions
and `code1`, `code2`, ... are textual type codes.  A textual type code can be either the name
of a scalar type code from the type code table (e.g., `int64`, `time`, etc), a name that
is a previously defined alias, or a
string decimal integer referring to said table or created with a typedef that appeared
earlier in the ZNG data stream.

#### 3.1.1.2 Array Typedef

An array type has the following syntax:
```
#array[<code>]
```
or
```
#<type-code>:array[<code>]
```
where `<code>` is a text-format type code as defined above.

#### 3.1.1.3 Set Typedef

A set type has the following syntax:
```
#<type-code>:set[<code>]
```
where `<code>` is a text type code as defined above.

#### 3.1.1.4 Alias Typedef

An alias typedef has the following structure:
```
#alias:<type-name>=<code>
```
where `<code>` is a text type code as defined above and
`<type-name>` is an identifier with semantics as defined in Section 2.1.1.4.

### 3.1.2 Ordering Hint

An ordering hint has the following structure:
```
#order:[+-]<field>,[+-]<field>,...
```
where the string present after the colon has the same semantics as
those described in Section 2.1.2.

### Type Grammar

Given the above textual definitions and the undelying BZNG specification, a
grammar describing the textual type encodings is:
```
<stype> := bool | byte | int16 | uint16 | int32 | uint32 | int64 | uint64 | float64
         | string | bytes | bstring | enum | ip | net | time | duration | any
         | <typecode>

<ctype> :=  array [ <stype> ]
          | set [ <stype-list> ]
          | record [ <columns> ]
          | record [ ]

<stype-list> :=    <stype>
                 | <stype-list> , <stype>

<columns> :=      <column>
                | <columns> , <column>

<column> := <id> : <stype>

<id> := <id_start> <id_continue>*

<id_start> := [A-Za-z_$]

<id_continue> := <id_start> | [0-9]

<typecode> := 0 | [1-9][0-9]*
```

A reference implementation of this type system is embedded in
[zq/zng](../).


### 3.2 ZNG Value Messages

A ZNG value is encoded on a line as typed value, which is encoded as
an integer type code followed by `:`, which is in turn followed
by a value encoding.

Here is a pseudo-grammar for typed values:
```
<typed-value> := <typecode> : <elem>
<elem> :=
          <terminal> ;
        | [ <list> ]
        | [ ]
<list> :=
          <elem>
        | <list> <elem>
<terminal> := <char>*
<char> := [^][;\n\\] | <esc-sequence>
<esc-sequence> := <JavaScript character escaping rules [1]>
```

[1] - [JavaScript character escaping rules](https://tc39.es/ecma262/#prod-EscapeSequence)

A terminal value is encoded as a string of UTF-8 characters terminated
by a semicolon (which must be escaped if it appears in the value).  A composite
values is encoded as a left bracket followed by one or more values (terminal or
composite) followed by a right bracket.
Any escaped characters shall be processed and interpreted as their escaped value.

Note that a terminal encoding of a typed value is accepted by this grammar, i.e.,
a `<terminal>` can have the form `<typecode>:<elem>` for values of type `any`.

Composite values are encoded as
* an open bracket,
* zero or more encoded values terminated with semicolon, and
* a close bracket.

Any value can be specified as "unset" with the ASCII character `-`.
This is typically used to represent columns of records where not all
columns have been set in a given record value, though any type can be
validly unset.  A value that is not to be interpreted as "unset"
but is the single-character string `-`, must be escaped (e.g., `\x2d`).

Note that this syntax can be scanned and parsed independent of the
actual type definition indicated by the descriptor.  It is a semantic error
if the parsed value does not match the indicated type in terms of number and
sub-structure of value elements present and their interpretation as a valid
string of the specified type.

It is an error for a value to reference a type code that has not been previously
defined by a typedef scoped to the stream in which the value appears.

#### 3.2.1 Character Escape Rules

Any character in a `bstring` value may be escaped from the ZNG formatting rules
using the hex escape syntax, i.e., `\xhh` where `h` is a hexadecimal digit.
This allows binary data that does not conform to a valid UTF-8 character encoding
to be embedded in the `bstring` data type.

These special characters must be hex escaped if they appear within a `bstring`
or a `string` type:
```
; \n \\
```
These characters are invalid in all other data types.

Likewise, these characters must be escaped if they appear as the first character
of a value:
```
[ ]
```
In addition, `-` must be escaped if representing the single ASCII byte equal
to `-` as opposed to representing an unset value.

#### 3.2.2 Value Syntax

Each UTF-string field parsed from a value line is interpreted according to the
type descriptor of the line.
The formats for each type is as follows:

Type | Format
---- | ------
`bool` | a single character `T` or `F`
`byte` | two-characters of hexadecimal digit
`int16` | decimal string representation of any signed, 16-bit integer
`uint16` | decimal string representation of any unsigned, 16-bit integer
`int32` | decimal string representation of any signed, 32-bit integer
`uint32` | decimal string representation of any unsigned, 32-bit integer
`int64` | decimal string representation of any signed, 64-bit integer
`uint64` | decimal string representation of any unsigned, 64-bit integer
`float64` | a decimal representation of a 64-bit IEEE floating point literal as defined in JavaScript
`string` | a UTF-8 string
`bytes` | a sequence of bytes encoded as base64
`bstring` | a UTF-8 string with `\x` escapes of non-UTF binary data
`enum` | a string representing an enumeration value defined outside the scope of ZNG
`ip` | a string representing an IP address in [IPv4 or IPv6 format](https://tools.ietf.org/html/draft-main-ipaddr-text-rep-02#section-3)
`net` | a string in CIDR notation representing an IP address and prefix length as defined in RFC 4632 and RFC 4291.
`time` | signed dotted decimal notation of seconds
`duration` | signed dotted decimal notation of seconds
`any` | integer type code and colon followed by a value as defined here

## 4. Examples

Here are some simple examples to get the gist of the ZNG text format.

Scalar types look like this and do not need typedefs:
```
bool
string
int
```
Composite types look like this and do need typedefs:
```
#23:vector[int]
#24:set[bool,string]
#25:record[x:double,y:double]
```
Composite types can be embedded in other composite types by referencing
an earlier-defined type code:
```
#26:record[a:string,b:string,c:23]
#27:set[26]
#28:record[v:23,s:24,r:25,s2:27]
```
This ZNG defines
the first implied type code (23) and references the string type code (9),
using them in three values:
```
#23:record[a:string,b:string]
9:hello, world;
23:[hello;world;]
9:this is a semicolon: \x3b;
```
which represents a stream of the following three values:
```
string("hello, world")
record(a:"hello",b:"world")
string("this is a semicolon: ;")
```

The semicolon terminator is important.  Consider this ZNG depicting
sets of strings:
```
#24:set[string]
24:[hello,world;]
24:[hello;world;]
24:[]
24:[;]
```
In this example:
* the first value is a `set` of one `string`
* the second value is a `set` of two `string` values, `hello` and `world`,
* the third value is an empty `set`, and
* the fourth value is a `set` containing one `string` of zero length.

In this way, an empty `set` and a `set` containing only a zero-length `string` can be distinguished.

This scheme allows composites to be embedded in composites, e.g., a
`record` inside of a `record` like this:
```
#25:record[compass:string,degree:double]
#26:record[city:string,lat:25,long:25]
26:[NYC;[N;40.7128;][W;74.0060;]]
```
An unset value indicates a field of a `record` that wasn't set by the encoder:
```
26:[North Pole;[N;90;]-;]
```
e.g., the North Pole has a latitude but no meaningful longitude.

## 5. Related Links

* [Zeek ASCII logging](https://docs.zeek.org/en/stable/examples/logs/)
* [Binary logging in Zeek](https://www.zeek.org/development/projects/binary-logging.html)
* [Hadoop sequence file](https://cwiki.apache.org/confluence/display/HADOOP2/SequenceFile)
* [Avro](https://avro.apache.org)
* [Parquet](https://en.wikipedia.org/wiki/Apache_Parquet)
* [Protobufs](https://developers.google.com/protocol-buffers)
* [MessagePack](https://msgpack.org/index.html)
* [gNMI](https://github.com/openconfig/reference/tree/master/rpc/gnmi)
