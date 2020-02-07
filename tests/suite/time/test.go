package time

import (
	"github.com/brimsec/zq/pkg/test"
)

var Internal = test.Internal{
	Name:         "count",
	Query:        "* | every 1d count()",
	Input:        test.Trim(input),
	OutputFormat: "zng",
	Expected:     test.Trim(expected),
}

// This log is path-less in order to make "ts" the first column and
// that verify we handle this case correctly.
const input = `
#separator \x09
#set_separator	,
#empty_field	(empty)
#unset_field	-
#fields	ts
#types	time
1425565514.419939
`

const expected = `
#0:record[ts:time,count:count]
0:[1425513600;1;]`
