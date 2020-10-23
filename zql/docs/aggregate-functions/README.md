# Aggregate Functions

A pipeline may contain one or more _aggregate functions_, which operate on
batches of events to carry out a running computation over values contained in
the events.

The [General Usage](#general-usage) section below describes details
relevant to all aggregate functions, then the following
[Available Aggregate Functions](#available-aggregate-functions) are
documented in detail:

* [`and`](#and)
* [`avg`](#avg)
* [`collect`](#collect)
* [`count`](#count)
* [`countdistinct`](#countdistinct)
* [`first`](#first)
* [`last`](#last)
* [`max`](#max)
* [`min`](#min)
* [`or`](#or)
* [`sum`](#sum)
* [`union`](#union)

**Note**: Per ZQL [search syntax](../search-syntax/README.md), many examples
below use shorthand that leaves off the explicit leading `* |`, matching all
events before invoking the first element in a pipeline.

# General Usage

All aggregate functions may be invoked with one or more
[grouping](../grouping/README.md) options that define the batches of events on
which they operate. If explicit grouping is not used, an aggregate function
will operate over all events in the input stream.

Multiple aggregate functions may be invoked at the same time.

#### Example:

To simultaneously calculate the minimum, maximum, and average of connection
duration:

```zq-command
zq -f table 'min(duration),max(duration),avg(duration)' conn.log.gz
```

#### Output:
```zq-output
MIN      MAX         AVG
0.000001 1269.512465 1.6373747834138621
```

As just shown, by default the result returned by an aggregate function is
placed in a field with the same name as the aggregate function. You may
instead use `=` to specify an explicit name for the generated field.

#### Example:

```zq-command
zq -f table 'quickest=min(duration),longest=max(duration),typical=avg(duration)' conn.log.gz
```

#### Output:
```zq-output
QUICKEST LONGEST     TYPICAL
0.000001 1269.512465 1.6373747834138621
```

---

# Available Aggregate Functions

## `and`

|                           |                                                                |
| ------------------------- | -------------------------------------------------------------- |
| **Description**           | Returns the boolean value `true` if the provided expression evaluates to `true` for all input values. Contrast with [`or`](#or). |
| **Syntax**                | `and(<expression>)`                                            |
| **Required<br>arguments** | `<expression>`<br>A valid ZQL [expression](../expressions/README.md). |
| **Optional<br>arguments** | None                                                           |
| **Developer Docs**        | https://pkg.go.dev/github.com/brimsec/zq/reducer/logical       |

#### Example:

Let's say you've been studying `weird` events and noticed that lots of
connections have made one or more bad HTTP requests.

```zq-command
zq -f table 'count() by name | sort -r count' weird.log.gz
```

#### Output:
```zq-output head:5
NAME                                        COUNT
bad_HTTP_request                            11777
line_terminated_with_single_CR              11734
unknown_HTTP_method                         140
above_hole_data_without_any_acks            107
...
```

To count the number of connections for which this was the _only_ category of
`weird` event observed:

```zq-command
zq -f table 'only_bads=and(name="bad_HTTP_request") by uid | count() where only_bads=true' weird.log.gz
```

#### Output:
```zq-output
COUNT
37
```

---
## `avg`

|                           |                                                                |
| ------------------------- | -------------------------------------------------------------- |
| **Description**           | Return the mean (average) of the values of a specified field. Non-numeric values are ignored. |
| **Syntax**                | `avg(<field-name>)`                                            |
| **Required<br>arguments** | `<field-name>`<br>The name of a field.                         |
| **Optional<br>arguments** | None                                                           |
| **Developer Docs**        | https://pkg.go.dev/github.com/brimsec/zq/reducer#Avg           |

#### Example:

To calculate the average number of bytes originated by all connections as
captured in Zeek `conn` events:

```zq-command
zq -f table 'avg(orig_bytes)' conn.log.gz
```

#### Output:
```zq-output
AVG
176.9861548654682
```

---

## `collect`

|                           |                                                                |
| ------------------------- | -------------------------------------------------------------- |
| **Description**           | Assemble all input values into an array. Contrast with [`union`](#union). |
| **Syntax**                | `collect(<field-name>)`                                        |
| **Required<br>arguments** | `<field-name>`<br>The name of a field.                         |
| **Optional<br>arguments** | None                                                           |
| **Developer Docs**        | https://pkg.go.dev/github.com/brimsec/zq/reducer#Collect       |

#### Example #1:

To assemble the sequence of HTTP methods invoked in each interaction with the
Bing search engine:

```zq-command
zq -f table 'host=www.bing.com | methods=collect(method) by uid | sort uid' http.log.gz
```

#### Output:
```zq-output head:5
UID                METHODS
C1iilt2FG8PnyEl0bb GET,GET,POST,GET,GET,POST
C31wi6XQB8h9igoa5  GET,GET,POST,POST,POST
CFwagt4ivDe3p6R7U8 GET,GET,POST,POST,GET,GET,GET,POST,POST,GET,GET,GET,GET,POST
CI0SCN14gWpY087KA3 GET,POST,GET,GET,GET,GET,GET,GET,GET,GET,GET,GET,GET
...
```

---

## `count`

|                           |                                                                |
| ------------------------- | -------------------------------------------------------------- |
| **Description**           | Return the number of events.                                   |
| **Syntax**                | `count([field-name])`                                          |
| **Required<br>arguments** | None                                                           |
| **Optional<br>arguments** | `[field-name]`<br>The name a field. If specified, only events that contain this field will be counted. |
| **Developer Docs**        | https://pkg.go.dev/github.com/brimsec/zq/reducer#Count         |

#### Example #1:

To count the number of events in the entire sample data set:

```zq-command
zq -f table 'count()' *.log.gz
```

#### Output:
```zq-output
COUNT
1462078
```

#### Example #2:

Let's say we wanted to know how many events contain a field called `mime_type`.
The following example shows us that count and that the field is present in
in our Zeek `ftp` and `files` events.

```zq-command
zq -f table 'count(mime_type) by _path | filter count > 0 | sort -r count' *.log.gz
```

```zq-output
_PATH COUNT
files 162986
ftp   93
```

---


## `countdistinct`

|                           |                                                                |
| ------------------------- | -------------------------------------------------------------- |
| **Description**           | Return a quick approximation of the number of unique values of a field. |
| **Syntax**                | `countdistinct(<field-name>)`                                  |
| **Required<br>arguments** | `<field-name>`<br>The name of a field containing values to be counted. |
| **Optional<br>arguments** | None                                                           |
| **Limitations**           | The potential inaccuracy of the calculated result is described in detail in the code and research linked from the [HyperLogLog repository](https://github.com/axiomhq/hyperloglog). |
| **Developer Docs**        | https://pkg.go.dev/github.com/brimsec/zq/reducer#CountDistinct |

#### Example:

To see an approximate count of unique `uid` values in our sample data set:

```zq-command
zq -f table 'countdistinct(uid)' *
```

#### Output:
```zq-output
COUNTDISTINCT
1029651
```

To see the precise value, which may take longer to execute:

```zq-command
zq -f table 'count() by uid | count()' *
```

#### Output:
```zq-output
COUNT
1021953
```

Here we saw the approximation was "off" by 0.75%. On the system that was used
to perform this test, the ZQL using `countdistinct()` executed almost 3x faster.

---

## `first`

|                           |                                                                |
| ------------------------- | -------------------------------------------------------------- |
| **Description**           | Return the first value observed for a specified field, based on input order. |
| **Syntax**                | `first(<field-name>)`                                          |
| **Required<br>arguments** | `<field-name>`<br>The name of a field.                         |
| **Optional<br>arguments** | None                                                           |
| **Developer Docs**        | https://pkg.go.dev/github.com/brimsec/zq/reducer#First         |

#### Example:

To see the `name` of the first Zeek `weird` event in our sample data:

```zq-command
zq -f table 'first(name)' weird.log.gz
```

#### Output:
```zq-output
FIRST
TCP_ack_underflow_or_misorder
```

---

## `last`

|                           |                                                                |
| ------------------------- | -------------------------------------------------------------- |
| **Description**           | Return the last value observed for a specified field, based on input order. |
| **Syntax**                | `last(<field-name>)`                                           |
| **Required<br>arguments** | `<field-name>`<br>The name of a field.                         |
| **Optional<br>arguments** | None                                                           |
| **Developer Docs**        | https://pkg.go.dev/github.com/brimsec/zq/reducer#Last          |

#### Example:

To see the final domain name queried via DNS in our sample data:

```zq-command
zq -f table 'last(query)' dns.log.gz
```

#### Output:
```zq-output
LAST
talk.google.com
```

---

## `max`

|                           |                                                                |
| ------------------------- | -------------------------------------------------------------- |
| **Description**           | Return the maximum value of a specified field. Non-numeric values are ignored. |
| **Syntax**                | `max(<field-name>)`                                            |
| **Required<br>arguments** | `<field-name>`<br>The name of a field.                         |
| **Optional<br>arguments** | None                                                           |
| **Developer Docs**        | https://pkg.go.dev/github.com/brimsec/zq/reducer/field#FieldReducer |

#### Example:

To see the maximum number of bytes originated by any connection in our sample
data:

```zq-command
zq -f table 'max(orig_bytes)' conn.log.gz
```

#### Output:
```zq-output
MAX
4862366
```

---

## `min`

|                           |                                                                |
| ------------------------- | -------------------------------------------------------------- |
| **Description**           | Return the minimum value of a specified field. Non-numeric values are ignored. |
| **Syntax**                | `min(<field-name>)`                                            |
| **Required<br>arguments** | `<field-name>`<br>The name of a field.                         |
| **Optional<br>arguments** | None                                                           |
| **Developer Docs**        | https://pkg.go.dev/github.com/brimsec/zq/reducer/field#FieldReducer |

#### Example:

To see the quickest round trip time of all DNS queries observed in our sample
data:

```zq-command
zq -f table 'min(rtt)' dns.log.gz
```

#### Output:
```zq-output
MIN
0.000012
```

---

## `or`

|                           |                                                                |
| ------------------------- | -------------------------------------------------------------- |
| **Description**           | Returns the boolean value `true` if the provided expression evaluates to `true` for any input value. Contrast with [`and`](#and). |
| **Syntax**                | `or(<expression>)`                                             |
| **Required<br>arguments** | `<expression>`<br>A valid ZQL [expression](../expressions/README.md). |
| **Optional<br>arguments** | None                                                           |
| **Developer Docs**        | https://pkg.go.dev/github.com/brimsec/zq/reducer/logical       |

#### Example:

Let's say you've noticed there's lots of HTTP traffic happening on ports higher
than the standard port `80`.

```zq-command
zq -f table 'count() by id.resp_p | sort -r' http.log.gz
```

#### Output:
```zq-output head:5
ID.RESP_P COUNT
80        134496
8080      5204
5800      1691
65534     903
...
```

The following query confirms this high-port traffic is present, but that none of
those ports are higher than what TCP allows.

```zq-command
zq -f table 'some_highports=or(id.resp_p>80),impossible_ports=or(id.resp_p>65535)' http.log.gz
```

#### Output:
```zq-output
SOME_HIGHPORTS IMPOSSIBLE_PORTS
T              F
```

---

## `sum`

|                           |                                                                |
| ------------------------- | -------------------------------------------------------------- |
| **Description**           | Return the total sum of the values of a specified field. Non-numeric values are ignored. |
| **Syntax**                | `sum(<field-name>)`                                            |
| **Required<br>arguments** | `<field-name>`<br>The name of a field.                         |
| **Optional<br>arguments** | None                                                           |
| **Developer Docs**        | https://pkg.go.dev/github.com/brimsec/zq/reducer/field#FieldReducer |

#### Example:

To calculate the total number of bytes across all file payloads logged in our
sample data:

```zq-command
zq -f table 'sum(total_bytes)' files.log.gz
```

#### Output:
```zq-output
SUM
3092961270
```

---

## `union`

|                           |                                                                |
| ------------------------- | -------------------------------------------------------------- |
| **Description**           | Gather all unique input values into a set. Contrast with [`collect`](#collect). |
| **Syntax**                | `union(<field-name>)`                                          |
| **Required<br>arguments** | `<field-name>`<br>The name of a field.                         |
| **Optional<br>arguments** | None                                                           |
| **Limitations**           | The data type of the input values must be uniform.             |
| **Developer Docs**        | https://pkg.go.dev/github.com/brimsec/zq/reducer#Union         |

#### Example #1:

To observe which HTTP methods were invoked in each interaction with the Bing
search engine:

```zq-command
zq -f table 'host=www.bing.com | methods=union(method) by uid | sort uid' http.log.gz
```

#### Output:
```zq-output head:9
UID                METHODS
C1iilt2FG8PnyEl0bb GET,POST
C31wi6XQB8h9igoa5  GET,POST
CFwagt4ivDe3p6R7U8 GET,POST
CI0SCN14gWpY087KA3 GET,POST
CJcF5E1DVn8FLq5JVc POST
CLsXgZ1W5l9gMzx7e8 GET,POST
CM2qfb4dhM2KJ6uAZk GET
CSOmBD4vJEGRU6pJmg POST
...
```
