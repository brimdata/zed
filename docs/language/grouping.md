# Grouping

> **Note:** Many examples below use the
> [educational sample data](../../testdata/edu).

Zed includes _grouping_ options that partition the input stream into batches
that are aggregated separately based on field values. Grouping is most often
used with [aggregate functions](aggregate-functions.md). If explicit
grouping is not used, an aggregate function will operate over all records in the
input stream.

Below you will find details regarding the available grouping mechanisms and
tips for their effective use.

- [Value Grouping - `by`](#value-grouping---by)
- [Note: Undefined Order](#note-undefined-order)

# Value Grouping - `by`

To create batches of records based on the values of fields or the results of
[expressions](expressions.md), specify
`by <field-name | name:=expression> [, <field-name | name:=expression> ...]`
after invoking your aggregate function(s).

#### Example #1:

The simplest example summarizes the unique values of the named field(s), which
requires no aggregate function. To see the different categories of status for
the schools in our example data:

```mdtest-command dir=testdata/edu
zq -z 'by StatusType | sort' schools.zson
```

#### Output:
```mdtest-output
{StatusType:"Active"}
{StatusType:"Closed"}
{StatusType:"Merged"}
{StatusType:"Pending"}
```

If you work a lot at the UNIX/Linux shell, you might have sought to accomplish
the same via a familiar, verbose idiom. This works in Zed, but the `by`
shorthand is preferable.

```mdtest-command dir=testdata/edu
zq -z 'cut StatusType | sort | uniq' schools.zson
```

#### Output:
```mdtest-output
{StatusType:"Active"}
{StatusType:"Closed"}
{StatusType:"Merged"}
{StatusType:"Pending"}
```

#### Example #2:

By specifying multiple comma-separated field names, one batch is formed for each
unique combination of values found in those fields. To see the average reading
test scores and school count for each county/district pairing:

```mdtest-command dir=testdata/edu
zq -f table 'avg(AvgScrRead),count() by cname,dname | sort -r count' testscores.zson
```

#### Output:
```mdtest-output head
cname           dname                                              avg                count
Los Angeles     Los Angeles Unified                                416.83522727272725 202
San Diego       San Diego Unified                                  472                44
Alameda         Oakland Unified                                    414.95238095238096 27
San Francisco   San Francisco Unified                              454.36842105263156 26
...
```

#### Example #3:

Instead of a simple field name, any of the comma-separated `by` groupings could
be based on the result of an [expression](expressions.md). The
expression must be preceded by the name of the expression result
for further processing/presentation downstream in your Zed pipeline.

To see a count of how many school names of a particular character length
appear in our example data:

```mdtest-command dir=testdata/edu
zq -f table 'count() by Name_Length:=len(School) | sort -r' schools.zson
```

#### Output:
```mdtest-output head
Name_Length count
89          2
85          2
84          2
83          1
...
```

#### Example #4

The fields referenced in a `by` grouping may or may not be present, or may be
inconsistently present, in a given record and the grouping will still have effect.
When a value is missing for a specified key, it will appear as `error("missing")`.

For instance, if we'd made an typographical error in our
[prior example](#example-2) when attempting to reference the `dname` field,
the misspelled column would appear as embedded missing errors:

```mdtest-command dir=testdata/edu
zq -f table 'avg(AvgScrRead),count() by cname,dnmae | sort -r count' testscores.zson
```

#### Output:
```mdtest-output head
cname           dnmae avg                count
Los Angeles     -     450.83037974683543 469
San Diego       -     496.74789915966386 168
San Bernardino  -     465.11764705882354 117
Riverside       -     463.8170731707317  110
Orange          -     510.91011235955057 107
...
```
# Note: Undefined Order

The order of results from a grouped aggregation is undefined. If you want to
ensure a specific order, a [`sort` operator](operators.md#sort)
should be used downstream of the aggregation in the Zed pipeline.
It is for this reason that our examples above all included an explicit
`| sort` at the end of each pipeline.
