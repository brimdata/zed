# Contributing

Thank you for contributing to `zq`!

Per [common practice](https://www.thinkful.com/learn/github-pull-request-tutorial/Feel-Free-to-Ask#Feel-Free-to-Ask),
please [open an issue](https://github.com/brimsec/zq/issues) before sending a pull request.  If you
think your ideas might benefit from some refinement via Q&A, come talk to us on
[Slack](https://www.brimsecurity.com/join-slack/) as well.

`zq` is early in its life cycle and will be expanding quickly.  Please star and/or
watch the repo so you can follow and track our progress.

In particular, we will be adding many more processors and aggregate functions.
If you want a fun, small project to help out, pick some functionality that is missing and
add a processor in [zq/proc](proc) or an aggregate function in [zq/expr/agg](expr/agg).


## Development

`zq` requires Go 1.16 or later, and uses [Go modules](https://github.com/golang/go/wiki/Modules).
Dependencies are specified in the [`go.mod` file](./go.mod) and fetched
automatically by commands like `go build` and `go test`.  No explicit
fetch commands are necessary.  However, you must set the environment
variable `GO111MODULE=on` if your repo is at
`$GOPATH/src/github.com/brimsec/zq`.

When `go.mod` or its companion `go.sum` are modified during development, run
`go mod tidy` and then commit the changes to both files.

To use a local checkout of a dependency, use `go mod edit`:
```
go mod edit -replace=github.com/org/repo=../repo
```

### Testing

Before any PRs are merged to master, all tests must pass.

To run unit tests in your local repo, execute:
```
make test-unit
```

System tests require Python 3.3 or better.  To run them, execute:
```
make test-system
```

## Licensing

This repository contains source code that is licensed under either a
BSD-3-Clause license or the Polyform Perimeter license - see the LICENSE.txt
file for specifics.

If you make a contribution to this repository, whether to source code licensed
under the BSD-3-Clause license or the Polyform Perimeter license, you agree
that you are licensing your contribution under the terms of the BSD-3-Clause
license found in LICENSE.txt, and you agree that you have the right to license
your contribution under those license terms. 

### Why the two licenses?

We want to prevent technology giants from using the Polyform Perimeter license
covered code to create replacement offerings of our projects.

The overwhelming majority of Brim or zqd users and developers will not be
restricted by this license, including those using Brim or zqd in commercial
settings.

The use of the source-available Polyform Perimeter license prevents use
cases like:
* Marketing a work as a “as-a-service” style offering for server
  components like zqd, while using material covered under the Polyform
  Perimeter license
* Marketing a work as a replacement for the Brim desktop application,
  while using material covered under the Polyform Perimeter license

We believe users and developers should have access to the source code for our
project, and we need a sustainable business model to continue funding our
work. Using the source-available Polyform Perimeter license on portions
of the source code lets us realize both.
