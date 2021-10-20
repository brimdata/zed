# Summarize Aggregations

> **Note:** Many examples below are generated using the
> [educational sample data](https://github.com/brimdata/zed-sample-data/tree/edu-data/edu),
> which you may wish to clone locally to reproduce the examples and create
> your own query variations.

The `summarize` operator performs zero or more aggregations with
zero or more [grouping expressions](grouping.md).
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

## General Usage

### Invoking

Multiple aggregate functions may be invoked at the same time.

#### Example:

To simultaneously calculate the minimum, maximum, and average of the math
test scores:

```mdtest-command dir=zed-sample-data/edu/zson
zq -f table 'min(AvgScrMath),max(AvgScrMath),avg(AvgScrMath)' testscores.zson
```

#### Output:
```mdtest-output
min max avg
289 699 484.99019042123484
```

### Field Naming

As just shown, by default the result returned is placed in a field with the
same name as the aggregate function. You may instead use `:=` to specify an
explicit name for the generated field.

#### Example:

```mdtest-command dir=zed-sample-data/edu/zson
zq -f table 'lowest:=min(AvgScrMath),highest:=max(AvgScrMath),typical:=avg(AvgScrMath)' testscores.zson
```

#### Output:
```mdtest-output
lowest highest typical
289    699     484.99019042123484
```

### Grouping

All aggregate functions may be invoked with one or more
[grouping](grouping.md) options that define the batches of records on
which they operate. If explicit grouping is not used, an aggregate function
will operate over all records in the input stream.

### `where` filtering

A `where` clause may also be added to filter the values on which an aggregate
function will operate.

#### Example:

To calculate average math test scores for the cities of Los Angeles and San
Francisco:

```mdtest-command dir=zed-sample-data/edu/zson
zq -Z 'LA_Math:=avg(AvgScrMath) where cname=="Los Angeles", SF_Math:=avg(AvgScrMath) where cname=="San Francisco"' testscores.zson
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
| **Required<br>arguments** | `<expression>`<br>A valid Zed [expression](expressions.md). |
| **Optional<br>arguments** | None                                                           |

#### Example:

Many of the school records in our sample data include websites, but many do
not. The following query shows the cities in which all schools have a website.

```mdtest-command dir=zed-sample-data/edu/zson
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

```mdtest-command dir=zed-sample-data/edu/zson
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

```mdtest-command dir=zed-sample-data/edu/zson
zq -f table 'avg(AvgScrMath)' testscores.zson
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
constructs an ordered list per city of their websites along with a parallel
list of which school each website represents.

```mdtest-command dir=zed-sample-data/edu/zson
zq -Z 'County=="Fresno" Website!=null | Websites:=collect(Website),Schools:=collect(School) by City | sort City' schools.zson
```

#### Output:
```mdtest-output head
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

To count the number of records in each of our example data sources:

```mdtest-command dir=zed-sample-data/edu/zson
zq -z 'count()' schools.zson && zq -z 'count()' testscores.zson && zq -z 'count()' webaddrs.zson
```

#### Output:
```mdtest-output
{count:17686(uint64)}
{count:2331(uint64)}
{count:2223(uint64)}
```

#### Example #2:

The `Website` field is known to be in our school and website address data
sources, but not in the test score data. To confirm this, we can count across
all data sources and specify the named field.

```mdtest-command dir=zed-sample-data/edu/zson
zq -z 'count(Website)' *
```

```mdtest-output
{count:19909(uint64)}
```

Since `17686 + 2223 = 19909`, the count result is what we expected.

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

To see an approximate count of unique school names in our sample data set:

```mdtest-command dir=zed-sample-data/edu/zson
zq -Z 'countdistinct(School)' schools.zson
```

#### Output:
```mdtest-output
{
    countdistinct: 13918 (uint64)
}
```

To see the precise value, which may take longer to execute:

```mdtest-command dir=zed-sample-data/edu/zson
zq -Z 'count() by School | count()' schools.zson
```

#### Output:
```mdtest-output
{
    count: 13876 (uint64)
}
```

Here we saw the approximation was "off" by 0.3%.

---

### `max`

|                           |                                                                |
| ------------------------- | -------------------------------------------------------------- |
| **Description**           | Return the maximum value of a specified field. Non-numeric values (including `null`) are ignored. |
| **Syntax**                | `max(<field-name>)`                                            |
| **Required<br>arguments** | `<field-name>`<br>The name of a field.                         |
| **Optional<br>arguments** | None                                                           |

#### Example:

To see the highest reported math test score:

```mdtest-command dir=zed-sample-data/edu/zson
zq -f table 'max(AvgScrMath)' testscores.zson
```

#### Output:
```mdtest-output
max
699
```

---

### `min`

|                           |                                                                |
| ------------------------- | -------------------------------------------------------------- |
| **Description**           | Return the minimum value of a specified field. Non-numeric values (including `null`) are ignored. |
| **Syntax**                | `min(<field-name>)`                                            |
| **Required<br>arguments** | `<field-name>`<br>The name of a field.                         |
| **Optional<br>arguments** | None                                                           |

#### Example:

To see the lowest reported math test score:

```mdtest-command dir=zed-sample-data/edu/zson
zq -f table 'min(AvgScrMath)' testscores.zson
```

#### Output:
```mdtest-output
min
289
```

---

### `or`

|                           |                                                                |
| ------------------------- | -------------------------------------------------------------- |
| **Description**           | Returns the boolean value `true` if the provided expression evaluates to `true` for one or more inputs. Contrast with [`and`](#and). |
| **Syntax**                | `or(<expression>)`                                             |
| **Required<br>arguments** | `<expression>`<br>A valid Zed [expression](expressions.md). |
| **Optional<br>arguments** | None                                                           |

#### Example:

Many of the school records in our sample data include websites, but many do
not. The following query shows the cities for which at least one school has
a listed website.

```mdtest-command dir=zed-sample-data/edu/zson
zq -Z 'has_at_least_one_school_website:=or(Website!=null) by City | sort City' schools.zson
```

#### Output:
```mdtest-output head
{
    City: "Acampo",
    has_at_least_one_school_website: true
}
{
    City: "Acton",
    has_at_least_one_school_website: true
}
{
    City: "Acton, CA",
    has_at_least_one_school_website: true
}
{
    City: "Adelanto",
    has_at_least_one_school_website: true
}
{
    City: "Adin",
    has_at_least_one_school_website: false
}
...
```

---

### `sum`

|                           |                                                                |
| ------------------------- | -------------------------------------------------------------- |
| **Description**           | Return the total sum of the values of a specified field. Non-numeric values (including `null`) are ignored. |
| **Syntax**                | `sum(<field-name>)`                                            |
| **Required<br>arguments** | `<field-name>`<br>The name of a field.                         |
| **Optional<br>arguments** | None                                                           |

#### Example:

To calculate the total of all the math, reading, and writing test scores
across all schools:

```mdtest-command dir=zed-sample-data/edu/zson
zq -Z 'AllMath:=sum(AvgScrMath),AllRead:=sum(AvgScrRead),AllWrite:=sum(AvgScrWrite)' testscores.zson
```

#### Output:
```mdtest-output
{
    AllMath: 840488 (uint64),
    AllRead: 832260 (uint64),
    AllWrite: 819632 (uint64)
}
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

```mdtest-command dir=zed-sample-data/edu/zson
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
