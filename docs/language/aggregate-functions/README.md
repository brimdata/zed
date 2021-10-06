# Summarize Aggregations

The `summarize` operator performs zero or more aggregations with
zero or more [grouping expressions](../grouping/README.md).
Each aggregation is performed by an
_aggregate function_ that operates on
batches of records to carry out a running computation over the values they
contain.  The `summarize` keyword is optional.

   * [General Usage](#general-usage)
     + [Invoking](#invoking)
     + [Field Naming](#field-naming)
     + [Grouping](#grouping)
     + [`where` filtering](#where-filtering)
   * [Available Aggregate Functions](#available-aggregate-functions)
     + [`and`](#and)
     + [`any`](#any)
     + [`avg`](#avg)
     + [`collect`](#collect)
     + [`count`](#count)
     + [`countdistinct`](#countdistinct)
     + [`max`](#max)
     + [`min`](#min)
     + [`or`](#or)
     + [`sum`](#sum)
     + [`union`](#union)

> **Note:** Per Zed [search syntax](../search-syntax/README.md), many examples
> below use shorthand that leaves off the explicit leading `* |`, matching all
> records before invoking the first element in a pipeline.

## General Usage

### Invoking

Multiple aggregate functions may be invoked at the same time.

#### Example:

To simultaneously calculate the minimum, maximum, and average of the math
test scores:

```mdtest-command zed-sample-data/edu/zson
zq -f table 'min(AvgScrMath),max(AvgScrMath),avg(AvgScrMath)' satscores.zson
```

#### Output:
```mdtest-output
min max avg
289 699 484.99019042123484
```

### Field Naming

As just shown, by default the result returned by an aggregate function is
placed in a field with the same name as the aggregate function. You may
instead use `:=` to specify an explicit name for the generated field.

#### Example:

```mdtest-command zed-sample-data/edu/zson
zq -f table 'lowest:=min(AvgScrMath),highest:=max(AvgScrMath),typical:=avg(AvgScrMath)' satscores.zson
```

#### Output:
```mdtest-output
lowest highest typical
289    699     484.99019042123484
```

### Grouping

All aggregate functions may be invoked with one or more
[grouping](../grouping/README.md) options that define the batches of records on
which they operate. If explicit grouping is not used, an aggregate function
will operate over all records in the input stream.

### `where` filtering

A `where` clause may also be added to filter the values on which an aggregate
function will operate.

#### Example:

To calculate average math test scores for the cities of Los Angeles and San
Francisco:

```mdtest-command zed-sample-data/edu/zson
zq -Z 'LA_Math:=avg(AvgScrMath) where cname=="Los Angeles", SF_Math:=avg(AvgScrMath) where cname=="San Francisco"' satscores.zson
```

#### Output:
```mdtest-output
{
    LA_Math: 456.27341772151897,
    SF_Math: 485.3636363636364
}
```

---

## Available Aggregate Functions

### `and`

|                           |                                                                |
| ------------------------- | -------------------------------------------------------------- |
| **Description**           | Returns the boolean value `true` if the provided expression evaluates to `true` for all inputs. Contrast with [`or`](#or). |
| **Syntax**                | `and(<expression>)`                                            |
| **Required<br>arguments** | `<expression>`<br>A valid Zed [expression](../expressions/README.md). |
| **Optional<br>arguments** | None                                                           |

#### Example:

Many of the reocrds in our school data mention their websites, but many do
not. The following query shows the cities for which all schools have a website.

```mdtest-command zed-sample-data/edu/zson
zq -Z 'all_schools_have_website:=and(Website!=null) by City | sort City' schools.zson
```

#### Output:
```mdtest-output head
{
    City: "Acampo",
    all_schools_have_website: false
}
{
    City: "Acton",
    all_schools_have_website: false
}
{
    City: "Acton, CA",
    all_schools_have_website: true
}
...
```

---

### `any`

|                           |                                                                |
| ------------------------- | -------------------------------------------------------------- |
| **Description**           | Return one value observed for a specified field.               |
| **Syntax**                | `any(<field-name>)`                                            |
| **Required<br>arguments** | `<field-name>`<br>The name of a field.                         |
| **Optional<br>arguments** | None                                                           |

#### Example:

To see the name of one of the schools in our sample data:

```mdtest-command zed-sample-data/edu/zson
zq -z 'any(School)' schools.zson
```

For small inputs that fit in memory, this will typically be the first such
field in the stream, but in general you should not rely upon this.  In this
case, the output is:

#### Output:
```mdtest-output
{any:"'3R' Middle"}
```

---

### `avg`

|                           |                                                                |
| ------------------------- | -------------------------------------------------------------- |
| **Description**           | Return the mean (average) of the values of a specified field. Non-numeric values (including `null`) are ignored. |
| **Syntax**                | `avg(<field-name>)`                                            |
| **Required<br>arguments** | `<field-name>`<br>The name of a field.                         |
| **Optional<br>arguments** | None                                                           |

#### Example:

To calculate the average of the math test scores:

```mdtest-command zed-sample-data/edu/zson
zq -f table 'avg(AvgScrMath)' satscores.zson
```

#### Output:
```mdtest-output
avg
484.99019042123484
```

---

### `collect`

|                           |                                                                |
| ------------------------- | -------------------------------------------------------------- |
| **Description**           | Assemble all input values into an array. Contrast with [`union`](#union). |
| **Syntax**                | `collect(<field-name>)`                                        |
| **Required<br>arguments** | `<field-name>`<br>The name of a field.                         |
| **Optional<br>arguments** | None                                                           |

#### Example

For schools in Fresno county that include websites, the following query
constructs a list per city of their websites along with a parallel list of
which school each website represents.

```mdtest-command zed-sample-data/edu/zson
zq -Z 'County=="Fresno" Website!=null | Websites:=collect(Website),Schools:=collect(School) by City | sort City' schools.zson
```

#### Output:
```
{
    City: "Auberry",
    Websites: [
        "www.sierra.k12.ca.us",
        "www.sierra.k12.ca.us",
        "www.pineridge.k12.ca.us",
        "www.pineridge.k12.ca.us"
    ],
    Schools: [
        "Auberry Elementary",
        "Balch Camp Elementary",
        "Pine Ridge Elementary",
        ""
    ]
}
{
    City: "Big Creek",
    Websites: [
        "www.bigcreekschool.com",
        "www.bigcreekschool.com"
    ],
    Schools: [
        "Big Creek Elementary",
        ""
    ]
}
...
```

---

### `count`

|                           |                                                                |
| ------------------------- | -------------------------------------------------------------- |
| **Description**           | Return the number of records.                                  |
| **Syntax**                | `count([field-name])`                                          |
| **Required<br>arguments** | None                                                           |
| **Optional<br>arguments** | `[field-name]`<br>The name of a field. If specified, only records that contain this field will be counted. |

#### Example #1:

To count the number of records in the entire sample data set:

```mdtest-command zed-sample-data/edu/zson
zq -Z 'count()' *
```

#### Output:
```mdtest-output
count
1462078
```

#### Example #2:

Let's say we wanted to know how many records contain a field called `mime_type`.
The following example shows us that count and that the field is present in
our Zeek `ftp` and `files` records.

```mdtest-command zed-sample-data/zeek-default
zq -f table 'count(mime_type) by _path | filter count > 0 | sort -r count' *.log.gz
```

```mdtest-output
_path count
files 162986
ftp   93
```

---

### `countdistinct`

|                           |                                                                |
| ------------------------- | -------------------------------------------------------------- |
| **Description**           | Return a quick approximation of the number of unique values of a field.|
| **Syntax**                | `countdistinct(<field-name>)`                                  |
| **Required<br>arguments** | `<field-name>`<br>The name of a field containing values to be counted. |
| **Optional<br>arguments** | None                                                           |
| **Limitations**           | The potential inaccuracy of the calculated result is described in detail in the code and research linked from the [HyperLogLog repository](https://github.com/axiomhq/hyperloglog).<br><br>Also, partial aggregations are not yet implemented for `countdistinct` ([zed/2743](https://github.com/brimdata/zed/issues/2743)), so it may not work correctly in all circumstances. |

#### Example:

To see an approximate count of unique `uid` values in our sample data set:

```mdtest-command zed-sample-data/zeek-default
zq -f table 'countdistinct(uid)' *
```

#### Output:
```mdtest-output
countdistinct
1029651
```

To see the precise value, which may take longer to execute:

```mdtest-command zed-sample-data/zeek-default
zq -f table 'count() by uid | count()' *
```

#### Output:
```mdtest-output
count
1021953
```

Here we saw the approximation was "off" by 0.75%. On the system that was used
to perform this test, the Zed using `countdistinct()` executed almost 3x faster.

---

### `max`

|                           |                                                                |
| ------------------------- | -------------------------------------------------------------- |
| **Description**           | Return the maximum value of a specified field. Non-numeric values are ignored. |
| **Syntax**                | `max(<field-name>)`                                            |
| **Required<br>arguments** | `<field-name>`<br>The name of a field.                         |
| **Optional<br>arguments** | None                                                           |

#### Example:

To see the maximum number of bytes originated by any connection in our sample
data:

```mdtest-command zed-sample-data/zeek-default
zq -f table 'max(orig_bytes)' conn.log.gz
```

#### Output:
```mdtest-output
max
4862366
```

---

### `min`

|                           |                                                                |
| ------------------------- | -------------------------------------------------------------- |
| **Description**           | Return the minimum value of a specified field. Non-numeric values are ignored. |
| **Syntax**                | `min(<field-name>)`                                            |
| **Required<br>arguments** | `<field-name>`<br>The name of a field.                         |
| **Optional<br>arguments** | None                                                           |

#### Example:

To see the quickest round trip time of all DNS queries observed in our sample
data:

```mdtest-command zed-sample-data/zeek-default
zq -f table 'min(rtt)' dns.log.gz
```

#### Output:
```mdtest-output
min
0.000012
```

---

### `or`

|                           |                                                                |
| ------------------------- | -------------------------------------------------------------- |
| **Description**           | Returns the boolean value `true` if the provided expression evaluates to `true` for one or more inputs. Contrast with [`and`](#and). |
| **Syntax**                | `or(<expression>)`                                             |
| **Required<br>arguments** | `<expression>`<br>A valid Zed [expression](../expressions/README.md). |
| **Optional<br>arguments** | None                                                           |

#### Example:

Let's say you've noticed there's lots of HTTP traffic happening on ports higher
than the standard port `80`.

```mdtest-command zed-sample-data/zeek-default
zq -f table 'count() by id.resp_p | sort -r count' http.log.gz
```

#### Output:
```mdtest-output head
id.resp_p count
80        134496
8080      5204
5800      1691
65534     903
...
```

The following query confirms this high-port traffic is present, but that none
of those ports are higher than what TCP allows.

```mdtest-command zed-sample-data/zeek-default
zq -f table 'some_high_ports:=or(id.resp_p>80),impossible_ports:=or(id.resp_p>65535)' http.log.gz
```

#### Output:
```mdtest-output
some_high_ports impossible_ports
T               F
```

---

### `sum`

|                           |                                                                |
| ------------------------- | -------------------------------------------------------------- |
| **Description**           | Return the total sum of the values of a specified field. Non-numeric values are ignored. |
| **Syntax**                | `sum(<field-name>)`                                            |
| **Required<br>arguments** | `<field-name>`<br>The name of a field.                         |
| **Optional<br>arguments** | None                                                           |

#### Example:

To calculate the total number of bytes across all file payloads logged in our
sample data:

```mdtest-command zed-sample-data/zeek-default
zq -f table 'sum(total_bytes)' files.log.gz
```

#### Output:
```mdtest-output
sum
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

#### Example:

For schools in Fresno county that include websites, the following query
constructs a set per city of all the unique websites for the schools in that
city.

```mdtest-command zed-sample-data/edu/zson
zq -Z 'County=="Fresno" Website!=null | Websites:=union(Website) by City | sort City' schools.zson
```

#### Output:
```mdtest-output head
{
    City: "Auberry",
    Websites: |[
        "www.sierra.k12.ca.us",
        "www.pineridge.k12.ca.us"
    ]|
}
{
    City: "Big Creek",
    Websites: |[
        "www.bigcreekschool.com"
    ]|
}
...
```
