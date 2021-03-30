# zql

This directory contains the zql parser implemented in PEG.

There is a single PEG input file that works with both
[pigeon](https://github.com/mna/pigeon), which is Go based, and
[pegjs](https://pegjs.org/), which is JavaScript based.  This allows us
to embed a zql compiler into either JavaScript or Go.

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

The [ast command](../../cmd/zast) can be used for easiliy testing the output of
the zql parser.

## Development

During development, the easiest way to run the parser
is with this `make` command at the root of the `zq repo`:
```
make peg
```
This will ensure the PEG-generated javascript and Go parsers are up to date
with `zql.peg` and will launch the `ast -repl` so you can type zql queries
and see the AST output during development.
