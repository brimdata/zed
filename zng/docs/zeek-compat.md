# ZSON Interoperability with Zeek Logs

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

As the ZSON data model was inspired by the [Zeek TSV log format](https://docs.zeek.org/en/stable/examples/logs/),
the ZSON/ZNG formats maintain comprehensive interoperability with Zeek.
In comparison, when Zeek is configured to output its logs in JSON format, much of the
rich type information is lost in translation.  On the other hand, Zeek TSV
can be converted to ZSON/ZNG and back to Zeek TSV without any loss of information.

This document describes how the ZSON type system
is able to represent each of the types that may appear in Zeek logs.

Tools like [`zq`](https://github.com/brimsec/zq) and [Brim](https://github.com/brimsec/brim)
maintain an internal ZNG representation of any Zeek data that is read or
imported. Therefore, knowing the equivalent types will prove useful when
performing [ZQL](../../zql/README.md) operations such as
[type casting](../../zql/docs/data-types#example) or looking at the
data when output as [ZSON](zson.md).

## Equivalent Types

The following table summarizes which ZSON data type corresponds to each
[Zeek data type](https://docs.zeek.org/en/current/script-reference/types.html)
that may appear in a Zeek log. While most types have a simple 1-to-1 mapping
from Zeek to ZSON and back to Zeek again, the sections linked from the
**Additional Detail** column describe cosmetic differences and other subtleties
applicable to handling certain types.

| Zeek Type  | ZSON Type   | Additional Detail |
|------------|------------|-------------------|
| [`bool`](https://docs.zeek.org/en/current/script-reference/types.html#type-bool)         | [`bool`](zson.md#33-primitive-values)     | |
| [`count`](https://docs.zeek.org/en/current/script-reference/types.html#type-count)       | [`uint64`](zson.md#33-primitive-values)   | |
| [`int`](https://docs.zeek.org/en/current/script-reference/types.html#type-int)           | [`int64`](zson.md#33-primitive-values)    | |
| [`double`](https://docs.zeek.org/en/current/script-reference/types.html#type-double)     | [`float64`](zson.md#33-primitive-values)  | See [`double` details](#double) |
| [`time`](https://docs.zeek.org/en/current/script-reference/types.html#type-time)         | [`time`](zson.md#33-primitive-values)     | |
| [`interval`](https://docs.zeek.org/en/current/script-reference/types.html#type-interval) | [`duration`](zson.md#33-primitive-values) | |
| [`string`](https://docs.zeek.org/en/current/script-reference/types.html#type-string)     | [`bstring` or `string`](zson.md#33-primitive-values) | See [`string` details](#string) |
| [`port`](https://docs.zeek.org/en/current/script-reference/types.html#type-port)         | [`uint16`](zson.md#33-primitive-values)   | See [`port` details](#port) |
| [`addr`](https://docs.zeek.org/en/current/script-reference/types.html#type-addr)         | [`ip`](zson.md#33-primitive-values)       | |
| [`subnet`](https://docs.zeek.org/en/current/script-reference/types.html#type-subnet)     | [`net`](zson.md#33-primitive-values)      | |
| [`enum`](https://docs.zeek.org/en/current/script-reference/types.html#type-enum)         | [`string`](zson.md#33-primitive-values)   | See [`enum` details](#enum) |
| [`set`](https://docs.zeek.org/en/current/script-reference/types.html#type-set)           | [`set`](zson.md#343-set-value)       | See [`set` details](#set) |
| [`vector`](https://docs.zeek.org/en/current/script-reference/types.html#type-vector)     | [`array`](zson.md#342-array-value)   | |
| [`record`](https://docs.zeek.org/en/current/script-reference/types.html#type-record)     | [`record`](zson.md#341-record-value) | See [`record` details](#record) |

* **Note**: The [Zeek data type](https://docs.zeek.org/en/current/script-reference/types.html)
page describes the types in the context of the
[Zeek scripting language](https://docs.zeek.org/en/current/examples/scripting/).
The Zeek types available in scripting are a superset of the data types that may
appear in Zeek log files. The encodings of the types also differ in some ways
between the two contexts. However, we link to this reference because there is
no authoritative specification of the Zeek log format.

## Example

The following example shows an input log that includes each Zeek data type,
how it's output as ZSON by `zq`, then how it's written back out again as a Zeek
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

$ zq -f zson zeek_types.log | tee zeek_types.zson
{
    my_bool: true,
    my_count: 123 (uint64),
    my_int: 456,
    my_double: 1.23456e+02,
    my_time: 2020-06-18T17:42:31.123456Z,
    my_interval: 2m3.456s (duration),
    my_printable_string: "smile😁smile" (bstring),
    my_bytes_string: "\t\x07\x04" (bstring),
    my_port: 80 (port=(uint16)),
    my_addr: 127.0.0.1,
    my_subnet: 10.0.0.0/8,
    my_enum: "tcp" (=zenum),
    my_set: |[
        "a" (bstring),
        "in" (bstring),
        "set" (bstring),
        "things" (bstring)
    ]| (=0),
    my_vector: [
        "order" (bstring),
        "is" (bstring),
        "important" (bstring)
    ] (=1),
    my_record: {
        name: "Jeanne" (bstring),
        age: 122 (uint64)
    } (=2)
} (=3)

$ zq -i zson -f zeek zeek_types.zson
#separator \x09
#set_separator	,
#empty_field	(empty)
#unset_field	-
#fields	my_bool	my_count	my_int	my_double	my_time	my_interval	my_printable_string	my_bytes_string	my_port	my_addr	my_subnet	my_enum	my_set	my_vector	my_record.name	my_record.age
#types	bool	count	int	double	time	interval	string	string	port	addr	subnet	enum	set[string]	vector[string]	string	count
T	123	456	123.456	1592502151.123456	123.456	smile\xf0\x9f\x98\x81smile	\x09\x07\x04	80	127.0.0.1	10.0.0.0/8	tcp	a,in,set,things	order,is,important	Jeanne	122
```

## Type-Specific Details

As `zq` acts as a reference implementation for ZSON/ZNG, it's helpful to understand
how it reads the following Zeek data types into ZSON equivalents and writes
them back out again in Zeek log format. Other ZSON implementations (should they
exist) may handle these differently.

Multiple Zeek types discussed below are represented via a
[type definition](zson.md#25-type-definitions) to one of ZSON's
[primitive types](zson.md#33-primitive-values). The use of the ZSON type names maintains
the history of the field's original Zeek type such that `zq` may restore it
if/when the field may be later output again in Zeek format. Knowledge of its
original Zeek type may also enable special operations in ZQL that are unique to
values known to have originated as a specific Zeek type, though no such
operations are currently implemented in `zq`.

### `double`

As they do not affect accuracy, "trailing zero" decimal digits on Zeek `double`
values will _not_ be preserved when they are formatted into a string, such as
via the ZSON/Zeek/table output options in `zq` (e.g. `123.4560` becomes
`123.456`).

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

ZSON includes a [primitive type](zson.md#33-primitive-values) called `bytes` that's
suited to storing the former "always binary" case and a `string` type for the
latter "always printable" case. However, Zeek logs do not currently communicate
detail that would allow a ZSON/ZNG implementation to know which Zeek `string` fields
to store as which of these two ZSON data types. Therefore the ZSON
`bstring` type is typically used to hold values that are read from Zeek
`string`-type fields.

One exception to this is Zeek's `_path` field. As it's a standard field that's
known to be populated by Zeek's logging system (or populated by `zq` when reading some
[Zeek JSON data](https://github.com/brimsec/zq/tree/master/zeek#type-definition-structure--importance-of-_path))
`zq` currently handles `_path` using ZSON's `string` type.

If Zeek were to provide an option to generate logs directly in ZSON/ZNG format, this
would create an opportunity to assign the appropriate ZSON `bytes` or `string`
type at the point of origin, depending on what's known about how the field's
value is intended to be populated and used.

### `port`

The numeric values that appear in Zeek logs under this type are represented
with a ZSON type name bound to the `uint16` type. See the text above regarding
[type names](#type-specific-details) for more details.

### `enum`

As they're encountered in common programming languages, enum variables
typically hold one of a set of predefined values. While this is
how Zeek's `enum` type behaves inside the Zeek scripting language,
when the `enum` type is output in a Zeek log, the log does not communicate
any such set of "allowed" values as they were originally defined. Therefore,
these values are represented with an ZSON type name bound to the `string` type. See the
text above regarding [type definitions](#type-specific-details) for more details.

### `set`

Because order within sets is not significant, no attempt is made by `zq` to
maintain the order of `set` elements as they originally appeared in a Zeek log.

### `record`

Zeek's `record` type is unique in that every Zeek log line effectively _is_ a
record, with its schema defined via the `#fields` and `#types` directives in
the headers of each log file. Unlike what we saw in the
[example ZSON output](#example), the word "record" never appears
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
$ zq -i zson -f zeek 'cut my_record' zeek_types.zson
#separator \x09
#set_separator	,
#empty_field	(empty)
#unset_field	-
#fields	my_record.name	my_record.age
#types	string	count
Jeanne	122
```
