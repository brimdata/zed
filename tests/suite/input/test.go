package input

import (
	"github.com/mccanne/zq/test"
)

var Internal = test.Internal{
	Name:     "input",
	Query:    "*",
	Input:    test.Trim(input),
	Format:   "zson",
	Expected: test.Trim(expected),
}

const input = `
{ "string1": "value1", "string2": "value1", "int1": 4, "bool1": true }
{ "int1": 4, "bool1": true, "string2": "value2", "string1": "value2" }

{ "obj1": { "null1": null } }
`

const expected = `
#0:record[bool1:bool,int1:double,string1:string,string2:string]
0:[true;4;value1;value1;]
0:[true;4;value2;value2;]
#1:record[obj1:record[null1:string]]
1:[[-;]]`
