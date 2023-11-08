---
sidebar_position: 2
sidebar_label: Installation
---

# Installation

Several options for installing `zq` and/or `zed` are available:
* [HomeBrew](#homebrew) for Mac or Linux,
* [Binary Download](#binary-download), or
* [Build from Source](#building-from-source).

To install the Zed Python client, see the
[Python library documentation](libraries/python.md).

## Homebrew

On macOS and Linux, you can use [Homebrew](https://brew.sh/) to install `zq`:

```bash
brew install brimdata/tap/zq
```

Similarly, to install `zed` for working with Zed lakes:
```bash
brew install brimdata/tap/zed
```

Once installed, run a [quick test](#quick-tests).

## Binary Download

We offer pre-built binaries for macOS, Windows and Linux for both x86 and arm
architectures in the Zed [Github Release page](https://github.com/brimdata/zed/releases).

Each archive includes the build for `zq` and `zed`.

Once installed, run a [quick test](#quick-tests).

## Building from source

If you have Go installed, you can easily build `zed` from source:

```bash
go install github.com/brimdata/zed/cmd/{zed,zq}@latest
```

This installs the `zed` and `zq` binaries in your `$GOPATH/bin`.

> If you don't have Go installed, download and install it from the
> [Go install page](https://golang.org/doc/install). Go 1.21 or later is
> required.

Once installed, run a [quick test](#quick-tests).

## Quick Tests

`zq` and `zed` are easy to test as they are completely self-contained
command-line tools and require no external dependendies to run.

### Test zq

To test `zq`, simply run this command in your shell:
```mdtest-command
echo '"hello, world"' | zq -z -
```
which should produce
```mdtest-output
"hello, world"
```

### Test zed

To test `zed`, we'll make a lake in `./scratch`, load data, and query it
as follows:
```
export ZED_LAKE=./scratch
zed init
zed create Demo
echo '{s:"hello, world"}' | zed load -use Demo -
zed query "from Demo"
```
which should display
```
{s:"hello, world"}
```
Alternatively, you can run a Zed lake service, load it with data using `zed load`,
and hit the API.

In one shell, run the server:
```
zed init -lake scratch
zed serve -lake scratch
```
And in another shell, run the client:
```
zed create Demo
zed use Demo
echo '{s:"hello, world"}' | zed load -
zed query "from Demo"
```
which should also display
```
{s:"hello, world"}
```
