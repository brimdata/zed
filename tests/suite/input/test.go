package input

import (
	"github.com/brimsec/zq/pkg/test"
)

var JSON = test.Internal{
	Name:         "json input",
	Query:        "*",
	Input:        test.Trim(jsonInput),
	OutputFormat: "zng",
	Expected:     test.Trim(jsonExpected),
}

const jsonInput = `
{ "string1": "value1", "string2": "value1", "int1": 4, "bool1": true }
{ "int1": 4, "bool1": true, "string2": "value2", "string1": "value2" }

{ "obj1": { "null1": null } }
`

const jsonExpected = `
#0:record[bool1:bool,int1:float64,string1:string,string2:string]
0:[T;4;value1;value1;]
0:[T;4;value2;value2;]
#1:record[obj1:record[null1:string]]
1:[[-;]]`

// Test that an escaped backslash is handled correctly.
const backslashInput = `
#separator \x09
#set_separator	,
#empty_field	(empty)
#unset_field	-
#path	Backslash
#fields	my_str
#types	string
co\\m
`

const backslashExpected = `
#0:record[_path:string,my_str:bstring]
0:[Backslash;co\\m;]
`

var Backslash = test.Internal{
	Name:         "escaped backslash",
	Query:        "*",
	Input:        test.Trim(backslashInput),
	OutputFormat: "zng",
	Expected:     test.Trim(backslashExpected),
}
