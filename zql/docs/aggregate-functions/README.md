# Aggregate Functions

A pipeline may contain one or more _aggregate functions_, which operate on
batches of events to carry out a running computation over values contained in
the events.

The [General Usage](#general-usage) section below describes details
relevant to all aggregate functions, then the following
[Available Aggregate Functions](#available-aggregate-functions) are
documented in detail:

* [`avg`](#avg)
* [`count`](#count)
* [`countdistinct`](#countdistinct)
* [`first`](#first)
* [`last`](#last)
* [`max`](#max)
* [`min`](#min)
* [`sum`](#sum)

**Note**: In the examples below, we'll use the `zq -f table` output format for human readability. Due to the width of the Zeek events used as sample data, you may need to "scroll right" in the output to see some field values.

**Note**: Per ZQL [search syntax](../search-syntax/README.md), many examples below use shorthand that leaves off the explicit leading `* |`, matching all events before invoking the first element in a pipeline.

# General Usage

All aggregate functions may be invoked with [Grouping](../Grouping/README.md)
options that define the batches on which an aggregate function will operate.

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
instead use `=` to specify an explicit name for the field.

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

## `avg`

|                           |                                                                |
| ------------------------- | -------------------------------------------------------------- |
| **Description**           | Return the mean (average) of the values of a specified field.  | 
| **Syntax**                | `avg(<field-name>)`                                            |
| **Required<br>arguments** | `<field-name>`<br>The name of a field containing numeric values to average. |
| **Optional<br>arguments** | None                                                           |
| **Developer Docs**        | https://godoc.org/github.com/brimsec/zq/reducer#Avg            |

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

## `count`

|                           |                                                                |
| ------------------------- | -------------------------------------------------------------- |
| **Description**           | Return the number of events. |
| **Syntax**                | `count([field-name])`                                          |
| **Required<br>arguments** | None                                                           |
| **Optional<br>arguments** | `<field-name>`<br>The name of a field. If specified, only events that contain this field will be counted. |
| **Developer Docs**        | https://godoc.org/github.com/brimsec/zq/reducer#Count          |

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
