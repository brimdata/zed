package zeek

import (
	"github.com/brimsec/zq/pkg/test"
	"github.com/brimsec/zq/zio/zeekio"
)

// Test that trying to output a union to zeek fails.
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

const inputZUnion = `
#separator \x09
#set_separator	,
#empty_field	(empty)
#unset_field	-
#fields	foo
#types	union[string,int]
0:bar
`

var UnionInput = test.Internal{
	Name:         "union is not accepted as an input type in zeek logs",
	Query:        "*",
	InputFormat:  "zeek",
	Input:        test.Trim(inputZUnion),
	OutputFormat: "zeek",
	ExpectedErr:  zeekio.ErrIncompatibleZeekType,
}

const inputRecord = `
#separator \x09
#set_separator	,
#empty_field	(empty)
#unset_field	-
#fields	foo
#types	record[s:string]
bar
`

var RecordInput = test.Internal{
	Name:         "record is not accepted as an input type in zeek logs",
	Query:        "*",
	InputFormat:  "zeek",
	Input:        test.Trim(inputRecord),
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

const inputArray2 = `
#separator \x09
#set_separator	,
#empty_field	(empty)
#unset_field	-
#fields	foo
#types	vector[record[s:string]]
bar
`

var ComplexArrayInput = test.Internal{
	Name:         "vector[record] is not accepted as an input type in zeek logs",
	Query:        "*",
	InputFormat:  "zeek",
	Input:        test.Trim(inputArray2),
	OutputFormat: "zeek",
	ExpectedErr:  zeekio.ErrIncompatibleZeekType,
}
