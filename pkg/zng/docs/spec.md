# ZNG Specification

> NOTE: This specification is a work in progress and is in "ALPHA".
> Zq's implementation of ZNG is not up to date with this latest description.

ZNG is a format for structured data values, ideally suited for streams
of heterogeneously typed records.  ZNG has both a text form simply called "ZNG",
comprised of a sequence of newline-delimited UTF-8 strings,
as well as a binary form called "BZNG".

ZNG is richly typed and thinner on the wire than JSON.
Like [newline-delimited JSON (NDJSON)](http://ndjson.org/),
the ZNG text format represents a sequence of data objects
that can be parsed line by line.

ZNG strikes a balance between the narrowly typed but flexible NDJSON format and
a more structured approach like
[Apache Avro](https://avro.apache.org).
ZNG is type rich and embeds all type/schema in the stream while having a
binary serialization format that allows "lazy parsing" of fields such that
only the fields of interest in a stream need to be extracted and parsed.
Unlike Avro,
ZNG embeds type information in the data stream and admits efficient
multiplexing of heterogenous data types by including with each data value
a simple integer identifier to reference its type.

ZNG is a superset of JSON in that any JSON input
can be mapped onto ZNG and recovered by decoding
that ZNG back into JSON.

The ZNG design [is also motivated by](./rationale.md)
and maintains backward compatibility with the original [default Zeek log format](https://docs.zeek.org/en/stable/examples/logs/).

## The ZNG data model

ZNG encodes a sequence of one or more typed data values comprising a stream.
The stream of values is interleaved with control messages
that provide type definitions for the data values.  The type of
a particular data value is specified by its "type code", which is an
integer identifier representing either a built-in type or
or a composite type definition occurring previously in the stream.

A type alias can specify a name for any type, e.g., the zeek log type "count"
can be aliased to ZNG type "uint64".

ZNG is designed for efficient use as a protocol for streaming values between
end systems and thus allows control messages to carry arbitrary data payloads as
signaling from a higher-layer protocol that is embedded in the ZNG stream.

The ZNG type system comprises the standard set of types like integers, floating point,
strings, byte arrays, etc as well as composite types build from the
standard types including records, arrays, and sets.

For example, a ZNG stream representing the single string "hello world"
looks like this:
```
9:hello, world
```
Here the type code is the integer "9" represents the string type, and the data
value "hello, world" is an instance of string.

ZNG gets more interesting when data types are interleaved in the stream.
For example,
```
9:hello, world
4:42
9:there's a fly in my soup!
9:no, there isn't.
4:3
```
where type code 4 represents an integer.  This encoding represents the string of
values:
```
"hello, world"
42
"there's a fly in my soup!"
"no, there isn't.""
3
```

Often, ZNG streams are comprised as a sequence of records, which works well to provide
an efficient representation of structured logs.  In this case, a new type code is
needed to define the schema for each distinct record.  To define a new
type, the "#" syntax is used.  For example,
logs from the open-source zeek system might look like this
```
#23:ip=addr
#24:record[_path:string,ts:time,uid:string,id:record[orig_h:addr,orig_p:port,resp_h:addr,resp_p:port]...
#25:record[_path:string,ts:time,uid:string,id:record[orig_h:addr,orig_p:port,resp_h:addr,resp_p:port]...
24:conn;1425565514.419939;CogZFI3py5JsFZGik;[192.168.1.1:;80/tcp;192.168.1.2;8080;]...
25:dns;1425565514.419987;CogZFI3py5JsFZGik;[192.168.1.1:;5353/udp;192.168.1.2;5353;]...
```
Note that the value encoding need not refer to the field names and types as that is
completely captured by the type code.  Values merely encode the value
information consistent with the referenced type code.

## ZNG Binary Format (BZNG)

The ZNG text format is defined in terms of the binary format BZNG.  So before
going into those details, the BZNG format is defined.

The BZNG binary format is based on machine-readable data types and was
inspired by [Protol Buffers](https://developers.google.com/protocol-buffers).
The data model described above is serialized into a stream of bytes.
The byte stream encodes a sequence of messages.

Each message is prefixed with a single-byte header code.  The upper bit of
the header code indicates whether the message is a control message (1)
or a value (0).

In the case of control message, the lower 7 bits define the control type.
Control type 0 is a "type definition" or typedef described below.

Control code 1 is reserved for the ordering hint.  All other control codes
are available to higher-layer protocols for carrying protocol-specific payloads
embedded in the ZNG stream.

Control messages may be used informatively and shall be
ignored by any data receivers.  The message can be any UTF-8 string.
Control payloads are guaranteed to be preserved
in order within the stream and presented to higher layer components through
any ZNG streaming API.  In this way, senders and receivers of ZNG can embed
protocol directives as ZNG control payloads rather than defining additional
encapsulating protocols.  See the
[zng-over-http](zng-over-http.md) protocol for an example.

### BZNG Types

Following a header byte of 0x80 is a "type definition" (typedef) that binds
"the next available" integer type code to a type encoding.  Type codes
begin at the value 23 and increase by one for each typedef. This bindings
are scoped to the stream in which the typedef occurs.

Type codes for the "scalar types" are predefined as follows:

| Type     | Code |
|----------|------|
| bool     |   0  |
| byte     |   1  |
| int16    |   2  |
| uint16   |   3  |
| int32    |   4  |
| uint32   |   5  |
| int64    |   6  |
| uint64   |   7  |
| float64  |   8  |
| string   |   9  |
| bytes    |  10  |
| bstring  |  11  |
| enum     |  12  |
| ip       |  13  |
| port     |  14  |
| net      |  15  |
| time     |  16  |
| duration |  17  |
| any      |  18  |

#### BZNG typedef

A typedef defines a new composite type (i.e., record, array, or set)
from the composite type code plus the composite type-specific encoding of the
new type.

The composite type codes are:

| Type     | Code |
|----------|------|
| record   |   0  |
| array    |   1  |
| set      |   2  |

Note that the composite type codes overlap with the scalar type codes because their
uses never overlap.  Composite type codes can only appear at the beginning of
a typedef.  Thus, the space of referenceable type codes in a given stream is
the union of the scalar type codes and the type codes created with typedefs.

A typedef is encoded as a single byte indicating the composite type code following by
the type encoding.  This creates a binding between the implied type code
(i.e., 23 plus the count of all previous typedefs in the stream) and the new
type definition.

A "uvarint" is an unsigned integer encoded according to the protobuf specification.

#### Record Type Encoding

Record types consist of an ordered set of columns where each column consists of
a name and a typed value.  Unlike JSON, the ordering of the columns is significant
and must be preserved through any APIs that consume, process, and emit ZNG records.

A record type is encoded as a count of fields followed by the field definitions,
where a field definition is a field name followed by a type code.

The field names in a record must be unique.

The count of fields is encoded as a uvarint.

The field name is encoded as a UTF8 string defining a "ZNG identifier"
The UTF8 string
is further encoded as a "counted string", which is uvarint encoding
of the length of the string followed by that many bytes of UTF8 encoded
string data.

```
------------------------------------------------------------------
|0x80|0x00|<num-fields>|<field1><typecode1><field2><typecode2>...|
------------------------------------------------------------------
```

N.B.: The rules for ZNG identifiers follow the same rules as
[JavaScript identifiers](https://tc39.es/ecma262/#prod-IdentifierName).

The type code follows the field name and is encoded as a uvarint.

A record may contain zero columns, in which case the only legal value
for such a type is unset (see below).  An unset record with zero columns
corresponds to a JSON empty object when translating ZNG to JSON and
vice versa.

#### Array Type Encoding

An array type is encoded as simply the type code of the elements of
the array encoded as a uvarint.

```
-----------------------
|0x80|0x01|<type-code>|
-----------------------
```

#### Set Type Encoding

A set type is encoded as simply the type code of the elements of
the array encoded as a uvarint.

```
-----------------------
|0x80|0x02|<type-code>|
-----------------------
```


XXX we need to get a handle on zeek multi-typed sets and define an
encoding for this...

### BZNG Values

Following a header byte with bit 7 zero is a "typed value"
with a modified uvarint encoding of the type code.

A "typed value" is encoded as a uvarint encoding the length of the
variable length type code plus the length of the value,
followed by a uvarint representing the type code and the
remaining N bytes of value.  N is calculated by subtracting
the length of the type code from the counted-length value encoding.
The type code indicates how the value should be decoded and interpreted
given its predetermined length N.

For a modified type code, the uppermost bit of the first byte of the uvarint
isn't used be used so only the lower 7 bits of the first byte of the
encoding are used in the otherwise standard protobuf uvarint definition.
(* this needs more explanation)

```
----------------------------------------
|<modified-uvarint>|<type-code>|<value>|
----------------------------------------
```

A typed value of length N is interpreted as follows:

| Type     | N        |              Value                           |
|----------|----------|----------------------------------------------|
| bool     | 1        |  one byte 0 (false) or 1 (true)              |
| byte     | 1        |  the byte                                    |
| int16    | variable |  signed int of length N                      |
| uint16   | variable |  unsigned int of length N                    |
| int32    | variable |  signed int of length N                      |
| uint32   | variable |  unsigned int of length N                    |
| int64    | variable |  signed int of length N                      |
| uint64   | variable |  unsigned int of length N                    |
| float64  | 8        |  8 bytes of IEEE 64-bit format               |
| string   | variable |  UTF-8 byte sequence of string               |
| bytes    | variable |  bytes of value                              |
| bstring  | variable |  UTF-8 byte sequence with \x escapes         |
| enum     | variable |  UTF-8 bytes of enum string                  |
| ip       | 4 or 16  |  4 or 16 bytes of IP address                 |
| net      | 8 or 32  |  8 or 32 bytes of IP prefix and subnet mask  |
| time     | 4        |  4 bytes of signed nanoseconds from epoch    |
| duration | 4        |  4 bytes of signed nanoseconds duration      |
| any      | variable |  <uvarint type code><value as defined here>  |

All multi-byte sequences are machine words are serialized in
little-endian format.

Array, set, and record types are variable length and are encoded
as a sequence of elements:

| Type     |          Value            |
|----------|---------------------------|
| array    | concatenation of elements |
| set      | concatenation of elements |
| record   | concatenation of elements |

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
and is encoded as a uvarint.

For example, tag value 0 is an unset scalar value and tag value 1
is an unset composite value.  Tag value 2 is a length zero scalar value,
e.g., it could represent empty string.  Tag 3 is a length 1 scalar value,
e.g., it would represent the boolean "true" if followed by byte value 1
in the context of type code 0.

Following the tag encoding is the value encoded in N bytes as described above.

## ZNG Text Format (ZNG)

The ZNG text format is a human-readable form that follows directly from the BZNG
binary format.  A stream of control messages and values messages is represented
as a sequence of UTF-8 lines each terminated by a newline.  Any newlines embedded
in values are escaped via "\n".

A line that begins with "#" a control message.

Any line that begins with a decimal integer and has the form
```
<integer>:<value text>
```
is a value.

A typedef control message has the form
```
#<type encoding>
```
i.e., there is a single "#" character followed by a type encoding establishing
a binding between the "next available" type code and the type defined by the
encoding.  For readability, the type code may be included in the typedef
with the following syntax:
```
#<integer>:<type encoding>
```
In this case, the integer type code must match the next implied type code.
This form is useful for test cases, debugging etc.

All other control messages have the form
```
#!<control code>:<payload>
```
where <control code> is a decimal integer in the range 1-127 and <payload>
is any UTF-8 string with escaped newlines.

Any line beginning that does not conform with the syntax described here
is an error.
When errors are encountered parsing ZNG, an implementation should return a
corresponding error and allow ZNG parsing to proceed if desired.

### ZNG Ordering Hint

The ordering directive has the following structure:
```
#!1:[+-]<field>,[+-]<field>,...
```
where `[+-]` indicates either `+` or `-` and `<field>` refers to the top-level
field name in a record of any subsequent regular or legacy value.
This directive guarantees that all subsequent value lines will
appear sorted in the file or stream, in ascending order in the case of `+` and
descending order in the case of `-`, according to the field provided.
If more than one sort
field is provided, then the values are guaranteed to be sorted by each
subsequent key for values that have previous keys of equal value.

It is an error for any such values to appear that contradicts the most
recent ordering directives.

### ZNG Typedefs

A text type encoding has one of three forms that corresponds directly with the
binary type encoding, a format each for:
* record,
* array, and
* set.

#### Record Type Encoding

A record type has the following syntax:
```
record[name1:code1,name2:code2,...]
```
where name1, name2, ... are field names as defined by the BZNG record type definitions
and code1, code2, ... are textual type codes.  A textual type code can be either the name
of a scalar type code from the type code table (e.g., "int64", "time", etc) or a
string decimal integer referring to said table or created with a typedef that appeared
earlier in the ZNG data stream.

#### Array Type Encoding

An array type has the following syntax:
```
array[code]
```
where code is a text type code defined above.

#### Set Type Encoding

An set type has the following syntax:
```
set[code]
```
where code is a text type code defined above.

XXX we need to handle multi-typed sets

#### Typedef Examples
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
Composite types can be embedded in other composite types by reference
an earlier-defined type code:
```
#26:record[a:string,b:string,c:23]
#27:set[26]
#28:record[v:23,s:24,r:25,s2:27]
```

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
[zq/pkg/zeek](../../zeek).


### ZNG Values

A ZNG textual value is encoded on a line as typed value, i.e.,
an integer type code followed by `:` followed
by a value encoding.  Here is a pseudo-grammar for value encodings:
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
a <terminal> can have the form <typecode>:<elem> for values of type "any".

Composite values are encoded as
* an open bracket,
* zero or more encoded values terminated with semicolon, and
* a close bracket.

Any value can be specified as "unset" with the ascii character `-`.
This is typically used to represent columns of records where not all
columns have been set in a given record value, though any type can be
validly unset.  A value that is not to be interpreted as "unset"
but is the single-character string `-`, must be escaped (e.g., `\-`).

Note that this syntax can be scanned and parsed independent of the
actual type definition indicated by the descriptor (unlike legacy values,
which parse set and vector values differently).  It is a semantic error
if the parsed value does not match the indicated type in terms of number and
sub-structure of value elements present and their interpretation as a valid
string of the specified type.

It is an error for a value to include a descriptor that has not been previously
defined by a descriptor directive.

### Character Escape Rules

Any character in a value line may be escaped from the ZSON formatting rules
using the hex escape syntax, i.e., `\xhh` where h is a hexadecimal digit.

Sequences of binary data can be embedded in values using these escapes but it is
a semantic error for arbitrary binary data to be carried by any types except
`string` and `bytes` (see [Type Semantics](#type-semantics)).

These special characters must be hex escaped if they appear within a value:
```
; \n \\
```
And these characters must be escaped if they appear as the first character
of a value:
```
[ ]
```
In addition, `-` must be escaped if representing the single ASCII byte equal
to `-` as opposed to representing an unset value.

## Value Syntax

Each UTF-string field parsed from a value line is interpreted according to the
type descriptor of the line.
The formats for each type is as follows:

Type | Format
---- | ------
`bool` | a single characeter `T` or `F`
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
`bstring` | a UTF-8 string with \x escapes of non-UTF binary data
`enum` | a string representing an enumeration value defined outside the scope of ZNG
`ip` | a string representing an IP address in [IPv4 or IPv6 format](https://tools.ietf.org/html/draft-main-ipaddr-text-rep-02#section-3)
`net` | a string in CIDR notation representing an IP address and prefix length as defined in RFC 4632 and RFC 4291.
`time` | unsigned dotted decimal notation of seconds (32-bit second, 32-bit nanosecond)
`duration` | signed dotted decimal notation of seconds (32-bit second, 32-bit nanosecond)
`any` | integer type code and colon followed by a value as defined here

* Note: A `bstring` can embed binary data using escapes.  It's up to the receiver to determine
with out-of-band information whether the data is ultimately arbitrary binary data or
a valid UTF-8 string.

## Examples

Here is a simple example to get the gist of this encoding.  This ZNG defines
a the first implied type code (23) and references the string type code (9):
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



## Related Links

* [Zeek ASCII logging](https://docs.zeek.org/en/stable/examples/logs/)
* [Binary logging in Zeek](https://www.zeek.org/development/projects/binary-logging.html)
* [Hadoop sequence file](https://cwiki.apache.org/confluence/display/HADOOP2/SequenceFile)
* [Avro](https://avro.apache.org)
* [Parquet](https://en.wikipedia.org/wiki/Apache_Parquet)
* [Protobufs](https://developers.google.com/protocol-buffers)
* [MessagePack](https://msgpack.org/index.html)
* [gNMI](https://github.com/openconfig/reference/tree/master/rpc/gnmi)
