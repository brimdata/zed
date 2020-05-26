
## Mapping Paruqet types to ZNG types

This page describes the data types that may appear in Parquet files
and how the ZNG reader maps (or can map) them into ZNG types.
Information about the Parquet format comes from the
[parquet-format github repo](https://github.com/apache/parquet-format#parquet-).
In particular, we refer to the [Apache Thrift](https://thrift.apache.org/)
definitions as the authoritative reference for what may be expressed
in a Parquet file.

By design, Parquet has a small number (7) of primitive types.
Annotations can be added to any type to describe additional semantics
(for example in ZNG, the `bytes`, `string`, `bstring`, and `enum` types
all have the same representation, Parquet uses the same primitive type
for its analogues of these types with an additional annotation to
indiciate e.g., "this field should be decoded as UTF-8")
Types can also be composed in limited ways to create more complicated
structures.

### Primitive Types

The Parquet primitive types are defined
[here](https://github.com/apache/parquet-format/blob/7390aa18ac855622f6d5cb737e9628eecd7565fd/src/main/thrift/parquet.thrift#L32-L41)

Un-annotated values of these types may be mapped to ZNG as follows:

| Parquet Type | ZNG Type | Notes |
| ------------ | -------- | ----- |
| BOOLEAN      | `bool`   | |
| INT32        | `int32`  | |
| INT64        | `int64`  | |
| INT96        | (none)   | (see note below) |
| FLOAT        | `float64` | This Parquet type is a 32 bit float, but the only float in ZNG is 64 bits |
| DOUBLE       | `float64` | |
| BYTE_ARRAY   | `bstring` | The ZNG `bytes` type would be more appropriate if/when it is implemented |
| FIXED_LEN_BYTE_ARRAY | `bstring` | (same as above) |

Note: We have come across INT96 valued columns "in the wild" in files
created with pyspark that use timestamp types.  They apparently use
[this format](https://github.com/xhochy/parquet-format/blob/cb4727767823ae201fd567f67825cc22834c20e9/LogicalTypes.md#int96-timestamps-also-called-impala_timestamp) but empirically, the pyspark writer seems to lose precision when
using this format so its probably not a very high priority to support it.

### Logical Types

Parquet types that include annotations to provide additional information
on how they should be interpreted are called logical types.
There is a good description of logical types at
<https://github.com/apache/parquet-format/blob/master/LogicalTypes.md>.
Note that there are two ways to express these annotations in the file:
"Converted Types" and "Logical Types".  Although the Parquet
specification focuses on Logical Types and describes Converted Types as
an "older representation", we haven't encountered files that use
Logical Types.  In particular, files created with recent
Nifi (version 1.11.4) use Converted Types.
In any case, expanding support to cover Logical Types should be
straightforward -- the difference is between the two is mostly about how
the annotations are formatted in the file and Logical Types simply
offer more flexibility and expressiveness.

The Thrift definitions for Converted Types are
[here](https://github.com/apache/parquet-format/blob/7390aa18ac855622f6d5cb737e9628eecd7565fd/src/main/thrift/parquet.thrift#L48-L177).
Thrift definitions for Logical Types are
[here](https://github.com/apache/parquet-format/blob/7390aa18ac855622f6d5cb737e9628eecd7565fd/src/main/thrift/parquet.thrift#L227-L344).

These types can be mapped to ZNG as follows:

| Parquet Type | ZNG Type | Notes |
| ------------ | -------- | ----- |
| Converted Type UTF8<br>Logical Type STRING | `string` ||
| Converted Types MAP, MAP_KEY_VALUE<br>Logical Type MAP | | see below |
| Converted Type LIST<br>Logical Type LIST | | see below |
| Converted Type ENUM<br>Logical Type ENUM | `string` | This could be the ZNG `enum` type if we ressurected it |
| Converted Type DECIMAL<br>Logical Type DECIMAL | (none) | ZNG doesn't have an equivalent type.  We could convert these to floating point, but that would come at the cost of lost precision -- presumably people are using this type to avoid that problem. |
| Converted Type DATE<br>Logical Type DATE | `time` | The Parquet type is just a date, not a particular time on a given date.  So, ZNG `time` is not exactly equivalent but we could define a convention such as "midnight UTC on the given date" |
| Converted Types TIME_MILLIS, TIME_MICROS<br>Logical Type TIME | (none) | This is a particular time without an associated date (e.g., 3:00 PM).  ZNG has no equivalent type |
| Converted Types TIMESTAMP_MILLIS, TIMESTAMP_MICROS<br>Logical Type TIMESTAMP | `time` | |
| Converted Types UINT_8, UINT_16, UINT_32, UINT_64, INT_8, INT_16, INT_32, INT_64<br>Logical Type INTEGER | `byte`, `uint16`, `uint32`, `uint64`, `int16`, `int32`, `int64` | ZNG has no signed 8-bit value, we could just convert that to an `int16`? |
| Converted Types JSON, BSON<br>Logical Types JSON, BSON | (none) | No ZNG equivalent.  We could represent as `bstring` to allow the data to be stored in ZNG, but it can't be operated on with ZQL. |
| Converted Type INTERVAL | `duration` | Parquet intervals can include months which makes them variable.  Such an interval can't be represented by the ZNG `duration` type but shorter intervals can be. |
| Logical Type UNKNOWN | `null` | |
| Logical Type UUID | `string` or `bstring` | |

Note: Parquet has no annotation to describe IP addresses or "subnets".
How do existing Zeek->Parquet converters handle this and how can we
ensure that we get this data back into an appropriate ZNG type?
In the short term, we can do something ad hoc, but the long-term solution
is probably something like what we do for JSON to supply additional
out-of-band information about how particular fields should be translated.

### Repeated Types

Orthogonally to Converted Types and Logical Types, any field in Parquet
can be "repeated".  This building block is used in conjunction with the
MAP and LIST types to build more complex structures.
The zq Parquet reader currently translates LIST structures as defined
[here](https://github.com/apache/parquet-format/blob/master/LogicalTypes.md#lists) into ZNG `vector` types.  Any other repeated values are silently
ignored by the Parquet reader.
For the MAP type speficially, ZNG doesn't have any native equivalent.
Adding maps as a ZNG feature has been discussed in the past.  In the
absence of native maps, we could do something like
`set[record[key:sometype, value:sometype]]` though it wouldn't be practical
to operate on these from ZQL.
