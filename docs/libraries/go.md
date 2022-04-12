# Go

The Zed system was developed in Go so support for Go clients is
fairly comprehensive.  That said, the code-embedded documentation of exported
package functions is scant and we are actively working to document
the functions of the key Go packages.

Also, our focus for the Go client packages has been on supporting
the core Zed implementation.  We intend to develop a Go package that
is easier to use for external clients.  In the meantime, clients
may use the internal Go packages though the APIs are subject to change.

## Installation

The Zed system is structured as a standard Go module so its easy to import into
other Go projects straight from the GitHub repo.

Some of the key packages are:

* [zed](https://pkg.go.dev/github.com/brimdata/zed) - core Zed values and types
* [zson](https://pkg.go.dev/github.com/brimdata/zed/zson) - ZSON support
* [zio](https://pkg.go.dev/github.com/brimdata/zed/zio) - I/O interfaces for Zed following the Reader/Writer patterns
* [zio/zsonio](https://pkg.go.dev/github.com/brimdata/zed/zio/zsonio) - ZSON reader/writer
* [zio/zngio](https://pkg.go.dev/github.com/brimdata/zed/zio/zngio) - ZNG reader/writer
* [lake/api](https://pkg.go.dev/github.com/brimdata/zed/lake/api) - interact with a Zed Lake

To install in your local Go project, simply run:
```
go get github.com/brimdata/zed
```

## Examples

### ZSON Reader

Read ZSON from stdin, derefence field `s`, and print results:
```
package main

import (
	"fmt"
	"log"
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
			log.Fatalln(err)
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
echo '{s:"hello"}{x:123}{s:"world"}' | go run .
```
which produces
```
"hello"
error("missing")
"world"
```

### Local Lake Reader

This example interacts with a Zed lake.  Note that it is straightforward
to support both direct access to a lake via the files (or S3 URL) as well as
access to a lake via a service endpoint.

First, we'll use `zed` to create a lake and load the example data:
```
zed init -lake scratch
zed create -lake scratch Demo
echo '{s:"hello, world"}{x:1}{s:"good bye"}' | zed load -lake scratch -use Demo -
```
Now replace main.go with this code:
```
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/lake/api"
	"github.com/brimdata/zed/pkg/storage"
	"github.com/brimdata/zed/zson"
)

func main() {
	if len(os.Args) != 2 {
		log.Fatalln("URI of Zed lake not provided")
	}
	uri, err := storage.ParseURI(os.Args[1])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	ctx := context.TODO()
	zctx := zed.NewContext()
	var lake api.Interface
	if api.IsLakeService(uri) {
		lake, err = api.OpenRemoteLake(ctx, uri.String())
	} else {
		lake, err = api.OpenLocalLake(ctx, uri)
	}
	if err != nil {
		log.Fatalln("URI of Zed lake not provided")
	}
	reader, err := lake.Query(ctx, nil, "from Demo")
	if err != nil {
		log.Fatalln("URI of Zed lake not provided")
	}
	defer reader.Close()
	for {
		val, err := reader.Read()
		if err != nil {
			log.Fatalln("URI of Zed lake not provided")
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
Now, run this command to interact with the lake via the local file system:
```
go run . ./scratch
```
which should output
```
{s:"hello, world"}
{s:"good bye"}
{x:1}
```
Note that the order of data has changed because the Zed lake stores data
in a sorted order.  Since we did not specify a "pool key" when we created
the lake, it ends up sorting the data by `this`.

### Lake Service Reader

We can use the same code above to talk to a Zed lake server.  All we do is
give it the URI of the service, which by default is on port 9867.

To try this out, first run a Zed service on the scratch lake we created
above:
```
zed serve -lake ./scratch
```
Finally, in another local shell, run the Go program and specify the servie
endpoint we just created:
```
go run . http://localhost:9867
```
and you should again get this result:
```
"hello, world"
"good bye"
error("missing")
```
