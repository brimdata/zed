# ZNG Compatibility with Zeek Logs

- [Introduction](#introduction)
- [Equivalent Types](#equivalent-types)
- [Example](#example)
- [Type-Specific Details](#type-specific-details)
  * [`double`](#double)
  * [`set`](#set)
  * [`enum`](#enum)
  * [`string`](#string)
  * [`record`](#record)

## Introduction

As the ZNG design was motivated by the [Zeek log format](https://docs.zeek.org/en/stable/examples/logs/),
care has been taken to maintain compatibility with it. This document describes
how the type system described in the [ZNG specification](spec.md)
is able to represent each of the types that may appear in Zeek logs.

Tools like [`zq`](https://github.com/brimsec/zq) and [Brim](https://github.com/brimsec/brim)
maintain an internal ZNG representation of any Zeek data that is read or
imported. Therefore, knowing the equivalent types will prove useful when
performing [ZQL](../../zql/README.md) operations such as
[type casting](../../zql/docs/data-types#example) or looking at the
data when output as [TZNG](spec.md#4-zng-text-format-tzng).

## Equivalent Types

The following table summarizes which ZNG data type corresponds to each
[Zeek data type](https://docs.zeek.org/en/current/script-reference/types.html)
that may appear in a Zeek log. While most types have a simple 1-to-1 mapping
from Zeek to ZNG and back to Zeek again, the sections linked from the
**Additional Detail** column describe cosmetic differences and other subtleties
applicable to handling certain types.

| Zeek Type  | ZNG Type   | Additional Detail |
|------------|------------|-------------------|
| [`bool`](https://docs.zeek.org/en/current/script-reference/types.html#type-bool)         | [`bool`](spec.md#5-primitive-types)     | |
| [`count`](https://docs.zeek.org/en/current/script-reference/types.html#type-count)       | [`uint64`](spec.md#5-primitive-types)   | |
| [`int`](https://docs.zeek.org/en/current/script-reference/types.html#type-int)           | [`int64`](spec.md#5-primitive-types)    | |
| [`double`](https://docs.zeek.org/en/current/script-reference/types.html#type-double)     | [`float64`](spec.md#5-primitive-types)  | See [`double` details](#double) |
| [`time`](https://docs.zeek.org/en/current/script-reference/types.html#type-time)         | [`time`](spec.md#5-primitive-types)     | |
| [`interval`](https://docs.zeek.org/en/current/script-reference/types.html#type-interval) | [`duration`](spec.md#5-primitive-types) | |
| [`string`](https://docs.zeek.org/en/current/script-reference/types.html#type-string)     | [`bstring` or `string`](spec.md#5-primitive-types) | See [`string` details](#string) |
| [`port`](https://docs.zeek.org/en/current/script-reference/types.html#type-port)         | [`port`](spec.md#5-primitive-types)     | |
| [`addr`](https://docs.zeek.org/en/current/script-reference/types.html#type-addr)         | [`ip`](spec.md#5-primitive-types)       | |
| [`subnet`](https://docs.zeek.org/en/current/script-reference/types.html#type-subnet)     | [`net`](spec.md#5-primitive-types)      | |
| [`enum`](https://docs.zeek.org/en/current/script-reference/types.html#type-enum)         | [`string`](spec.md#5-primitive-types)   | See [`enum` details](#enum) |
| [`set`](https://docs.zeek.org/en/current/script-reference/types.html#type-set)           | [`set`](spec.md#3113-set-typedef)       | See [`set` details](#set) | 
| [`vector`](https://docs.zeek.org/en/current/script-reference/types.html#type-vector)     | [`array`](spec.md#3112-array-typedef)   | |
| [`record`](https://docs.zeek.org/en/current/script-reference/types.html#type-record)     | [`record`](spec.md#3111-record-typedef) | See [`record` details](#record) |

* **Note**: The [Zeek data type](https://docs.zeek.org/en/current/script-reference/types.html)
page describes the types in the context of the
[Zeek scripting language](https://docs.zeek.org/en/current/examples/scripting/).
The Zeek types available in scripting are a superset of the data types that may
appear in Zeek log files. The encodings of the types also differ in some ways
between the two contexts. However, we link to this reference because there is
no authoritative specification of the Zeek log format.

## Example

The following example shows an input log that includes each Zeek data type,
how it's output as TZNG by `zq`, then how it's written back out again as a Zeek
log. You may find it helpful to refer to this example when reading the
[Type-Specific Details](#type-specific-details) sections.

```
$ cat zeek_types.log 
#separator \x09
#set_separator	,
#empty_field	(empty)
#unset_field	-
#fields	my_bool	my_count	my_int	my_double	my_time	my_interval	my_printable_string	my_bytes_string	my_port	my_addr	my_subnet	my_enum	my_set	my_vector	my_record.name	my_record.age
#types	bool	count	int	double	time	interval	string	string	port	addr	subnet	enum	set[string]	vector[string]	string	count
T	123	456	123.4560	1592502151.123456	123.456	smile\xf0\x9f\x98\x81smile	\x09\x07\x04	80	127.0.0.1	10.0.0.0/8	tcp	things,in,a,set	order,is,important	Jeanne	122

$ zq -t zeek_types.log 
#zenum=string
#0:record[my_bool:bool,my_count:uint64,my_int:int64,my_double:float64,my_time:time,my_interval:duration,my_printable_string:bstring,my_bytes_string:bstring,my_port:port,my_addr:ip,my_subnet:net,my_enum:zenum,my_set:set[bstring],my_vector:array[bstring],my_record:record[name:bstring,age:uint64]]
0:[T;123;456;123.456;1592502151.123456;123.456;smile😁smile;\x09\x07\x04;80;127.0.0.1;10.0.0.0/8;tcp;[a;in;set;things;][order;is;important;][Jeanne;122;]]

$ zq -t zeek_types.log | zq -f zeek -
#separator \x09
#set_separator	,
#empty_field	(empty)
#unset_field	-
#fields	my_bool	my_count	my_int	my_double	my_time	my_interval	my_printable_string	my_bytes_string	my_port	my_addr	my_subnet	my_enum	my_set	my_vector	my_record.name	my_record.age
#types	bool	count	int	double	time	interval	string	string	port	addr	subnet	enum	set[string]	vector[string]	string	count
T	123	456	123.456	1592502151.123456	123.456	smile\xf0\x9f\x98\x81smile	\x09\x07\x04	80	127.0.0.1	10.0.0.0/8	tcp	a,in,set,things	order,is,important	Jeanne	122
```

## Type-Specific Details

As `zq` acts as a reference implementation for ZNG, it's helpful to understand
how it reads the following Zeek data types into ZNG equivalents and writes
them back out again in Zeek log format. Other ZNG implementations (should they
exist) may handle these differently.

### `double`

As they do not affect accuracy, "trailing zero" decimal digits on Zeek `double`
values will _not_ be preserved when they are formatted into a string, such as
via the TZNG/Zeek/table output options in `zq` (e.g. `123.4560` becomes
`123.456`).

### `set`

Because order within sets is not significant, no attempt is made by `zq` to
maintain the order of `set` elements as they originally appeared in a Zeek log.

### `enum`

As they're encountered in common programming languages, enum variables
typically hold one of a set of predefined values. While this is
how Zeek's `enum` type behaves inside the Zeek scripting language,
when the `enum` type is output in a Zeek log, the log does not communicate
any such set of "allowed" values as they were originally defined. Therefore,
when `zq` reads a Zeek `enum` into ZNG, it defines a
[type alias](spec.md#412-type-alias) called `zenum` to use for such a field,
ultimately treating the value as if it were of the ZNG `string` type. The use
of the alias maintains the history of the field having originally been read in
from a Zeek `enum` field. This allows `zq` to restore the Zeek `enum` type
if/when the field may be later output again in Zeek log format. However, when
working with the value in ZQL, only `string`-type operations will be possible.

As explained in the [alpha notice in the ZNG specification](spec.md), a true
ZNG `enum` type with predefined values has not yet been defined in the spec
nor implemented in `zq`. Once available in ZNG, Zeek could potentially
offer direct log output in ZNG format that communicates the full definition of
an `enum`, including the set of allowed values.

### `string`

Zeek's `string` data type is complicated by its ability to hold printable ASCII
and UTF-8 as well as arbitrary unprintable bytes represented as `\x` escapes.
Because such binary data may need to legitimately be captured (e.g. to record
the symptoms of DNS exfiltration), it's helpful that Zeek has a mechanism to
log it. Unfortunately, Zeek's use of the single `string` type for these
multiple uses leaves out important detail about the intended interpretation and
presentation of the bytes that make up the value. For instance, one Zeek
`string` field may hold arbitrary network data that _coincidentally_ sometimes
form byte sequences that could be interpreted as prinable UTF-8, but they are
_not_ intended to be read or presented as such. Meanwhile, another Zeek
`string` field may be populated such that it will _only_ ever contain printable
UTF-8. These details are currently only captured within the Zeek source code
itself that defines how these values are generated.

ZNG includes a [primitive type](spec.md#5-primitive-types) called `bytes` that's
suited to storing the former "always binary" case and a `string` type for the
latter "always printable" case. However, Zeek logs do not currently communicate
detail that would allow a ZNG implementation to know which Zeek `string` fields
to store as which of these two ZNG data types. Therefore the single ZNG
`bstring` type is typically used to hold values that are read from Zeek
`string`-type fields.

One exception to this is Zeek's `_path` field. As it's a standard field that's
known to be populated by Zeek's logging system (or populated by `zq` when reading some
[Zeek JSON data](https://github.com/brimsec/zq/tree/master/zeek#type-definition-structure--importance-of-_path))
`zq` currently handles `_path` using ZNG's `string` type.

If Zeek were to provide an option to generate logs directly in ZNG format, this
would create an opportunity to assign the appropriate ZNG `bytes` or `string`
type at the point of origin, depending on what's known about how the field's
value is intended to be populated and used.

### `record`

Zeek's `record` type is unique in that every Zeek log line effectively _is_ a
record, with its schema defined via the `#fields` and `#types` directives in
the headers of each log file. Unlike what we saw in the
[example TZNG output](#example), the word "record" never appears
explicitly in the schema definition in Zeek logs.

Embedded records also subtly appear within Zeek log lines in the form of
dot-separated field names. A common example in Zeek is the
[`id`](https://docs.zeek.org/en/current/scripts/base/init-bare.zeek.html#type-conn_id),
record which captures the source/destination IP & port combination for a
network connection as fields `id.orig_h`, `id.orig_p`, `id.resp_h`, and
`id.resp_p`. When reading such fields into their ZNG equivalent, `zq` restores
the hierarchical nature of the record as it originally existed inside of Zeek
itself before it was output by its logging system. This enables operations in
ZQL that refer to the record at a higher level but affect all values lower
down in the record hierarchy. Revisiting the data from our
example:

```
$ zq -t zeek_types.log | zq -f zeek "cut my_record" -
#separator \x09
#set_separator	,
#empty_field	(empty)
#unset_field	-
#fields	my_record.name	my_record.age
#types	string	count
Jeanne	122
```
