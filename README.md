# zq

zq is a command-line tool for processing
[zeek](https://www.zeek.org) logs.  If you are familiar with
[zeek-cut](https://github.com/zeek/zeek-aux/tree/master/zeek-cut),
you can think of zq as zeek-cut on steroids.  If you missed
[the name change](https://blog.zeek.org/2018/10/renaming-bro-project_11.html),
zeek was formerly known as "bro".

zq is comprised of
* an [execution engine](pkg/proc) for log pattern search and analytics,
* a [query language](pkg/zql/README.md) that compiles into a program that runs on
the execution engine, and
* an open specification for structured logs called [zson](pkg/zson/docs/spec.md).

zq takes zeek/zson logs as input and filters, transforms, and performs
analytics using the
[zq log query language](pkg/zql/README.md),
producing a log stream as its output.

## Install

We don't yet distribute pre-built binaries, so to install zq, you must
clone the repo and compile the source.
To install the binaries in `$GOPATH/bin`, grab this repo and
execute a good old-fashioned make install:

```
git clone https://github.com/mccanne/zq
cd zq
make install
```
## Usage

For zq command usage, see the built-in help by running
```
zq help
```
zq program syntax and semantics are documented in
[zq language README](pkg/zql/README.md)

### Examples

Here are a few examples.

To cut the columns of a conn log like
[zeek-cut](https://github.com/zeek/zeek-aux/tree/master/zeek-cut) does, run:
```
zq conn.log "* | cut ts,id.orig_h,id.orig_p"
```
The "*" tells zq to match every line, which is sent to the cut processor
using the unix-like pipe syntax.

The default output is a zson file.  If you want just the tab-separated lines
like zeek-cut, you can specify text output:
```
zq -f text conn.log "* | cut ts,id.orig_h,id.orig_p"
```
If you want the old-style zeek log format, run the command with the -f flag
specifying "zeek" for the output format:
```
zq -f zeek conn.log "* | cut ts,id.orig_h,id.orig_p"
```
To summarize data, you can use an aggregate function to summarize data over one or
more fields, e.g., summing field, counting, or computing an average.

TBD: keep going here... explain _path, zq *.log > all.zson

## Development

Zq is a [Go module](https://github.com/golang/go/wiki/Modules), so
dependencies are specified in the [go.mod file](/go.mod) and managed
automatically by commands like `go build` and `go test`.  No explicit
fetch commands are necessary.  However, you must set the environment
variable `GO111MODULE=on` if your repo is at
`$GOPATH/src/github.com/mccanne/zq`.

Zq currently requires Go 1.13 or later so make sure your install is up to date.

When go.mod or its companion go.sum is modified during development, run
`go mod tidy` and then commit the changes to both files.

To use a local checkout of a dependency, use `go mod edit`:
```
go mod edit -replace=github.com/org/repo=../repo
```

Note that local checkouts must have a go.mod file, so it may be
necessary to create a temporary one:
```
echo 'module github.com/org/repo' > ../repo/go.mod
```

### Testing

Before any PRs are merged to master, all tests must pass.

To run unit tests in your local repo, execute
```
make test-unit
```

And to run system tests, execute
```
make test-system
```

### Profiling

To use the [Go profiler ](https://golang.org/pkg/net/http/pprof/) to see where CPU
is being used, see the built-in help for the profiling command *-P*.

This will output a pprof command that you can view as follows:

```
go tool pprof -http localhost:8081 localhost:9867
open http://localhost:8081/ui/
```

The flame graph is usually pretty helpful.

## Contributing

zq is developed on github by its community. We welcome contributions.

Feel free to
[post an issue](https://github.com/mccanne/zq/issues),
fork the repo, or send us a pull request.

zq is early in its life cycle and will be expanding quickly.  Please star and/or
watch the repo so you can follow and track our progress.

In particular, we will be adding many more processors and aggregate functions.
If you want a fun small project to help out, pick some functionality that is missing and
add a processor in
[zq/pkg/proc](pkg/proc)
or an aggregate function in
[zq/pkg/reducer](pkg/reducer).
