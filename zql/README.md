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

[main](main) contains a simple test program that takes zql queries typed
on the command line, parses them with both parsers, and prints out both
results so you can compare for accuracy.  Type this to run the test
program:

`go run ./main`

## Development

For development, each time you make changes to zql.peg, you need to
rebuild the parser and run the test program.  There is a makefile target
to simplify this workflow.  Just type:

`make run`
