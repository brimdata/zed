---
sidebar_position: 1
sidebar_label: Install
---

# Installing `zed` and `zq`

## Homebrew

On macOS and Linux, you can also use [Homebrew](https://brew.sh/) to install `zq`:

```bash
brew install brimdata/tap/zq
```

Similarly `zed` can be installed with:

```bash
brew install brimdata/tap/zed
```

## Pre-built Binaries

We offer pre-built binaries for macOS, Windows and Linux for both x86 and arm
architectures in the Zed [Github Release page](https://github.com/brimdata/zed/releases).

Each archive includes the build for `zq` and `zed`.

## Building from source

It's also easy to build `zed` from source:

```bash
git clone https://github.com/brimdata/zed
cd zed
make install
```

This installs the `zed` and `zq` binaries in your `$GOPATH/bin`.

> If you don't have Go installed, download and install it from the
> [Go install page](https://golang.org/doc/install). Go version 1.17 or later is
> required.
