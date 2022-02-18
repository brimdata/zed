# Contributing

Thank you for contributing to `zed`!

Per common practice, please [open an issue](https://github.com/brimdata/zed/issues)
before sending a pull request.  If you think your ideas might benefit from somerefinementrefinement via Q&A, come talk to us on [Slack](https://www.brimdata.io/join-slack/) as well.

`zed` is early in its life cycle and will be expanding quickly.  Please star and/or
watch the repo so you can follow and track our progress.

In particular, we will be adding many more processors and aggregate functions.
If you want a fun, small project to help out, pick some functionality that is missing and
add a processor in [runtime/op](runtime/op) or an aggregate function
in [runtime/expr/agg](runtime/expr/agg).


## Development

`zed` requires Go 1.17 or later, and uses [Go modules](https://github.com/golang/go/wiki/Modules).
Dependencies are specified in the [`go.mod` file](./go.mod) and fetched
automatically by commands like `go build` and `go test`.  No explicit
fetch commands are necessary.  However, you must set the environment
variable `GO111MODULE=on` if your repo is at
`$GOPATH/src/github.com/brimdata/zed`.

When `go.mod` or its companion `go.sum` are modified during development, run
`go mod tidy` and then commit the changes to both files.

To use a local checkout of a dependency, use `go mod edit`:
```
go mod edit -replace=github.com/org/repo=../repo
```

### Testing

Before any PRs are merged to main, all tests must pass.

To run unit tests in your local repo, execute:
```
make test-unit
```

System tests require Python 3.3 or better.  To run them, execute:
```
make test-system
```
