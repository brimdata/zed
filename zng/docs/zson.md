# ZSON - ZNG Structured Object-record Notation

> ### DRAFT 11/30/20
> ### Note: This specification is in BETA development.
> We plan to have a final form established by Spring 2021.
>
> ZSON is intended as a literal superset of JSON syntax (not semantics).
> If there is anything in this document that leads to this not being the
> case, please let us know and we will address it.
>
> Regarding the state of the implementation of the ZSON format in zq:
> * An input reader is not yet implemented,
> * End-of-sequence markers are not yet conveyed in the writer.

> TBD: add a ToC and section numbering

ZSON is a data model and serialization format that embodies both
"structured" as well as "semi-structured" data.  It is
semantically a superset of the schema-flexible document model of JSON
and the schema-rigid table model of relational systems.

ZSON has the look and feel of JSON but differs substantially
under the covers as it enjoys:
* a comprehensive, embedded type system with first-class types and definitions,
* well-defined streaming semantics, and
* a semantically-matched, performant binary form called ZNG.

ZSON builds upon the elegant simplicity of JSON with "type decorators".
These decorators appear throughout the value syntax unobstrusively,
in an ergonomic and human-readable syntax, to establish a well-defined
type of every value expressed in a ZSON input.

In addition to the human-readable format described here, the ZSON data model
is realized in an efficient binary format called
[ZNG](spec.md) and a columnar variation
of ZNG called [ZST](../../zst/README.md).
While ZNG and ZST are suited for production workflows and operational systems
where performance matters most,
ZSON is appropriate for human-level inspection of raw data
and for test and debug and for low-performance APIs where ergonomics matters
more than performance.

