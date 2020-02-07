package filter

import (
	"github.com/brimsec/zq/pkg/test"
)

const in = `
#0:record[s:string]
0:[A=B;]
0:[A=*;]
0:[-;]
0:[;]
`

const out1 = `
#0:record[s:string]
0:[A=B;]
`

const out2 = `
#0:record[s:string]
0:[A=*;]
`

const out3 = `
#0:record[s:string]
0:[A=B;]
0:[A=*;]
`

var EscapedEqual = test.Internal{
	Name:         "Filter Escaped =",
	Query:        `A\=B`,
	Input:        test.Trim(in),
	OutputFormat: "zng",
	Expected:     test.Trim(out1),
}

var EscapedAsterisk = test.Internal{
	Name:         "Filter Escaped *",
	Query:        `A\=\*`,
	Input:        test.Trim(in),
	OutputFormat: "zng",
	Expected:     test.Trim(out2),
}

var UnescapedAsterisk = test.Internal{
	Name:         "Filter Unescaped *",
	Query:        `A\=*`,
	Input:        test.Trim(in),
	OutputFormat: "zng",
	Expected:     test.Trim(out3),
}

var NullWithNonexistentField = test.Internal{
	Name:     "null with nonexistent field",
	Query:    "not (t=null or t!=null)",
	Input:    in,
	Expected: test.Trim(in),
}

var NullWithUnsetField = test.Internal{
	Name:  "null with unset field",
	Query: "s=null",
	Input: in,
	Expected: test.Trim(`
#0:record[s:string]
0:[-;]
`),
}
