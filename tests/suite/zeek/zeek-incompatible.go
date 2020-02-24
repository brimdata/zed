package zeek

import (
	"github.com/brimsec/zq/pkg/test"
	"github.com/brimsec/zq/zio/zeekio"
)

const inputUnion = `
#0:record[u:union[string,int32]]
0:[0:foo;]
`

var UnionIncompat = test.Internal{
	Name:         "union is incompatible with zeek",
	Query:        "*",
	Input:        test.Trim(inputUnion),
	OutputFormat: "zeek",
	ExpectedErr:  zeekio.ErrIncompatibleZeekType,
}

const inputArray = `
#0:record[a:array[record[s:string]]]
0:[[[foo;]]]
`

var ComplexArrayIncompat = test.Internal{
	Name:         "array of complex type is incompatible with zeek",
	Query:        "*",
	Input:        test.Trim(inputArray),
	OutputFormat: "zeek",
	ExpectedErr:  zeekio.ErrIncompatibleZeekType,
}
