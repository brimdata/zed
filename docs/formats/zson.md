# ZSON - Zed Structured Object-record Notation

* [1. Introduction](#1-introduction)
* [2. The ZSON Format](#2-the-zson-format)
  + [2.1 Identifiers](#21-identifiers)
  + [2.2 Type Decorators](#22-type-decorators)
    - [2.2.1 Type Definitions](#221-type-definitions)
  + [2.3 Primitive Values](#23-primitive-values)
    - [2.3.1 String Escape Rules](#231-string-escape-rules)
  + [2.4 Complex Values](#24-complex-values)
    - [2.4.1 Record Value](#241-record-value)
    - [2.4.2 Array Value](#242-array-value)
    - [2.4.3 Set Value](#243-set-value)
    - [2.4.4 Union Value](#244-union-value)
    - [2.4.5 Enum Value](#245-enum-value)
    - [2.4.6 Map Value](#246-map-value)
    - [2.4.7 Type Value](#247-type-value)
    - [2.4.7 Error Value](#248-error-value)
  + [2.5 Type Syntax](#25-type-syntax)
    - [2.5.1 Record Type](#251-record-type)
    - [2.5.2 Array Type](#252-array-type)
    - [2.5.3 Set Type](#253-set-type)
    - [2.5.4 Union Type](#254-union-type)
    - [2.5.5 Enum Type](#255-enum-type)
    - [2.5.6 Map Type](#256-map-type)
    - [2.5.7 Type Type](#257-type-type)
    - [2.5.8 Error Type](#258-error-type)
  + [2.6 Null Value](#26-null-value)
* [3. Grammar](#3-grammar)

ZSON is the human-readable, text-based serialization format of
the [Zed data model](zdm.md).

ZSON builds upon the elegant simplicity of JSON with "type decorators".
Where the type of a value is not implied by its syntax, a parenthesized
type decorator is appended to the value thus establishing a well-defined
type for every value expressed in ZSON text.

ZSON is also a superset of JSON: all JSON documents are valid ZSON values.

## 2. The ZSON Format

A ZSON text is a sequence of UTF-8 characters organized either as a bounded input
or as or an unbounded stream.  Like NDJSON, ZSON input represents a sequence of
data objects that can be incrementally parsed and is human readable,
though ZSON values must be individually parsed to find their boundaries
as they are not newline delimited.

A ZSON UTF-8 input is organized as a
sequence of one or more values optionally separated by and interspersed
with arbitrary and ignored whitespace.
Single-line (`//`) and multi-line (`/* ... */`) comments are
treated like whitespace and ignored.

All subsequent references to characters and strings in this section refer to
the Unicode code points that result when the stream is decoded.
If a ZSON input includes data that is not valid UTF-8, the input is invalid.

### 2.1 Identifiers

ZSON identifiers are used in several contexts, as names of:
* unquoted fields,
* unquoted enum symbols, and
* external type definitions.

Identifiers are case-sensitive and can contain Unicode letters, `$`, `_`,
and digits (0-9), but may not start with a digit.  An identifier cannot be
`true`, `false`, or `null`.

### 2.2 Type Decorators

A value may be explicitly typed by tagging it with a type decorator.
The syntax for a decorator is a parenthesized type, as in
```
<value> ( <decorator> )
```
where a `<decorator>` is either a type or a type definition.

It is an error for the decorator to be type incompatible with its referenced value.  

#### 2.2.1 Type Definitions

New type names are created within a Zed value by binding a name to a type with
an assignment decorator of the form
```
<value> (= <type-name> )
```
This creates a new type whose name is given by the type name and whose
type is equivalent to the type of `<value>`.  This new
type may then be referenced by other values within the same complex type.
The name of the type definition
must not be equal to any of the primitive type names.

A type definition may also appear recursively inside a decorator as in
```
<type-name> = ( <type> )
```
where the result of this expression is the newly named type.
With this syntax, you can create a new type and decorate a value
as follows
```
<value> ( <type-name> = ( <type> ) )
```
e.g.,
```
80 (port=(uint16))
````
is the value 80 of type "port", where "port" is a type name bound to `uint16`.

One decorator is allowed per value except for nested type-union values, which
may include additional decorators to successively refine the union type for union values
that live inside other union types. This allows an already-decorated value to be
further decorated with its union type and provides a means to distinguish
a union value's precise member type when it is otherwise ambiguous as described in
[Section 3.4.4](#244-union-value).

### 2.3 Primitive Values

The syntax for the primitive values defined by the Zed data model
are as follows:

| Type       | Value Syntax                                                  |
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
> not all possible RFC 3339 date/time strings are valid.  In addition,
> nanosecond epoch times overflow on April 11, 2262.
> For the world of 2262, a new epoch can be created well in advance
> and the old time epoch and new time epoch can live side by side with
> the old using a typedef for the new epoch time aliased to the old `time`.
> An app that requires more than 64 bits of timestamp precision can always use
> a typedef of a `bytes` type and do its own conversions to and from the
> corresponding bytes values.

#### 2.3.1 String Escape Rules

Double-quoted `string` syntax is the same as that of JSON as described
[RFC 8259](https://tools.ietf.org/html/rfc8259#section-7), specifically:

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

### 2.4 Complex Values

Complex values are built from primitive values and/or other complex values
and each conform to one of six complex types:  _record_, _array_, _set_,
_union_, _enum_, _map_, and _error_.

#### 2.4.1 Record Value

A record value has the following syntax:
```
{ <name> : <value>, <name> : <value>, ... }
```
where `<name>` is either an identifier or a quoted string and `<value>` is
any optionally-decorated ZSON value inclusive of other records.
There may be zero or more key/values pairs.

Any value of a field may have an ambiguous type, in which case
the record value must include a decorator, e.g., of the form
```
{ <name> : <value>, <name> : <value>, ... } ( <decorator> )
```
Similarly, a record value may omit the field names when decorated with
a record type value:
```
{ <value>, <value>, ... } ( <decorator> )
```

#### 2.4.2 Array Value

An array value has the following syntax:
```
[ <value>, <value>, ... ]
```
A type decorator applied to an array must be an array type.
If the elements of the array are not of uniform type, then the implied type of
the array elements is a union of the types ordered in the sequence they are encountered.

An array value may be empty.  An empty array value without a type decorator is
presumed to be an empty array of type `null`.

#### 2.4.3 Set Value

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

#### 2.4.4 Union Value

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
Where a union value is inside of nested union types, multiple decorators are
needed to resolve the ambiguity, e.g.,
```
"hello, world" (int32, string) ((int32,string),[int32])
```

#### 2.4.5 Enum Value

An enum type represents a symbol from a finite set of symbols
referenced by name.

An enum value is indicated with the sigil `%` and has the form
```
%<name>
```
where the `<name>` is defined as in the enum type.

Such an enum value must appear in a context where the enum type is known, i.e.,
with an explicit enum type decorator or within a complex type where the
contained enum type is defined by the complex type's decorator.

A sequence of enum values might look like this:
```
%HEADS (flip=(%{HEADS,TAILS}))
%TAILS (flip)
%HEADS (flip)
```

#### 2.4.6 Map Value

A [Zed map value](zdm.md#526-map-value) has the following ZSON syntax:
```
|{ <key> : <value>, <key> : <value>, ... }|
```
where zero or more comma-separated, key/value pairs are present.

Whitespace around keys and values is generally optional, but to
avoid ambiguity, whitespace must separate an IPv6 key from the colon
that follows it.

A empty map value without a type decorator is
presumed to be an empty map of type `|{null: null}|`.

#### 2.4.7 Type Value

The type of a type value is `type` while its value is depicted by enclosing
its type description in angle brackets (as defined in [Section 3.5](#25-type-syntax).  For example,
a type value of a record with a single field called `t` of type `type` would look
like this:
```
{ t: <string> (type) }
```
Since type values have implied types, the `type` type decorator can be omitted:
```
{ t: <string> }
```
Now supposing we created a second field called `t2` whose type is
computed by introspecting the type of `t`.  This result is
```
{
    t: <string>,
    t2: <type>
}
```

#### 2.4.7 Error Value

An error value has the following syntax:
```
error(<value>)
```
where `<value>` is any ZSON value.

### 2.5 Type Syntax

The syntax of a type mirrors the value syntax.

A primitive type is the name of the primitive type, i.e., `string`,
`uint16`, etc.

The syntax of complex types parallels the syntax of complex values.

#### 2.5.1 Record Type

A _record type_ has the form:
```
{ <name> : <type>, <name> : <type>, ... }
```
where `<type>` is any type.  The order of the columns is significant,
e.g., type `{a:int32,b:int32}` is distinct from type `{b:int32,a:int32}`.
In contrast to schema abstractions in other formats, ZSON has no way to mark
a field optional as all fields are, in a sense optional: any field can be
encoded with a null value.  If an instance of a record value omits a value
by dropping the field altogether rather than using a null, then that record
value corresponds to a different record type that elides the field in question.

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

#### 2.5.4 Union Type

A _union type_ has the form:
```
( <type>, <type>, ... )
```
where there are at least two types in the list.

#### 2.5.5 Enum Type

An _enum type_ has the form:
```
%{ <name>, <name>, ... }
```
where `<name>` is either an identifier or a quoted string.
Each enum name must be unique.

#### 2.5.6 Map Type

A _map type_ has the form:
```
|{ <key-type>, <value-type> }|
```
where `<key-type>` is the type of the keys and `<value-type>` is the
type of the values.

#### 2.5.7 Type Type

The "type" type represents value and its syntax is simply `type`.

#### 2.5.8 Named Type

Any type name created with a type definition is referred to
simply with the same identifier used in its definition.

Type values may refer to external type tames and may elide the definition
fo the external type when the type definition for that name is known,
e.g., if `conn` is a type name, then
```
[conn]
```
is an array of elements of type `conn`.

The canonical form a type value includes the definitions of any and all referenced
types in the value using the embedded type definition syntax:
```
<name> = ( <type> )
```
where `<name>` is a type name or integer, e.g.,
```
conn=({ info:string, src:(socket=({ addr:ip, port:uint16 }), dst:socket })
```
Types in canonical form can be decoded and interpreted independently
of a type context.

A type name has the same form as an identifier except the characters
`/` and `.` are also permitted.

#### 2.5.9 Error Type

An _error type_ has the form:
```
error(<type>)
```
where `<type>` is the type of the underlying ZSON values wrapped as an error.

### 2.6 Null Value

The null value is represented by the string `null`.

Any value in ZSON can take on a null representation.  It is up to an
implementation to decide how external data structures map into and
out of values with nulls.  Typically, a null value means either the
zero value or, in the case of record fields, an optional field whose
value is not present, though these semantics are not explicitly
defined by ZSON.

## 3. Grammar

Here is a left-recursive pseudo-grammar of ZSON.  Note that not all
acceptable inputs are semantically valid as type mismatches may arise.
For example, union and enum values must both appear in a context
the defines their type.

```
<zson> = <zson> <eos> <dec-value> | <zson> <dec-value> | <dec-value>

<eos> = .

<value> = <any> | <any> <val-typedef> | <any> <decorators>

<val-typedef> = "(" "=" <type-name> ")"

<decorators> = "(" <type> ")" | <decorators> "(" <type> ")"

<any> = <primitive> | <record> | <array> | <set> |
            <union> | <enum> | <map> | <type-val>

<primitive> = primitive value as defined above

<record> = "{" <flist> "}"  |  "{"  "}"

<flist> = <flist> "," <field> | <field>

<field> = <field-name> ":" <value> | <value>

<field-name> = <identifier> | <quoted-string>

<quoted-string> = quoted string as defined above

<identifier> = as defined above

<array> = "[" <vlist> "]"  |  "["  "]"

<vlist> = <vlist> "," <value> | <value>

<set> = "|[" <vlist> "]|"  |  "|["  "]|"

<union> = <value>

<enum> = "%" ( <field-name> | <quoted-string> )

<map> = "|{" <mlist> "}|"  |  "|{"  "}|"

<mlist> = <mvalue> | <mlist> "," <mvalue>

<mvalue> = <value> ":" <value>

<type-value> = "<" <type> ">"

<error-value> = "error(" <value> ")"

<type> = <primitive-type> | <record-type> | <array-type> | <set-type> |
            <union-type> | <enum-type> | <map-type> |
            <type-def> | <type-name> | <error-type>

<primitive-type> = uint8 | uint16 | etc. as defined above including "type"

<record-type> = "{" <tflist> "}"  |  "{" "}"

<tflist> = <tflist> "," <tfield> | <tfield>

<tfield> = <field-name> ":" <type>

<array-type> = "[" <type> "]"  |  "[" "]"

<set-type> = "|[" <type> "]|"  |  "|[" "]|"

<union-type> = "(" <type> "," <tlist> ")"

<tlist> = <tlist> "," <type> | <type>

<enum-type> = "%{" <flist> "}"

<map-type> = "{" <type> "," <type> "}"

<type-def> = <identifier> = <type-type>

<type-name> = as defined above

<error-type> = "error(" <type> ")"
```
