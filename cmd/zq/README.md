# `zq`

`zq` is a command-line tool to search, analyze, and transform structured logs. 
 It evaluates [ZQL ](../.../zql/docs/README.md) queries against input log
  files, producing an output log stream in the [ZNG](../../zng/docs/spec.md)
  format by default. For all `zq` options, see the built-in help by running:

```
zq help
```

## Examples

Here are a few examples using a small Zeek formatted log file, `conn.log
`, located in this directory. See the
[zq-sample-data repo](https://github.com/brimsec/zq-sample-data) for more test
data, which is used in the examples in the
[query language documentation](../../zql/docs/README.md).

To cut the columns of a Zeek "conn" log like `zeek-cut` does, run:
```
zq "* | cut ts,id.orig_h,id.orig_p" conn.log
```
The "`*`" tells `zq` to match every line, which is sent to the `cut` processor
using the UNIX-like pipe syntax.

When looking over everything like this, you can omit the search pattern
as a shorthand and simply type:
```
zq "cut ts,id.orig_h,id.orig_p" conn.log
```

The default output is a ZNG file.  If you want just the tab-separated lines
like `zeek-cut`, you can specify text output:
```
zq -f text "cut ts,id.orig_h,id.orig_p" conn.log
```
If you want the old-style Zeek [ASCII TSV](https://docs.zeek.org/en/stable/examples/logs/)
log format, run the command with the `-f` flag specifying `zeek` for the output
format:
```
zq -f zeek "cut ts,id.orig_h,id.orig_p" conn.log
```
You can use an aggregate function to summarize data over one or
more fields, e.g., summing field values, counting, or computing an average.
```
zq "sum(orig_bytes)" conn.log
zq "orig_bytes > 10000 | count()" conn.log
zq "avg(orig_bytes)" conn.log
```

The [ZNG specification](../../zng/docs/spec.md) describes the significance of
 the
`_path` field.  By leveraging this, diverse Zeek logs can be combined into a single
file.
```
zq *.log > all.tzng
```

### Comparisons

Revisiting the `cut` example shown above:

```
zq -f text "cut ts,id.orig_h,id.orig_p" conn.log
```

This is functionally equivalent to the `zeek-cut` command-line:

```
zeek-cut ts id.orig_h id.orig_p < conn.log
```

If your Zeek events are stored as JSON and you are accustomed to querying with `jq`,
the equivalent would be:

```
jq -c '. | { ts, "id.orig_h", "id.orig_p" }' conn.ndjson
```

Comparisons of other simple operations and their relative performance are described
at the [performance](../../performance/README.md) page.


## Formats

| Format | Read | Auto-Detect | Write | Description |
|--------|------|-------------|-------|-------------|
| zng | yes | yes | yes | [ZNG specification](../../zng/docs/spec.md) |
| tzng | yes | yes | yes | [TZNG specification](../../zng/docs/spec.md#4-zng-text-format-tzng) |
| ndjson | yes | yes | yes | Newline delimited JSON records |
| zeek  | yes | yes | yes | [Zeek compatible](https://docs.zeek.org/en/stable/examples/logs/) tab separated values |
| zjson | yes | yes | yes | Zeek JSON |
| parquet | yes | no | no | [Parquet file format](https://github.com/apache/parquet-format#file-format)
| table | no | no | yes | table output, with column headers |
| text | no | no | yes | space separated output |
| types | no | no | * | Used to output text description of read record types |
