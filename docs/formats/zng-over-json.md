# ZNG over JSON

* [ZJSON](#zjson)
  + [Type Encoding](#type-encoding)
    - [Record Type](#record-type)
    - [Array Type](#array-type)
    - [Set Type](#set-type)
    - [Union type](#union-type)
    - [Enum Type](#enum-type)
  + [Alias Encoding](#alias-encoding)
  + [Value Encoding](#value-encoding)
* [Framing ZJSON objects](#framing-zjson-objects)
* [Example](#example)

The ZNG data format has richly typed records and a deterministic column order.
Thus, encoding ZNG directly into JSON objects would not work without loss
of information.

For example, consider this ZNG (in [ZSON](zson.md) format):
```
{
    ts: 2018-03-24T17:15:21.926018012Z,
    a: "hello, world",
    b: {
        x: 4611686018427387904,
        y: 127.0.0.1
    }
}
```
A straightforward translation to JSON might look like this:
```
{
  "ts": 1521911721.926018012,
  "a": "hello, world",
  "b": {
    "x": 4611686018427388000,
    "y": "127.0.0.1"
  }
}
```
But, when this JSON is transmitted to a JavaScript client and parsed,
the result looks something like this:
```
{
  "ts": 1521911721.926018,
  "a": "hello, world",
  "b": {
    "x": 4611686018427388000,
    "y": "127.0.0.1"
  }
}
```
The good news is the `a` field came through just fine, but there are
a few problems with the remaining fields:
* the timestamp lost precision (due to 53 bits of mantissa in a JavaScript
IEEE 754 floating point number) and was converted from a time type to a number,
* the int64 lost precision for the same reason, and
* the IP address has been converted to a string.

As a comparison, Python's `json` module handles the 64-bit integer to full
precision, but loses precision on the floating point timestamp.
Also, as mentioned, it is at the whim of a JSON implementation whether
or not the order of object keys is preserved.

While JSON is well suited for data exchange of generic information, it is not
so appropriate for a structured data format like ZNG.
That said, JSON can be used as an encoding format for ZNG by mapping ZNG data
onto a JSON-based protocol.  This allows clients like web apps or
electron apps to receive and understand ZNG and, with the help of client
libraries like [zealot](https://github.com/brimdata/brim/tree/master/zealot),
to be enabled with rich, structured ZNG types that are implemented on top of
the basic JavaScript types.

In other words,
because JSON objects do not have a deterministic column order nor does JSON
in general have typing beyond the basics (i.e., strings, floating point numbers,
objects, arrays, and booleans), we decided to encode the ZNG data format with
its embedded type model all in a layer above regular JSON.

## ZJSON

The format for representing ZNG in JSON is called ZJSON.
Converting ZNG to ZJSON and back results in a complete and accurate
restoration of the original ZNG.

The ZJSON data model follows that of the underlying ZNG model by embedding
type information in the stream: type definitions declare arbitrarily complex
and nested data types, and values are sent referencing the type information
recursively with small-integer type identifiers.

Since ZNG steams are self describing and type information is embedded
in the stream itself, the embedded types are likewise encoded in the
ZJSON format.

A ZJSON stream is defined as a sequence of JSON objects where each object
represents a ZNG value.  Each object includes an identifier that denotes
its type, or _schema_.  A schema generically refers to the type of the
ZNG record that is defined by a given JSON object.

Each object contains the following fields:
* `id` a small integer encoded as a JSON number indicating the schema that
applies to this value,
* `values` a JSON array of strings and arrays encoded as defined below,
* `schema` an optional field encoding the type of this object's values
and the schema of all subsequent values that have the same identifier, as
defined below, and
* `aliases` a JSON array of objects that define bindings between string names
and ZNG types as defined below.

The ID provides a mapping to a type so that future values in the stream may
reference a schema by ID.  An implementation maintains a table to map schemas
to types as it decodes values.  The IDs are scoped to the particular ZJSON
data stream in which they are embedded and otherwise have no global persistence
or meaning.

Objects in a ZJSON stream have the following JSON structure:
```
{
        id: <id>,
        schema: <type>,
        values: [ <val> ... [ <val>, ... ] ... ]
        aliases: [ <alias1>, <alias2>, ... <aliasn> ],
}
```

### Type Encoding

The type format follows the terminology in the ZNG spec, where primitive types
represent concrete values like strings, integers, times, and so forth, while
complex types are composed of primtive types and/or other complex types, e.g.,
records, sets, arrays, and unions.

The ZJSON type encoding for a primitive type is simply its string name,
e.g., "int32" or "string".  Complex types are structured and their
mapping onto JSON depends on the type.  For example,
the ZNG type `{s:string,x:int32}` has this JSON format:
```
{
  "type": "record",
  "of": [
    {
      "name": "s",
      "type": "string"
    },
    {
      "name": "x",
      "type": "int32"
    }
  ]
}
```

#### Record Type

More formally, a ZNG record type is a JSON object of the form
```
{
        type: "record",
        of: [ <col1>, <col2>, ... <coln> ]
}
```
where each of the `n` columns has the form
```
{
        name: <name>,
        type: <type>,
        of: <of>
}
```
and `<name>` is a string defining the ZNG column name, `<type>` is a string
indicating a primitive type or complex type, and `of` is an optional field
if `<type>` is a complex type where the `<of>` value is defined in accordance
with its complex type definition.

#### Array Type

A ZNG array type is defined by a JSON object having the form
```
{
        type: "array",
        of: <type>
}
```
where `<type>` is any ZJSON type described herein, i.e., a string for a primitive
type or a JSON object defining a complex type.

#### Set Type

A ZNG set type is defined by a JSON object having the form
```
{
        type: "set",
        of: <type>
}
```
where `<type>` is any ZJSON type described herein.


#### Union type

A ZNG union type is defined by a JSON object having the form
```
{
        type: "union",
        of: [ <type1>, <type2>, ... <typen> ]
}
```
where `<type1>` through `<typen>` comprise the types of the union and
encode any ZJSON type described herein.


#### Enum Type

A ZNG enum type is a JSON object of the form
```
{
        type: "enum",
        of: [ <type>, <elem1>, ... <elemn> ]
}
```
where `<type>` represents the type of the enum values and each elements
`<elem1>` ... `<elemn>` is of the form
```
{
        name: <name>,
        value: <val>,
}
```
where `<name>` is a string defining the enumeration element name and `<value>`
is a string representing the value of the element as encoded according to the
syntax below.

### Alias Encoding

Aliases are encoded as a binding between a name and a ZNG type.
A top-level object can define zero or more aliases as follows:
```
{
        ...
        aliases: [ { <name1>:<type1>, <name2>:<type2>, ... <namen>:<typen> ]
        ...
}
```
where `<name1>` etc. are JSON strings and `<type1>` etc. are ZNG types as
defined above.

### Value Encoding

The primitive values comprising an arbitrarily complex ZNG data value are encoded
as a JSON array of strings mixed with nested JSON arrays whose structure
conforms to the nested structure of the value's schema as follows:
* each record, array, and set is encoded as a JSON array of its composite values,
* a union is encoded as a string of the form '<selector:<value>>' where `selector`
is an integer string representing the positional index in the union's list of
types that specifies the type of `<value>`, which is a JSON string or array
as described recursively herein, and
* each primitive is encoded as a string conforming to its TZNG representation,
as described in the
[corresponding section of the ZNG specification](spec.md#5-primitive-types).

For example, a record with three columns --- a string, an array of integers,
and an array of union of string, and float64 --- might have a value that looks like this:
```
[ "hello, world", ["1","2","3","4"], ["1:foo", "0:10" ] ]
```

## Framing ZJSON objects

A sequence of ZJSON objects may be framed in two primary ways.

First, they can simply be a sequence of newline delimited JSON where
each object is transmitted as a single line terminated with a newline character,
e.g., the [zq](https://github.com/brimdata/zq) CLI command writes its
ZJSON output as lines of NDJSON.

Second, the objects may be encoded in a JSON array embedded in some other
JSON-framed protocol, e.g., embedded in the the search results messages
of the [zqd REST API](https://github.com/brimdata/zq/blob/main/api/api.go).

It is up to an implementation to determine how the ZJSON
objects are framed according to its particular use case.

## Example

Here is an example that illustrates values of a repeated type,
nesting, records, array, and union:

```
{s:"hello",r:{a:1 (int32),b:2 (int32)} (=0)} (=1)
{s:"world",r:{a:3,b:4}} (1)
{s:"hello",r:{a:[1 (int32),2 (int32),3 (int32)] (=2)} (=3)} (=4)
{s:"goodnight",r:{x:{u:"foo" (5=((string,int32)))} (=6)} (=7)} (=8)
{s:"gracie",r:{x:{u:12 (int32)}}} (8)
```

This data is represented in ZJSON as follows;

```
{
  "id": 24,
  "schema": {
    "type": "record"
    "of": [
      {
        "name": "s",
        "type": "string"
      },
      {
        "type": "record"
        "name": "r",
        "of": [
          {
            "name": "a",
            "type": "int32"
          },
          {
            "name": "b",
            "type": "int32"
          }
        ],
      }
    ],
  },
  "values": [
    "hello",
    [
      "1",
      "2"
    ]
  ]
}
{
  "id": 24,
  "values": [
    "world",
    [
      "3",
      "4"
    ]
  ]
}
{
  "id": 27,
  "schema": {
    "type": "record"
    "of": [
      {
        "name": "s",
        "type": "string"
      },
      {
        "name": "r",
        "type": "record"
        "of": [
          {
            "name": "a",
            "type": "array"
            "of": "int32",
          }
        ],
      }
    ],
  },
  "values": [
    "hello",
    [
      [
        "1",
        "2",
        "3"
      ]
    ]
  ]
}
{
  "id": 31,
  "schema": {
    "type": "record"
    "of": [
      {
        "name": "s",
        "type": "string"
      },
      {
        "name": "r",
        "type": "record"
        "of": [
          {
            "name": "x",
            "type": "record"
            "of": [
              {
                "type": "union"
                "name": "u",
                "of": [
                  "string",
                  "int32"
                ],
              }
            ],
          }
        ],
      }
    ],
  },
  "values": [
    "goodnight",
    [
      [
        [
          "0",
          "foo"
        ]
      ]
    ]
  ]
}
{
  "id": 31,
  "values": [
    "gracie",
    [
      [
        [
          "1",
          "12"
        ]
      ]
    ]
  ]
}
```
