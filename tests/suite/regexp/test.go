package regexp

import (
	"github.com/brimsec/zq/pkg/test"
)

var Internal = test.Internal{
	Name:         "regexp",
	Query:        "foo*",
	Input:        test.Trim(input),
	OutputFormat: "zng",
	Expected:     test.Trim(expected),
}

const input = `
#0:record[a:string,b:string]
0:[hello;there;]
0:[foox;there;]
0:[hello;foox;]
0:[;foo;]
`

const expected = `
#0:record[a:string,b:string]
0:[foox;there;]
0:[hello;foox;]
0:[;foo;]
`
