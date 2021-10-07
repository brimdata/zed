# Data Types

Comprehensive documentation for working with data types in Zed is still a work
in progress. In the meantime, here's a few tips to get started with.

* Values are stored internally and treated in expressions using one of the Zed
  data types described in the
  [Primitive Values](../../formats/zson.md#33-primitive-values) section of the
  ZSON spec.
* Users of [Zeek](../../../zeek/README.md) logs should review the
  [Equivalent Types](../../../zeek/Data-Type-Compatibility.md#equivalent-types)
  table for details on which Zed data types correspond to the
  [data types](https://docs.zeek.org/en/current/script-reference/types.html)
  that appear in Zeek logs.
* Zed allows for [type casting](https://en.wikipedia.org/wiki/Type_conversion)
  by specifying a destination Zed data type followed by the value to be
  converted to that type, enclosed in parentheses.

#### Example:

Consider the following NDJSON file `shipments.ndjson` that contains what
appear to be timestamped quantities of shipped items.

```mdtest-input shipments.ndjson
{"ts":"2021-10-07T13:55:22Z", "quantity": 873}
{"ts":"2021-10-07T17:23:44Z", "quantity": 436}
{"ts":"2021-10-07T23:01:34Z", "quantity": 123}
{"ts":"2021-10-08T09:12:45Z", "quantity": 998}
{"ts":"2021-10-08T12:44:12Z", "quantity": 744}
{"ts":"2021-10-09T20:01:19Z", "quantity": 2003}
{"ts":"2021-10-09T04:16:33Z", "quantity": 977}
{"ts":"2021-10-10T05:04:46Z", "quantity": 3004}
```

This data suffers from a notorious limitation of JSON: The lack of a native
"time" type requires storing timestamps as strings. As a result, if read into
`zq` as the strings they are, these `ts` values are not usable with
time-specific operations in Zed, such as this attempt to use
[time grouping](../grouping#time-grouping---every) to calculate total
quantities shipped per day.

```mdtest-command
zq -f table 'every 1d sum(quantity) | sort ts' shipments.ndjson
```

#### Output:
```mdtest-output
```

However, if we cast the `ts` field to the Zed `time` type, now the
calculation works as expected.

```mdtest-command
zq -f table 'ts:=time(ts) | every 1d sum(quantity) | sort ts' shipments.ndjson
```

#### Output:
```mdtest-output
ts                   sum
2021-10-07T00:00:00Z 1432
2021-10-08T00:00:00Z 1742
2021-10-09T00:00:00Z 2980
2021-10-10T00:00:00Z 3004
```
