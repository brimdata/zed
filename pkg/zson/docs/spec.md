# ZSON specification

ZSON is a format for structured data values, ideally suited for streams
of heterogeneously typed records.
ZSON is richly typed and thinner than JSON.
Like [newline-delimited JSON (NDJSON)](http://ndjson.org/),
ZSON represents a sequence of data objects that can be parsed line by line.

ZSON strikes a balance between the narrowly typed but flexible NDJSON format and
a more structured approach like
[Apache Avro](https://avro.apache.org).
ZSON is type rich and
embeds all type/schema in the stream, while having a value syntax
independent of the schema, making it easy and efficient to parse on the fly
and mix and match streams from different sources with heterogeneous types.
Like Avro,
ZSON embeds schema information in the data stream but ZSON schema bindings
are lighter-weight and specified with a simple integer encoding that
accompanies each data value.

The ZSON design [is also motivated by](./rationale.md)
and maintains backward compatibility with the original [Zeek ASCII TSV log format](https://docs.zeek.org/en/stable/examples/logs/).

## ZSON format

ZSON is a UTF-8 encoded stream of "lines" where each line is terminated by
newline.  Each line is either a directive or a value.

Directives and values, in turn, come in two flavors: regular and legacy.
Thus, there are four types of lines:
* regular directives,
* regular values,
* legacy directives, and
* legacy values.

Any line that begins with character `#` is a directive.
All other lines are values.  If a value line begins with `#`, then this
character must be escaped.
(Such lines can only exist as legacy values and not regular values
as regular values always begin with an integer descriptor.)

Directives indicate how subsequent values in the ZSON stream are interpreted.
A value is a regular value if the most recent preceding directive in the stream
is a regular directive; otherwise, it is a legacy value.

Any line beginning with `#` that does not conform with the syntax of a
directive is an error.
When errors are encountered parsing ZSON, an implementation should return a
corresponding error and allow ZSON parsing to proceed if desired.

## Regular Directives

Regular directives have just three forms, each with a corresponding structure.

| Directive     | Example structure                   |
|---------------|-------------------------------------|
| Descriptor    | `#<int>:<type>`                     |
| Ordering hint | `#sort [+-]<field>,[+-]<field>,...` |
| Comment       | `#!<comment>`                       |

### Descriptor Directive

A descriptor directive defines the mapping between a decimal integer called
a "descriptor" and a type according to the following form:
```
#<descriptor>:<type>
```
There must be a single colon between the ASCII integer descriptor
and the type definition, and the integer must be composed of a string of
ASCII digits `0-9` with no leading `0` for any non-zero descriptor.
The syntax for `<type>` is described by the [type grammar](#type-grammar).
The same grammar applies to both regular types and legacy types (except for
an exception regarding `.` in field names).

The descriptor directive is the only directive that begins with an ASCII decimal
digit.

For example, a directive that is a binding between descriptor `27`
and a record comprised of fields
"foo" of type `string` and "bar" of type `int` is expressed as follows:
```
#27:record[foo:string,bar:int]
```

### Ordering Directive

The ordering directive has the following structure:
```
#sort [+-]<field>,[+-]<field>,...
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

### Comment Directive

The comment directive has the following structure:
```
#!<comment-text>
```
Comments may be used informatively and shall be
ignored by any data receivers.
`<comment-text>` can be any UTF-8 string exclusive of newline.
Comments are guaranteed to be preserved
in order within the stream and presented to higher layer components through
any ZSON parsing API.  In this way, senders and receivers of ZSON can embed
protocol directives as ZSON comments rather than defining additional
encapsulating protocols.  See the
[zson-over-http](zson-over-http.md) protocol for an example.

A comment directive may also be used to resume the interpretation of line values
as regular values instead of legacy values (as there is no legacy comment directive).

### Type Grammar

The syntax for ZSON types is a superset of the type syntax produced by Zeek logs
(Zeek logs do not produce `record` or `bytes` types).
Here is a pseudo-grammar for ZSON types:
```
<type> :=  string | bytes | int | count | double | time |
         | interval | port | addr | subnet | enum
         | vector [ <type> ]
         | set [ <type-list> ]
         | record [ <columns> ]
         | <descriptor>

<type-list> :=    <type>
                | <type-list> , <type>

<columns> :=      <column>
                | <columns> , <column>

<column> := <id> : <type>

<id> := <identifier as defined by JavaScript spec>

<descriptor> := 0 | [1-9][0-9]*
```

A reference implementation of this type system is embedded in
[zq/pkg/zeek](../../zeek).

Record types consist of an ordered set of columns where each column consists of
a name and a typed value.  Unlike JSON, the ordering of the columns is significant
and must be preserved through any APIs that consume, process, and emit ZSON records.

#### Type Examples
Simple types look like this:
```
bool
string
int
```
Container types look like this:
```
vector[int]
set[bool,string]
record[x:double,y:double]
```
Containers can be embedded in containers:
```
record[v:vector[int],s:set[bool,string],r:record[x:double,y:double],s2:set[record[a:string,b:string]]
```

Types can also refer to previously defined descriptors, e.g.,
```
#8:string
#9:record[s:8]
```
Or more usefully, descriptor references can refer to previously
declared `record` types:
```
#10:record[src:addr,srcport:port,dst:addr,dstport:port]
#11:record[list:set[10],info:string]
```

## Regular Values

A regular value is encoded on a line as a type descriptor followed by `:` followed
by a value encoding.  Here is a pseudo-grammar for value encodings:
```
<line> := <descriptor> : <elem> ;
<elem> :=
          <terminal>
        | [ <list> ]
        | [ ]
<list> :=
          <elem> ;
        | <list> <elem> ;
<terminal> := <char>*
<char> := [^\n\\;[]] | <esc-sequence>
<esc-sequence> := <JavaScript character escaping rules>
```
A value is encoded as a string of UTF-8 characters terminated
by a semicolon (which must be escaped if it appears in the value) where
composite values are contained in brackets as one or more such
semicolon terminated strings.  Any escaped characters shall be processed
and interpreted as their escaped value.

Container values are encoded as
* an open bracket,
* zero or more encoded values, and
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
using the JavaScript rules for escape syntax, i.e.,
* `\ddd` for octal escapes,
* `\xdd` for hex escapes,
* `\udddd` for unicode escapes,
* and the various single character escapes of JavaScript.

Sequences of binary data can be embedded in values using these escapes but it is
a semantic error for arbitrary binary data to be carried by any types except
`string` and `bytes` (see [Type Semantics](#type-semantics)).

These special characters must be escaped if they appear within a value:
```
; \n \\
```
In addition, `-` must be escaped if representing the single ASCII byte equal to `-` as opposed to representing an unset value.

## Examples

Here is a simple example to get the gist of this encoding.  This ZSON defines
two descriptors then uses them in three values:
```
#1:string
#2:record[a:string,b:string]
1:hello, world;
2:[hello;world;];
1:this is a semicolon: \;;
```
which represents a stream of the following three values:
```
string("hello, world")
record(a:"hello",b:"world")
string("this is a semicolon: ;")
```

The semicolon terminator is important.  Consider this ZSON depicting
sets of strings:
```
#3:set[string]
3:[hello,world;];
3:[hello;world;];
3:[];
3:[;];
```
In this example:
* the first value is a `set` of one `string`
* the second value is a set of two `string` values, `hello` and `world`,
* the third value is an empty `set`, and
* the fourth value is a `set` of one zero-length string.

In this way, empty set and set of zero-length string can be distinguished.

This scheme allows composites to be embedded in composites, e.g., a
`record` inside of a `record` like this:
```
#4:record[compass:string,degree:double]
#5:record[city:string,lat:4,long:4]
5:[NYC;[NE;40.7128];[W;74.0060;];];
```
An unset value indicates a field of a `record` that wasn't set by the encoder:
```
5:[North Pole;[N;90];-;];
```
e.g., the North Pole has a latitude but no meaningful longitude.

A `record` type can use shorthand notation as defined by
the [type grammar](#type-grammer), where reference can be made
to a previously defined `record` via its descriptor.  e.g., the `record`
defined above could be defined as follows:
```
#4:record[a:string,b:double,c:string]
#5:record[a:string,b:4,c:string]
```

## Legacy Directives

The legacy directives are backward compatible with the Zeek log format:
```
#separator<separator><char>
#set_separator<separator><char>
#empty_field<separator><string>
#unset_field<separator><string>
#path<separator><string>
#open<separator><zeek-ts>
#close<separator><zeek-ts>
#fields[<separator><string>]+
#types[<separator><type>]+
```
where `<separator>` is intialized to a space character at the beginning of the
stream, then overridden by the `#separator` directive.

In the legacy format, the `#separator` character and the `#set_separator` character
define how to parse both a legacy value line and a legacy descriptor.

Every legacy value line corresponds to a `record` type defined by the
fields and types directives possibly modified for the `#path` directive
as described below.

Record types may not be used in the `#types` directive,
which means there is no need to recursively parse the `set` and `vector`
container values (`set` and `vector` values are split according to
the `#set_separator` character).

## Legacy Values

A legacy value exists on any line that is not a directive and whose most
recent directive was a legacy directive.  The legacy value is parsed by simply
splitting the line using the `#separator` character, then splitting each container
value by the `#set_separator` character.
For example,
```
#separator \t
#set_separator  ,
#fields msg     list
#types  string  set[int]
hello\, world   1,2,3
```
represents the value
```
record(
    msg: string("hello, world")
    list: set[int](1, 2, 3)
)
```
The special characters that must be escaped if they appear within a value are:
* newline (`\n`)
* backslash (`\`)
* the `#separator` character (usually `\t`)
* the `#set_separator` character inside of set and vector values (usually `,`)
* `#unset_field` (usually `-`) if it appears as a value not be interpreted as "unset",

Similarly, a `set` with no values must be specified by the `#empty_field` string (usually `(empty)`)
to distinguish it from a `set` with a single value that's a zero-length string, and this must be escaped if it
is a single-element set with the value `(empty)`, i.e., escaped as `\x28empty)`.

When processing legacy values, a column of type `string` named `_path` is
inserted into each value, provided a `#path` directive previously appeared in the
stream.  The contents of this `_path` field is set to the string value indicated
in the `#path` directive. It becomes the leftmost column in the value and all the other columns are shifted one space to
the right.

For example,
```
#separator \t
#set_separator  ,
#path   foo
#fields msg     list
#types  string  set[int]
hello, world    1,2,3
```
represents the value
```
record(
    _path: string("foo")
    msg: string("hello, world")
    list: set[int](1, 2, 3)
)
```
This allows existing Zeek log files to be easily
ingested into ZSON-aware systems while retaining the [Zeek log type](https://docs.zeek.org/en/stable/script-reference/log-files.html) as the
`_path` field.

To maintain backward compatibility where Zeek uses `.` in columns that
came from Zeek records, e.g., `id.orig_h`, such columns shall be converted by
a legacy parser and converted to ZSON records.  Likewise, emitters of legacy
Zeek files shall flatten any records in the output by converting each sub-field
of a record to the corresponding flattened field name using dotted notation.
e.g.,
```
#separator ,
#fields id.orig_h,id.orig_p,id.resp_h,id.resp_p,message
#types addr,port,addr,port,string
```
would be interpreted as the following ZSON `record`:
```
record[id:record[orig_h:addr,orig_p:port,resp_h:addr,resp_p:port],message:string]
```

# Type Semantics

Each string parsed from a value line is interpreted according to the
type descriptor of the line.
The formats for each type is as follows:

Type | Format
---- | ------
`bool` | a single characeter `T` or `F`
`string` | a UTF-8 string that may optionally include escape sequences
`bytes` | a sequence of raw bytes encoded as base64
`int` | decimal string representation of any signed, 64-bit integer
`count` | decimal string of any unsigned, 64-bit integer
`double` | a decimal representation of a 64-bit IEEE floating point literal as defined in JavaScript
`time` | unsigned dotted decimal notation of seconds (32-bit second, 32-bit nanosecond)
`interval` | signed dotted decimal notation of seconds (32-bit second, 32-bit nanosecond)
`port` | an integer string in `[0,65535]` with an optional suffix of `/udp` or `/tcp`
`addr` | a string representing a numeric in IPv4 address form or IPv6 form
`subnet` | a string in CIDR notation representing an IP address and prefix length as defined in RFC 4632 and RFC 4291.
`enum` | a string representing an enumeration value defined outside the scope of ZSON

* Note: A `string` can embed binary data using escapes.  It's up to the receiver to determine
with out-of-band information whether the data is ultimately arbitrary binary data or
a valid UTF-8 string.

# Binary ZSON

TBD: encode values in a protobuf-like syntax.

## Related Links

* [Zeek ASCII logging](https://docs.zeek.org/en/stable/examples/logs/)
* [Binary logging in Zeek](https://www.zeek.org/development/projects/binary-logging.html)
* [Hadoop sequence file](https://cwiki.apache.org/confluence/display/HADOOP2/SequenceFile)
* [Avro](https://avro.apache.org)
* [Parquet](https://en.wikipedia.org/wiki/Apache_Parquet)
* [Protobufs](https://developers.google.com/protocol-buffers)
* [MessagePack](https://msgpack.org/index.html)
* [gNMI](https://github.com/openconfig/reference/tree/master/rpc/gnmi)