The ZSON design was motivated by and [is compatible with](./zeek-compat.md) the
[Zeek log format](https://docs.zeek.org/en/stable/examples/logs/).
As far as we know, the Zeek log format pioneered the concept of
embedding the schemas of log lines as metadata within the log files themselves
and ZSON modernizes this original approach with a JSON-like syntax.

## Some Examples

The simplest ZSON text is a single value, perhaps a string like this:
```
"hello, world"
```
There's no need for a type decorator here.  It's explicitly a string.

A structured SQL table might look like this:
```
{ city: "Berkeley", state: "CA", population:121643 (uint32) } (=schema)
{ city: "Broad Cove", state: "ME", population:806 } (schema)
{ city: "Baton Rouge", state: "LA", population:221599 } (schema)
```
This ZSON text depicts three record values.  It defines a type called `schema`
based on the first value and decorates the two subsequent values with that type.
The implied value of the `schema` record type is:
```
{ city:string, state:string, population:uint32 }
```
When all the values have the same record type, the of values can be interpreted
as a _table_, where the ZSON record values are _rows_ and the fields of
the records form _columns_.

In contrast, a ZSON text representing a semi-structured sequence of log lines
might look like this:
```
{
    info: "Connection Example",
    src: { addr: 10.1.1.2, port:80 (uint16) } (=socket),
    dst: { addr: 10.0.1.2, port:20130 } (socket)
} (=conn)
{
    info: "Connection Example 2",
    src: { addr: 10.1.1.8, port:80 },
    dst: { addr: 10.1.2.88, port:19801 }
} (conn)
{
    info: "Access List Example",
    nets: [ 10.1.1.0/24, 10.1.2.0/24 ]
} (=access_list)
{ metric: "A", ts:2020-11-24T08:44:09.586441-08:00, value:120 }
{ metric: "B", ts:2020-11-24T08:44:20.726057-08:00, value:0.86 }
{ metric: "A", ts:2020-11-24T08:44:32.201458-08:00, value:126 }
{ metric: "C", ts:2020-11-24T08:44:43.547506-08:00, value:{ x:10, y:101 } }
```
In this case, the first record value defines not just the new record type
called `conn`, but also a second embedded record type called `socket`.
The parenthesized decorators are used where a type is not gleaned from
the value itself:
* `socket` is a record with typed fields `addr` and `port` where `port` is an unsigned 16-bit integer, and
* `conn` is a record with typed fields `info`, `src`, and `dst`.

The next value in the sequence is also a `conn` type but given that this
complex type has been defined previously, there is no need to specify the
type decorators for each nested field since they are all determined by the
top-level record type.

The subsequent value defines a type called `access_list`.  In this case,
the `nets` field is an array of networks and illustrates the helpful range of
primitive types in ZSON.  Note that the syntax here implies
the type of the array, as its inferred from the type of the elements.

Finally, there are four more values that show the ZSON's efficacy for
representing metrics.  Here, there is no type decorators as all of the field
types are implied by their syntax, and hence, the top-level record type is implied.
For instance, the `ts` field is an RFC-3339 date/time string,
unambiguously the primitive type `time`.  Further,
note that the `value` field takes on different types and even a complex record
type on the last line.  In this case, there is a different type top-level
record type implied by each of the three variations of type of the `value` field.

## The ZSON Data Model

ZSON data is defined as an ordered sequence of one or more typed data values
separated at any point or not at all by an end-of-sequence marker `.`.
Each sequence implies a type context as described below and the `.` marker causes
a new type context to be established for subsequent elements of the sequence.

Each value's type is either a "primitive type", a "complex type", the "type type",
or the "null type".

### Primitive Types

Primitive types include signed and unsigned integers, IEEE floating point of
several widths, IEEE decimal,
string, bstring, byte sequence, boolean, IP address, and IP network.

> Note: The `bstring` type is an unusual type representing a hybrid type
> mixing a UTF-8 string with embedded binary data.  This type is
> useful in systems that, for instance, pull data off the network
> while expecting a string, but there can be embedded binary data due to
> bugs, malicious attacks, etc.  It is up to the receiver to determine
> with out-of-band information or inference whether the data is ultimately
> arbitrary binary data or a valid UTF-8 string.

### Complex Types

Complex types are composed of primitive types and/or other complex types
and include
* _record_ - an ordered collection of zero or more named values called fields,
* _array_ - an ordered sequence of zero or more values called elements,
* _set_ - a set of zero or more unique values called elements,
* _union_ - a type representing values whose type is any of a specified collection of two or more types,
* _enum_ - a type representing values taken from a specified collection of one or more uniformly
typed values, which can be either primitive or complex, and
* _map_ - a collection of zero or more key/value pairs where the keys are of a
uniform type called the key type and the values are of a uniform type called
the value type.

### The Type Type

ZSON also includes first-class types, where a value can be of type `type`.
The syntax for type values is given below.

### The Null Type

The _null_ type is a primitive type representing only a `null` value.  A `null`
value can have any type.

Any value in ZSON can take on a null representation.  It is up to an
implementation to decide how external data structures map into and
out of values with nulls.  Typically, a null value means either the
zero value or in the case of record fields, it means the field is optional
and the value is not present, though these semantics are not explicitly
defined by ZSON.

### Type Definitions and Contexts

Type definitions embedded in the data sequence associate a name with a type
so that later values can refer to their type by name instead of explicitly
enumerating the type of every element.  A collection of mappings from name to
type is called a "type context."  As bindings are read from a sequence, they
create the sequence's type context.  The context may be reset at any point in
the sequence, implying that a new binding must be defined from that point on
for each unique use of a type.

A type name is either internal or external.  Internal names are used exclusively
the organize the types in the type context within the data sequence itself and have
no meaning outside of that data.  External names are visible outside of
data sequence as named types providing external systems a way to
discover and refer to types by name where the type is defined within the data
sequence.  The binding between an external name and its type must be reestablished
in each type context in which it used.  External names are analogous to the notion
of "logical types" in Parquet and Avro.

Internal type names are represented by integer names while external names are
represented by identifiers.  When ZSON data is organized into a sequence
comprised of two or more subsequences where each subsequence has its own
type context, the internal names may be "reused" across type contexts to refer
to different types but external names must have the same type value across
subsequences.  It is a "type mismatch" error if an external name has different
type values across subsequences.  It is also a "type mismatch" error if any
any type binding

## The ZSON Format

A ZSON text is a sequence of UTF-8 characters organized either as a bounded input
or as or an unbounded stream.  Like NDJSON, ZSON input represents a sequence of
data objects that can be incrementally parsed and is human readable,
though ZSON values must be individually parsed to find their boundaries
as they are not new-line delimited.

A ZSON UTF-8 input is organized as a
sequence of one or more values optionally separated by and interspersed
with arbitrary and ignored whitespace.
Single line `//` comments and multi-line comments `/* ... */` are
treated as whitespace.

All subsequent references to characters and strings in this section refer to
the Unicode code points that result when the stream is decoded.
If a ZSON input includes data that is not valid UTF-8, the input is invalid.

### Identifiers

ZSON identifiers are used in several contexts:
* unquoted field names,
* names of enum elements, and
* external type definition names,

Identifiers are case-sensitive and can contain Unicode letters, `$`, `_`,
and digits (0-9), but may not start with a digit.

### Type Decorators

Any value may be explicitly typed by tagging it with a type decorator.
The syntax for a decorator is a parenthesized type, as in
```
<value> ( <decorator> )
```
where a `<decorator>` is either an integer representing an internal type
or a ZSON type value.  Note that an internal type is not a ZSON type value.

It is an error for the decorator to be type incompatible with its referenced value.  

> TBD: define precisely the notion of type compatibility

#### Type Definitions

Type names are defined by binding a name to a type with an assignment decorator
of the form
```
<value> (= <identifier> )
```
This creates a new type whose name is given as the identifier whose value
is the type value of `<value>`.  This new
type may then be used as or within a type value or decorator.  The name
of the type definition
must not be equal to any of the primitive type names.

The value of a type definition is the given value whose type value is
given by `<identifier>` as the newly defined type.

It is an error for an external type to be defined to a different type
than its previously definition though multiple definitions of the same
type are legal (thereby allowing for concatenation of otherwise
indepdent sequences).

### Primitive Values

There are 23 types of primitive values with syntax defined as follows:

| Type       | Value Syntax                                                  |
|------------|---------------------------------------------------------------|
| `uint8`    | decimal string representation of any unsigned, 8-bit integer  |
| `uint16`   | decimal string representation of any unsigned, 16-bit integer |
| `uint32`   | decimal string representation of any unsigned, 32-bit integer |
| `uint64`   | decimal string representation of any unsigned, 64-bit integer |
| `int8`     | decimal string representation of any signed, 8-bit integer    |
| `int16`    | decimal string representation of any signed, 16-bit integer   |
| `int32`    | decimal string representation of any signed, 32-bit integer   |
| `int64`    | decimal string representation of any signed, 64-bit integer   |
| `duration` | a _duration string_ representing signed 64-bit nanoseconds |
| `time`     | an RFC-3339 UTC data/time string representing signed 64-bit nanoseconds from epoch |
| `float16`  | a _point string_ representing an IEEE-754 binary16 value |
| `float32`  | a _point string_ representing an IEEE-754 binary32 value |
| `float64`  | a _point string_ representing an IEEE-754 binary64 value |
| `decimal`  | a _point string_ representing an IEEE-754 decimal128 value |
| `bool`     | the string `true` or `false` |
| `bytes`    | a sequence of bytes encoded as a hexadecimal string prefixed with `0x` |
| `string`   | a double-quoted UTF-8 string |
| `bstring`  | a doubled-quoted UTF-8 string with `\x` escapes of non-UTF binary data |
| `ip`       | a string representing an IP address in [IPv4 or IPv6 format](https://tools.ietf.org/html/draft-main-ipaddr-text-rep-02#section-3) |
| `net`      | a string in CIDR notation representing an IP address and prefix length as defined in RFC 4632 and RFC 4291. |
| `type`     | a type value encoded according to Section [TBD] |
| `error`    | a UTF-8 byte sequence of string of error message|
| `null`     | the string `null` |

The format of a _duration string_
is an optionally-signed sequence of decimal numbers,
each with optional fraction and a unit suffix,
such as "300ms", "-1.5h" or "2h45m".
Valid time units are "ns", "us" (or "µs"), "ms", "s", "m", "h".

The format of a _point string_ is the subset of strings defined by the
JSON specification for a number that include the character `.` (thereby
explicitly differentiating from integers) or one of the strings:
`+Inf`, `-Inf`, or `Nan`.

Of the 23 primitive types, ten of them represent _implied-type_ values:
`int64`, `time`, `float64`, `bool`, `bytes`, `string`, `ip`, `net`, `type`, and `null`.
Values for these types are determined by the syntax of the value and
thus do not need decorators to clarify the underlying type, e.g.,
```
123 (int64)
```
is the same as `123`.

Values that do not have implied types must include a type decorator to clarify
its type or appear in a context for which its type is defined (i.e., as a field
value in a record, as an element in an array, etc).

> Note that `time` values correspond to 64-bit epoch nanoseconds and thus
> not all possible RFC-3339 date/time strings are valid.  In addition,
> nanosecond epoch times overflow on April 11, 2262.
> For the world of 2262, a new epoch can be created well in advance
> and the old time epoch and new time epoch can live side by side with
> the old using a typedef for the new epoch time aliased to the old `time`.
> An app that wants more than 64 bits of timestamp precision can always use
> a typedef to a `bytes` type and do its own conversions to and from the
> corresponding bytes values.

#### String Escape Rules

Double-quoted `string` syntax is the same as that of JSON as described
[RFC-7159](https://tools.ietf.org/html/rfc7159), specifically:

* The sequence `\uhhhh` where each `h` is a hexadecimal digit represents
  the Unicode code point corresponding to the given
  4-digit (hexadecimal) number, or:
* `\u{h*}` where there are from 1 to 6 hexadecimal digits inside the
  brackets represents the Unicode code point corresponding to the given
  hexadecimal number.

`\u` followed by anything that does not conform to the above syntax
is not a valid escape sequence.  The behavior of an implementation
that encounters such invalid sequences in a `string` type is undefined.

These escaping rules apply also to quoted field names in record values and
record types.

Doubled-quoted `bstring` values may also included embedded binary data
using the hex escape syntax, i.e., `\xhh` where `h` is a hexadecimal digit.
This allows binary data that does not conform to a valid UTF-8 character encoding
to be embedded in the `bstring` data type.
`\x` followed by anything other than two hexadecimal digits is not a valid
escape sequence. The behavior of an implementation that encounters such
invalid sequences in a `bstring` type is undefined.

Additionally, the backslash character itself (U+3B) may be represented
by a sequence of two consecutive backslash characters.  In other words,
the `bstring` values `\\` and `\x3b` are equivalent and both represent
a single backslash character.

### Complex Values

Complex values are built from primitive values and/or other complex values
and each conform to one of six complex types:  _record_, _array_, _set_,
_union_, _enum_, and _map_.

#### Record Value

A record value has the following syntax:
```
{ <name> : <value>, <name> : <value>, ... }
```
where `<name>` is either an identifier or a quoted string and `<value>` is
any optionally-decorated ZSON value inclusive of other records.
There may be zero or more key/values pairs.

Any of the field values of a record may have an ambiguous type, in which case
the record value must include a decorator, e.g., of the form
```
{ <name> : <value>, <name> : <value>, ... } ( <decorator> )
```

#### Array Value

An array value has the following syntax:
```
[ <value>, <value>, ... ]
```
A type decorator applied to an array must be an array type.
If the elements of the array are not of uniform type, then the implied type of
the array elements is a union of the types ordered in the sequence they are encountered.

An array value may be empty.  An empty array value without a type decorator is
presumed to be an empty array of type `null`.

#### Set Value

A set value has the following syntax:
```
|[ <value>, <value>, ... ]|
```
where the indicated values must be distinct.
A type decorator applied to a set must be a set type.
If the elements of the set are not of uniform type, then the implied type of
the set elements is a union of the types ordered in the sequence they are encountered.

A set value may be empty.  An empty set value without a type decorator is
presumed to be an empty set of type `null`.

#### Union Value

A union value is simply a value that conforms with a union type.
If the value appears in a context in which the type has not been established,
then the value must be decorated with the union type.

A union type decorator is simply a list of the types that comprise the union
and has the form
```
( <type> , <type> ... )
```
where there are at least two types in the comma separator list of types.

For example,
```
"hello, world" (int32, string)
```
is the `string` value of union type comprising the sub-types `int32` and `string`
Likewise, this is an integer value of the same union:
```
123 (int32, string)
```
Where there is ambiguity, a decorator may resolve the ambiguity as in
```
123 (int8) (int32, int8)
```

> TBD: list the ambiguous possibilities so this is clear.  I think they are
> float16, float32, (u)int8, (u)int16, (u)int32, and uint64 as well as union
> values inside of union values.

#### Enum Value

An enum type represents a named set of values where enum values are
referenced by name.  The simple form of an
enum is a list of names, where the values are of type `int64` but
enums may be defined as set of values of an arbitrary type.

An enum value has the form
```
<identifier>
```
where the indicated identifier is the name of one of the enum elements.

Such a value must appear in a context where the enum type is known, i.e.,
with an explicit enum type decorator or within a complex type where the
contained enum type is defined by the complex type.

When a string `true`, `false`, or `null` is intended as an enum value,
a type decorator of the corresponding enum type is required to resolve
the ambiguity, e.g.,
```
true (<true,false,unknown>)
```
is an enum value of the indicated enum and is not the boolean value `true`.

#### Map Value

A map value has the following syntax:
```
|{ {<key>, <value>}, {<key>, <value>}, ... }|
```
A type decorator applied to an map can either be one element, referring to a
map type, or two elements referring to the type of the keys and type of the values.

A map value may be empty.  An empty map value without a type decorator is
presumed to be an empty map of type (`null`, `null`).

### Type Values

The syntax of a type value or of a decorator expressed as a type value mirrors
the value syntax.

A primitive type value is the name of the primitive type, i.e., `string`,
`uint16`, etc.

The syntax of complex type values parallels the syntax of complex values.

A _record type value_ has the form:
```
{ <name> : <type>, <name> : <type>, ... }
```
where `<type>` is any type value.  The order of the columns is significant,
e.g., type `{a:int32,b:int32}` is distinct from type `{b:int32,a:int32}`.
In contrast to schema abstractions in other formats, ZSON has no way to mark
a field optional as all fields are, in a sense optional: any field can be
encoded with a null value.  If an instance of a record value omits a value
by dropping the field altogether rather than using a null, then that record
value corresponds to a different record type that elides the field in question.

An _array type value_ has the form:
```
[ <type> ]
```

A _set type value_ has the form:
```
|[ <type> ]|
```

A _map type value_ has the form:
```
|{ <key-type>, <value-type> }|
```
where `<key-type>` is any type value of the keys and `<value-type>` is any
type value of the values.

A _union type value_ has the form:
```
( <type>, <type>, ... )
```
where there are at least two types in the list.

An _enum type value_ has two forms.  The simple form is:
```
< <identifier>, <identifier>, ... >
```
and the complex form is:
```
< <identifier>:<value>, <identifier>=<value>, ... >
```
In the same form, the underlying enum value is equal to its positional
index in the list of identifiers as an uint64 type.

## Discussion and Examples

> TBD: Add a range of helpful examples and also some that cover the more obscure corner cases.
> Example of pico-second timestamp.

### zeek
```
{
    _path: "conn",
    ts: 2018-03-24T17:15:20.600725Z,
    uid: "C1zOivgBT6dBmknqk" (bstring),
    id: {
        orig_h: 10.47.1.152,
        orig_p: 49562 (uint16) (=port),
        resp_h: 23.217.103.245,
        resp_p: 80 (port)
    } (=24),
    proto: "tcp" (=zenum),
    service: null (bstring),
    duration: 9.698493s (duration),
    orig_bytes: 0 (uint64),
    resp_bytes: 90453565 (uint64),
    conn_state: "SF" (bstring),
    local_orig: null (bool),
    local_resp: null (bool),
    missed_bytes: 0 (uint64),
    history: "^dtAttttFf" (bstring),
    orig_pkts: 57490 (uint64),
    orig_ip_bytes: 2358856 (uint64),
    resp_pkts: 123713 (uint64),
    resp_ip_bytes: 185470730 (uint64),
    tunnel_parents: null (|[bstring]| (=25))
} (=26)
{
    _path: "conn",
    ts: 2018-03-24T17:15:20.605945Z,
    uid: "CayJxr1WvNLdJ6L9B4",
    id: {
        orig_h: 10.128.0.207,
        orig_p: 8,
        resp_h: 10.47.23.178,
        resp_p: 0
    },
    proto: "icmp",
    service: null,
    duration: 4µs,
    orig_bytes: 0,
    resp_bytes: 0,
    conn_state: "OTH",
    local_orig: null,
    local_resp: null,
    missed_bytes: 0,
    history: null,
    orig_pkts: 2,
    orig_ip_bytes: 56,
    resp_pkts: 0,
    resp_ip_bytes: 0,
    tunnel_parents: null
} (26)
```

### enum

```
{ rank: A (<2,3,4,5,6,7,8,9,10,Jack,Queen,King,Ace>), suit: H (<Hearts,Diamonds,Spaces,Clubs>) } (=card)
```
is the same as
```
{ rank: A, suit:H } ({rank: <2,3,4,5,6,7,8,9,10,Jack,Queen,King,Ace>, suit:<Hearts,Diamonds,Spaces,Clubs>}) (=card)
```

### union

```
{ u: 12 (int32, string, {a:int8,b:ip} (=27)), |[27]| } (=28) (=ex_union)
{ u: "foo" } (=ex_union)
```
is the same as
```
{ u: 12 (int32, string, {a:int8,b:ip} (=27)), |[27]| } (=28) (=rec_with_union)
{ u: "foo" } (rec_with_union)
```

If you have a set of two different types inside of a union that "look the same" then you don't
know which is which so you need a type decorator to distinguish....
```
{ u: |[12, 13 ]| (int8) (=26) (26, |[int16]| (=27)) (=28) } (=29) (=union_ex2)
{ u: |[14, 15 ]| (27) } (union_ex2)
```
is the same as
```
{ u: |[12, 13 ]| (int8) (|[int8]|, |[int16]| } (=29) (=union_ex2)
{ u: |[14, 15 ]| (int16) } (union_ex2)
```
