# Zed parser

This directory contains the Zed parser implemented in PEG.

There is a single PEG input file that works with both
[pigeon](https://github.com/mna/pigeon), which is Go based, and
[pegjs](https://pegjs.org/), which is JavaScript based.  This allows us
to embed a Zed compiler into either JavaScript or Go.

The single parser file is run through the C pre-processor allowing
macro and ifdef logic to create the two variants of PEG.

## Install

You need pegjs, pigeon, and goimports to build the parsers.  To install
them, run:

```
go get github.com/mna/pigeon golang.org/x/tools/cmd/goimports
npm install -g pegjs
```

## Build

To build the parsers, just run make:

`make`

This will run the C pre-processor to make the two PEG files and run
pigeon and pegjs to create the two parsers.

## Testing

The [zed dev compile command](../../cmd/zc) can be used for easily testing the output of
the Zed parser.

## Development

During development, the easiest way to run the parser
is with this `make` command at the root of this repository:
```
make peg
```
This will ensure the PEG-generated JavaScript and Go parsers are up to date
with `parser.peg`

To update the parser and launch the `zc -repl`, your can run `make peg-run`.
