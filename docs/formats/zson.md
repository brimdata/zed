---
sidebar_position: 2
sidebar_label: ZSON
---

# ZSON - Zed Super-structured Open Notation

* [1. Introduction](#1-introduction)
* [2. The ZSON Format](#2-the-zson-format)
  + [2.1 Names](#21-names)
  + [2.2 Type Decorators](#22-type-decorators)
  + [2.3 Primitive Values](#23-primitive-values)
    - [2.3.1 Strings](#231-strings)
  + [2.4 Complex Values](#24-complex-values)
    - [2.4.1 Record Value](#241-record-value)
    - [2.4.2 Array Value](#242-array-value)
    - [2.4.3 Set Value](#243-set-value)
    - [2.4.4 Map Value](#244-map-value)
    - [2.4.5 Union Value](#245-union-value)
    - [2.4.6 Enum Value](#246-enum-value)
    - [2.4.7 Error Value](#247-error-value)
  + [2.5 Types](#25-types)
    - [2.5.1 Record Type](#251-record-type)
    - [2.5.2 Array Type](#252-array-type)
    - [2.5.3 Set Type](#253-set-type)
    - [2.5.4 Map Type](#254-map-type)
    - [2.5.5 Union Type](#255-union-type)
    - [2.5.6 Enum Type](#256-enum-type)
    - [2.5.7 Named Type](#257-named-type)
    - [2.5.8 Error Type](#258-error-type)
  + [2.6 Null Value](#26-null-value)
* [3. Examples](#3-examples)
* [4. Grammar](#4-grammar)

## 1. Introduction

ZSON is the human-readable, text-based serialization format of
the super-structured [Zed data model](zed.md).

ZSON builds upon the elegant simplicity of JSON with "type decorators".
Where the type of a value is not implied by its syntax, a parenthesized
type decorator is appended to the value thus establishing a well-defined
type for every value expressed in ZSON text.

ZSON is also a superset of JSON in that all JSON documents are valid ZSON values.

## 2. The ZSON Format

A ZSON text is a sequence of UTF-8 characters organized either as a bounded input
or an unbounded stream.

The input text is organized as a sequence of one or more Zed values optionally
separated by and interspersed with whitespace.
Single-line (`//`) and multi-line (`/* ... */`) comments are
treated as whitespace and ignored.

All subsequent references to characters and strings in this section refer to
the Unicode code points that result when the stream is decoded.
If a ZSON input includes data that is not valid UTF-8, the input is invalid.

### 2.1 Names

ZSON _names_ encode record fields, enum symbols, and named types.
A name is either an _identifier_ or a [quoted string](#231-strings).
Names are referred to as `<name>` below.

An _identifier_ is case-sensitive and can contain Unicode letters, `$`, `_`,
and digits (0-9), but may not start with a digit.  An identifier cannot be
`true`, `false`, or `null`.

### 2.2 Type Decorators

A value may be explicitly typed by tagging it with a type decorator.
The syntax for a decorator is a parenthesized type:
```
<value> ( <type> )
```
For union values, multiple decorators might be
required to distinguish the union-member type from the possible set of
union types when there is ambiguity, as in
```
123 (float64) (int64,float64)
```
In contrast, this union value is unambiguous:
```
123.0 (int64,float64)
```

The syntax of a union value decorator is
```
<value> ( <type> ) [ ( <type> ) ...]
```
where the rightmost type must be a union type if more then one decorator
is present.

A decorator may also define a [named type](#257-named-type):
```
<value> ( =<name> )
```
which declares a new type with the indicated type name using the
implied type of the value.

It is an error for the decorator to be type incompatible with its referenced value.  

Note that the `=` sigil here disambiguates between the case that a new
type is defined, which may override a previous definition of a different type with the
same name, from the case that an existing named type is merely decorating the value.

### 2.3 Primitive Values

The type names and format for
[Zed primitive](zed.md#1-primitive-types) values is as follows:

| Type       | Value Format                                                  |
|------------|---------------------------------------------------------------|
| `uint8`    | decimal string representation of any unsigned, 8-bit integer  |
| `uint16`   | decimal string representation of any unsigned, 16-bit integer |
| `uint32`   | decimal string representation of any unsigned, 32-bit integer |
| `uint64`   | decimal string representation of any unsigned, 64-bit integer |
| `uint128`   | decimal string representation of any unsigned, 128-bit integer |
| `uint256`   | decimal string representation of any unsigned, 256-bit integer |
| `int8`     | decimal string representation of any signed, 8-bit integer    |
| `int16`    | decimal string representation of any signed, 16-bit integer   |
| `int32`    | decimal string representation of any signed, 32-bit integer   |
| `int64`    | decimal string representation of any signed, 64-bit integer   |
| `int128`    | decimal string representation of any signed, 128-bit integer   |
| `int256`    | decimal string representation of any signed, 256-bit integer   |
| `duration` | a _duration string_ representing signed 64-bit nanoseconds |
| `time`     | an RFC 3339 UTC data/time string representing signed 64-bit nanoseconds from epoch |
| `float16`  | a _non-integer string_ representing an IEEE-754 binary16 value |
| `float32`  | a _non-integer string_ representing an IEEE-754 binary32 value |
| `float64`  | a _non-integer string_ representing an IEEE-754 binary64 value |
| `float128`  | a _non-integer string_ representing an IEEE-754 binary128 value |
| `float256`  | a _non-integer string_ representing an IEEE-754 binary256 value |
| `decimal32`  | a _non-integer string_ representing an IEEE-754 decimal32 value |
| `decimal64`  | a _non-integer string_ representing an IEEE-754 decimal64 value |
| `decimal128`  | a _non-integer string_ representing an IEEE-754 decimal128 value |
| `decimal256`  | a _non-integer string_ representing an IEEE-754 decimal256 value |
| `bool`     | the string `true` or `false` |
| `bytes`    | a sequence of bytes encoded as a hexadecimal string prefixed with `0x` |
| `string`   | a double-quoted or backtick-quoted UTF-8 string |
| `ip`       | a string representing an IP address in [IPv4 or IPv6 format](https://tools.ietf.org/html/draft-main-ipaddr-text-rep-02#section-3) |
| `net`      | a string in CIDR notation representing an IP address and prefix length as defined in RFC 4632 and RFC 4291. |
| `type`     | a string in canonical form as described in [Section 3.5](#25-type-value) |
| `null`     | the string `null` |

The format of a _duration string_
is an optionally-signed concatenation of decimal numbers,
each with optional fraction and a unit suffix,
such as "300ms", "-1.5h" or "2h45m", representing a 64-bit nanosecond value.
Valid time units are
"ns" (nanosecond),
"us" (microsecond),
"ms" (millisecond),
"s" (second),
"m" (minute),
"h" (hour),
"d" (day),
"w" (7 days), and
"y" (365 days).
Note that each of these time units accurately represents its calendar value,
except for the "y" unit, which does not reflect leap years and so forth.
Instead, "y" is defined as the number of nanoseconds in 365 days.

The format of floating point values is a _non-integer string_
conforming to any floating point representation that cannot be
interpreted as an integer, e.g., `1.` or `1.0` instead of
`1` or `1e3` instead of `1000`.  Unlike JSON, a floating point number can
also be one of:
`Inf`, `+Inf`, `-Inf`, or `Nan`.

A floating point value may be expressed with an integer string provided
a type decorator is applied, e.g., `123 (float64)`.

Decimal values require type decorators.

A string may be backtick-quoted with the backtick character `` ` ``.
None of the text between backticks is escaped, but by default, any newlines
followed by whitespace are converted to a single newline and the first
newline of the string is deleted.  To avoid this automatic deletion and
preserve indentation, the backtick-quoted string can be preceded with `=>`.

Of the 30 primitive types, eleven of them represent _implied-type_ values:
`int64`, `time`, `duration`, `float64`, `bool`, `bytes`, `string`, `ip`, `net`, `type`, and `null`.
Values for these types are determined by the format of the value and
thus do not need decorators to clarify the underlying type, e.g.,
```
123 (int64)
```
is the same as `123`.

Values that do not have implied types must include a type decorator to clarify
its type or appear in a context for which its type is defined (i.e., as a field
value in a record, as an element in an array, etc.).

While a type value may represent a complex type, the value itself is a singleton
and thus always a primitive type.  A type value is encoded as:
* a left angle bracket `<`, followed by
* a type as [encoded below](#25-types), followed by
* a right angle bracket `>`.

A `time` value corresponds to 64-bit Unix epoch nanoseconds and thus
not all possible RFC 3339 date/time strings are valid.  In addition,
nanosecond epoch times overflow on April 11, 2262.
For the world of 2262, a new epoch can be created well in advance
and the old time epoch and new time epoch can live side by side with
the old using a named type for the new epoch time defined as the old `time` type.
An app that requires more than 64 bits of timestamp precision can always use
a typedef of a `bytes` type and do its own conversions to and from the
corresponding bytes values.

#### 2.3.1 Strings

Double-quoted `string` syntax is the same as that of JSON as described
in [RFC 8259](https://tools.ietf.org/html/rfc8259#section-7).  Notably,
the following escape sequences are recognized:

| Sequence | Unicode Character      |
|----------|------------------------|
| `\"`     | quotation mark  U+0022 |
| `\\`     | reverse solidus U+005C |
| `\/`     | solidus         U+002F |
| `\b`     | backspace       U+0008 |
| `\f`     | form feed       U+000C |
| `\n`     | line feed       U+000A |
| `\r`     | carriage return U+000D |
| `\t`     | tab             U+0009 |
| `\uXXXX` |                 U+XXXX |

In `\uXXXX` sequences, each `X` is a hexadecimal digit, and letter
digits may be uppercase or lowercase.

The behavior of an implementation that encounters an unrecognized escape
sequence in a `string` type is undefined.

`\u` followed by anything that does not conform to the above syntax
is not a valid escape sequence.  The behavior of an implementation
that encounters such invalid sequences in a `string` type is undefined.

These escaping rules apply also to quoted field names in record values and
record types as well as enum symbols.

### 2.4 Complex Values

Complex values are built from primitive values and/or other complex values
and conform the Zed data model's complex types:
[record](zed.md#21-record),
[array](zed.md#22-array),
[set](zed.md#23-set),
[map](zed.md#24-map),
[union](zed.md#25-union),
[enum](zed.md#26-enum), and
[error](zed.md#27-error).

Complex values have an implied type when their constituent values all have
implied types.

#### 2.4.1 Record Value

A record value has the form:
```
{ <name> : <value>, <name> : <value>, ... }
```
where `<name>` is a [ZSON name](#21-names) and `<value>` is
any optionally-decorated ZSON value inclusive of other records.
Each name/value pair is called a _field_.
There may be zero or more fields.

#### 2.4.2 Array Value

An array value has the form:
```
[ <value>, <value>, ... ]
```
If the elements of the array are not of uniform type, then the implied type of
the array elements is a union of the types present.

An array value may be empty.  An empty array value without a type decorator is
presumed to be an empty array of type `null`.

#### 2.4.3 Set Value

A set value has the form:
```
|[ <value>, <value>, ... ]|
```
where the indicated values must be distinct.

If the elements of the set are not of uniform type, then the implied type of
the set elements is a union of the types present.

A set value may be empty.  An empty set value without a type decorator is
presumed to be an empty set of type `null`.

#### 2.4.6 Map Value

A map value has the form:
```
|{ <key> : <value>, <key> : <value>, ... }|
```
where zero or more comma-separated, key/value pairs are present.

Whitespace around keys and values is generally optional, but to
avoid ambiguity, whitespace must separate an IPv6 key from the colon
that follows it.

An empty map value without a type decorator is
presumed to be an empty map of type `|{null: null}|`.

#### 2.4.4 Union Value

A union value is a value that conforms to one of the types within a union type.
If the value appears in a context in which the type is unknown or ambiguous,
then the value must be decorated as [described above](#22-type-decorators).

#### 2.4.5 Enum Value

An enum type represents a symbol from a finite set of symbols
referenced by name.

An enum value is indicated with the sigil `%` and has the form
```
%<name>
```
where the `<name>` is [ZSON name](#21-names).

An enum value must appear in a context where the enum type is known, i.e.,
with an explicit enum type decorator or within a complex type where the
contained enum type is defined by the complex type's decorator.

A sequence of enum values might look like this:
```
%HEADS (flip=(enum(HEADS,TAILS)))
%TAILS (flip)
%HEADS (flip)
```

#### 2.4.7 Error Value

An error value has the form:
```
error(<value>)
```
where `<value>` is any ZSON value.

### 2.5 Types

A primitive type is simply the name of the primitive type, i.e., `string`,
`uint16`, etc.  Complex types are defined as follows.

#### 2.5.1 Record Type

A _record type_ has the form:
```
{ <name> : <type>, <name> : <type>, ... }
```
where `<name>` is a [ZSON name](#21-names) and
`<type>` is any type.

The order of the record fields is significant,
e.g., type `{a:int32,b:int32}` is distinct from type `{b:int32,a:int32}`.

#### 2.5.2 Array Type

An _array type_ has the form:
```
[ <type> ]
```

#### 2.5.3 Set Type

A _set type_ has the form:
```
|[ <type> ]|
```

#### 2.5.6 Map Type

A _map type_ has the form:
```
|{ <key-type>: <value-type> }|
```
where `<key-type>` is the type of the keys and `<value-type>` is the
type of the values.

#### 2.5.4 Union Type

A _union type_ has the form:
```
( <type>, <type>, ... )
```
where there are at least two types in the list.

#### 2.5.5 Enum Type

An _enum type_ has the form:
```
enum( <name>, <name>, ... )
```
where `<name>` is a [ZSON name](#21-names).
Each enum name must be unique and the order is not significant, e.g.,
enum type `enum(HEADS,TAILS)` is equal to type `enum(TAILS,HEADS)`.

#### 2.5.7 Named Type

A named type has the form:
```
<name> = ( <type> )
```
where a new type is defined with the given name and type.

When a named type appears in a complex value, the new type name may be
referenced by any subsequent value in left-to-right depth-first order.

For example,
```
{p1:80 (port=(uint16)), p2: 8080 (port)}
````
is valid but
```
{p1:80 port, p2: 8080 (port=(uint16))}
````
is invalid.

Named types may be redefined, in which case subsequent references
resolve to the most recent definition according to
* sequence order across values, or
* left-to-right depth-first order within a complex value.

#### 2.5.8 Error Type

An _error type_ has the form:
```
error( <type> )
```
where `<type>` is the type of the underlying ZSON values wrapped as an error.

### 2.6 Null Value

The null value is represented by the string `null`.

A value of any type can be null.  It is up to an
implementation to decide how external data structures map into and
out of null values of different types.  Typically, a null value means either the
zero value or, in the case of record fields, an optional field whose
value is not present, though these semantics are not explicitly
defined by ZSON.

## 3. Examples

The simplest ZSON value is a single value, perhaps a string like this:
```
"hello, world"
```
There's no need for a type declaration here.  It's explicitly a string.

A relational table might look like this:
```
{ city: "Berkeley", state: "CA", population: 121643 (uint32) } (=city_schema)
{ city: "Broad Cove", state: "ME", population: 806 (uint32) } (=city_schema)
{ city: "Baton Rouge", state: "LA", population: 221599 (uint32) } (=city_schema)
```
This ZSON text here depicts three record values.  It defines a type called `city_schema`
and the inferred type of the `city_schema` has the signature:
```
{ city:string, state:string, population:uint32 }
```
When all the values in a sequence have the same record type, the sequence
can be interpreted as a _table_, where the ZSON record values form the _rows_
and the fields of the records form the _columns_.  In this way, these
three records form a relational table conforming to the schema `city_schema`.

In contrast, a ZSON text representing a semi-structured sequence of log lines
might look like this:
```
{
    info: "Connection Example",
    src: { addr: 10.1.1.2, port: 80 (uint16) } (=socket),
    dst: { addr: 10.0.1.2, port: 20130 (uint16) } (=socket)
} (=conn)
{
    info: "Connection Example 2",
    src: { addr: 10.1.1.8, port: 80 (uint16) } (=socket),
    dst: { addr: 10.1.2.88, port: 19801 (uint16) } (=socket)
} (=conn)
{
    info: "Access List Example",
    nets: [ 10.1.1.0/24, 10.1.2.0/24 ]
} (=access_list)
{ metric: "A", ts: 2020-11-24T08:44:09.586441-08:00, value: 120 }
{ metric: "B", ts: 2020-11-24T08:44:20.726057-08:00, value: 0.86 }
{ metric: "A", ts: 2020-11-24T08:44:32.201458-08:00, value: 126 }
{ metric: "C", ts: 2020-11-24T08:44:43.547506-08:00, value: { x:10, y:101 } }
```
In this case, the first records defines not just a record type
with named type `conn`, but also a second embedded record type called `socket`.
The parenthesized decorators are used where a type is not inferred from
the value itself:
* `socket` is a record with typed fields `addr` and `port` where `port` is an unsigned 16-bit integer, and
* `conn` is a record with typed fields `info`, `src`, and `dst`.

The subsequent value defines a type called `access_list`.  In this case,
the `nets` field is an array of networks and illustrates the helpful range of
primitive types in ZSON.  Note that the syntax here implies
the type of the array, as it is inferred from the type of the elements.

Finally, there are four more values that show ZSON's efficacy for
representing metrics.  Here, there are no type decorators as all of the field
types are implied by their syntax, and hence, the top-level record type is implied.
For instance, the `ts` field is an RFC 3339 date and time string,
unambiguously the primitive type `time`.  Further,
note that the `value` field takes on different types and even a complex record
type on the last line.  In this case, there is a different type top-level
record type implied by each of the three variations of type of the `value` field.

## 4. Grammar

Here is a left-recursive pseudo-grammar of ZSON.  Note that not all
acceptable inputs are semantically valid as type mismatches may arise.
For example, union and enum values must both appear in a context
the defines their type.

```
<zson> = <zson> <eos> <dec-value> | <zson> <dec-value> | <dec-value>

<eos> = .

<value> = <any> | <any> <val-typedef> | <any> <decorators>

<val-typedef> = "(" "=" <name> ")"

<decorators> = "(" <type> ")" | <decorators> "(" <type> ")"

<any> = <primitive> | <type-val> | <record> | <array> | <set> | <map> | <enum>

<primitive> = primitive value as defined above

<record> = "{" <flist> "}"  |  "{"  "}"

<flist> = <flist> "," <field> | <field>

<field> = <name> ":" <value>

<name> = <identifier> | <quoted-string>

<quoted-string> = quoted string as defined above

<identifier> = as defined above

<array> = "[" <vlist> "]"  |  "["  "]"

<vlist> = <vlist> "," <value> | <value>

<set> = "|[" <vlist> "]|"  |  "|["  "]|"

<enum> = "%" ( <name> | <quoted-string> )

<map> = "|{" <mlist> "}|"  |  "|{"  "}|"

<mlist> = <mvalue> | <mlist> "," <mvalue>

<mvalue> = <value> ":" <value>

<type-value> = "<" <type> ">"

<error-value> = "error(" <value> ")"

<type> = <primitive-type> | <record-type> | <array-type> | <set-type> |
            <union-type> | <enum-type> | <map-type> |
            <type-def> | <name> | <error-type>

<primitive-type> = uint8 | uint16 | etc. as defined above

<record-type> = "{" <tflist> "}"  |  "{" "}"

<tflist> = <tflist> "," <tfield> | <tfield>

<tfield> = <name> ":" <type>

<array-type> = "[" <type> "]"  |  "[" "]"

<set-type> = "|[" <type> "]|"  |  "|[" "]|"

<union-type> = "(" <type> "," <tlist> ")"

<tlist> = <tlist> "," <type> | <type>

<enum-type> = "enum(" <nlist> ")"

<nlist> = <nlist> "," <name> | <name>

<map-type> = "{" <type> "," <type> "}"

<type-def> = <identifier> = <type-type>

<name> = as defined above

<error-type> = "error(" <type> ")"
```
