# Go

The Zed system was developed in Go so support for Go clients is
fairly complete.  That said, the code documentation of exported
package functions is scant and we are actively working to document
the functions of the key Go packages.

## Installation

Top-level zed package...

## Library API

XXX refer to key packages:

* [zed](https://pkg.go.dev/github.com/brimdata/zed)
* [zio](https://pkg.go.dev/github.com/brimdata/zed/zio)
* [lake/api](https://pkg.go.dev/github.com/brimdata/zed/lake/api)


## Examples

_Read ZSON, derefence field `s`, and print results._
```
package main

import (
	"fmt"
	"os"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/zio/zsonio"
	"github.com/brimdata/zed/zson"
)

func main() {
	zctx := zed.NewContext()
	reader := zsonio.NewReader(os.Stdin, zctx)
	for {
		val, err := reader.Read()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		if val == nil {
			return
		}
		s := val.Deref("s")
		if s == nil {
			s = zctx.Missing()
		}
		fmt.Println(zson.String(s))
	}
}
```
To build, create a directory for the main package, initialize it,
copy the above code into main.go, fetch the quired zed packages.
```
mkdir example
cd example
cat > main.go < [paste from above]
go get github.com/brimdata/zed
go get github.com/brimdata/zson
go get github.com/brimdata/zio/zsonion
```
To run type:
```
echo '{s:"hello, world"}{x:1}{s:"good bye"}' | go run .
```
which produces
```
"hello, world"
error("missing")
"good bye"
```

_Read ZNG from a Zed lake and do the same as above._

First, create a lake and load the example data:
```
mkdir testlake
zed init -lake testlake
zed create -lake testlake testpool
echo '{s:"hello, world"}{x:1}{s:"good bye"}' | zed load -lake testlake -use testpool -
```
Now replace main.go with this code:
```
TBD
```
