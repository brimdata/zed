# Grouping

All [aggregate functions](../aggregate-functions/README.md) may be invoked with
grouping options that define the batches on which an aggregate function will
operate. If explicit grouping is not used, an aggregate function will operate
over all events in the input stream.

Below you will find details regarding the available grouping mechanisms and
tips for their effective use.

# Undefined Order

The order of results from a grouped aggregation are undefined. If you want to
ensure a specific order, a [`sort` processor](../processors/README.md#sort)
should be used downstream of the aggregate function in the ZQL pipeline.

#### Example:

If we were counting events into 5 minute batches and wanted to see these
results ordered by incrementing timestamp of each batch:

```zq-command
zq -f table 'every 5 minutes count() | sort ts' *.log.gz
```

#### Output:
```zq-output
TS                COUNT
1521911700.000000 441229
1521912000.000000 337264
1521912300.000000 310546
1521912600.000000 274284
1521912900.000000 98755
```

If we'd wanted to see them ordered from lowest to highest event count:

```zq-command
zq -f table 'every 5 minutes count() | sort count' *.log.gz
```

#### Output:
```zq-output
TS                COUNT
1521912900.000000 98755
1521912600.000000 274284
1521912300.000000 310546
1521912000.000000 337264
1521911700.000000 441229
```

# Time Grouping (`every`)

To create batches of events close together by time, specify `every <duration>`
before invoking an aggregate function.

The `<duration>` may be expressed in any of the following units of time. A
numeric value may also precede the unit specification, in which case any of
the shorthand variations may also be used.

| **Unit**  | **Plural shorthand (optional) **  |
|-----------|-----------------------------------|
| `second`  | `seconds`, `secs`, `sec`, `s`     |
| `minute`  | `minutes`, `mins`, `min`, `m`     |
| `hour`    | `hours`, `hrs`, `hr`, `h`         |
| `day`     | `days`, `d`                       |
| `week`    | `weeks`, `wks`, `wk`, `w`         |

#### Example #1:

To see the total number of bytes originated across all connections every
minute:

```zq-command
zq -f table 'every minute sum(orig_bytes) | sort -r ts' conn.log.gz
```

#### Output:
```zq-output head:5
TS                SUM
1521912960.000000 1443272
1521912900.000000 3851308
1521912840.000000 4704644
1521912780.000000 10189155
...
```

#### Example #2:

To see which 30-second interval contained the most events:

```zq-command
zq -f table 'every 30sec count() | sort -r count' *.log.gz
```

#### Output:
```zq-output head:5
TS                COUNT
1521911940.000000 73512
1521911790.000000 59701
1521912000.000000 51229
...
```

# Value Grouping (`by`)

To create batches of events based on the values of one or more fields, specify
`by <field-list>` after invoking the aggregate function. The `<field-list>`
may consist of one or more comma-separate field names or assignments.

#### Example #1:


