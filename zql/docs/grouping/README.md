# Grouping

All [aggregate functions](../aggregate-functions/README.md) may be invoked with
grouping options that define the batches on which an aggregate function will
operate. If explicit grouping is not used, an aggregate function will operate
over all events in the input stream.

Below you will find details regarding the available grouping mechanisms and
tips for their effective used.

# Undefined Order

The order of results from a grouped aggregation are undefined. If you want to
ensure a specific order, a [`sort` processor](../processors/README.md#sort)
should be used downstream of the aggregation.

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

# Grouping By Time (`every`)

To create batches of events close together by time, specify `every <duration>`
before invoking an aggregate function.

The units of `<duration>` may be expressed in any of the following units of
time, with available abbrevations.

| **Unit**  | **Abbrevation (optional) **  |
|-----------|------------------------------|
| `seconds` | `second`, `secs`, `sec`, `s` |
| `minutes` | `minute`, `mins`, `min`, `m` |
| `hours`   | `hour`, `hrs`, `hr`, `h`     |
| `days`    | `day`, `d`                   |
| `weeks`   | `week`, `wks`, `wk`, `w`     |

The `<duration>` may also be preceded by a numeric value. If the numeric value
is absent, a value of `1` is assumed.

#### Example #1:

To see the total number of bytes originated across all connections every 5
minutes:

```zq-command
zq -f table 'every 5 minutes sum(orig_bytes) | sort -r ts' conn.log.gz
```

#### Output:
```zq-output
TS                SUM
1521912900.000000 5294580
1521912600.000000 31210926
1521912300.000000 37266841
1521912000.000000 36370908
1521911700.000000 70701729
```

#### Example #2:


