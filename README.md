# `zq` [![CI][ci-img]][ci] [![GoDoc][doc-img]][doc]

`zq` is a command-line tool for searching and analyzing logs,
particularly [Zeek](https://www.zeek.org) logs.  If you are familiar with
[`zeek-cut`](https://github.com/zeek/zeek-aux/tree/master/zeek-cut),
you can think of `zq` as `zeek-cut` on steroids.

`zq` is comprised of:

* an [execution engine](proc) for log pattern search and analytics,
* a [query language](zql/docs/README.md) that compiles into a program that runs on
the execution engine, and
* an open specification for structured logs, called [ZNG](zng/docs/README.md).<br>
(**Note**: The ZNG format is in Alpha and subject to change.)

`zq` takes Zeek/ZNG logs as input and filters, transforms, and performs
analytics using the
[zq query language](zql/docs/README.md),
producing a log stream as its output.

## Install

To install `zq`, you can
clone the repo and compile the source.  For Windows, MacOS, and Linux there are pre-compiled binary [releases](https://github.com/brimsec/zq/releases).

If you don't have Go installed,
download and install it from the [Go downloads page](https://golang.org/dl/).

If you're new to Go, remember to set GOPATH.  A common convention is to create ~/go
and point GOPATH at $HOME/go.

To install the binaries in `$GOPATH/bin`, clone this repo and
execute `make install`:

```
git clone https://github.com/brimsec/zq
cd zq
make install
```
## Usage

For `zq` command usage, see the built-in help by running:
```
zq help
```
`zq` program syntax and semantics are documented in the
[query language README](zql/docs/README.md).

### Examples

Here are a few examples based on a very simple "conn" log from Zeek [(conn.log)](conn.log),
located in this directory. See the
[zq-sample-data repo](https://github.com/brimsec/zq-sample-data) for more
test data, which is used in the examples in the
[query language documentation](zql/docs/README.md).

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

The [ZNG specification](zng/docs/spec.md) describes the significance of the
`_path` field.  By leveraging this, diverse Zeek logs can be combined into a single
file.
```
zq *.log > all.zng
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
at the [performance](performance/README.md) page.


## Contributing

See the [contributing guide](CONTRIBUTING.md) on how you can help improve `zq`!

[doc-img]: https://godoc.org/github.com/brimsec/zq?status.svg
[doc]: https://godoc.org/github.com/brimsec/zq
[ci-img]: https://circleci.com/gh/brimsec/zq.svg?style=svg
[ci]: https://circleci.com/gh/brimsec/zq

## Join the Community

Join our [Public Slack](https://join.slack.com/t/brimsec/shared_invite/zt-cy34xoxg-hZiTKUT~1KdGjlaBIuUUdg) workspace for announcements, Q&A, and to trade tips!
