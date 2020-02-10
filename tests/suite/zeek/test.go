package zeek

import (
	"github.com/brimsec/zq/pkg/test"
)

const log = `
#separator \x09
#set_separator	,
#empty_field	(empty)
#unset_field	-
#path	conn
#fields	s	e	vi	vs
#types	string	enum	vector[int]	vector[string]
foo	bar	1,2,3	a,b,c
`

// Test that Zeek types that get special handling for compatibility with
// the ZNG type system are handled correctly (i.e., that the Zeek types
// are preserved on a pass through zq)
var Test = test.Internal{
	Name:         "Zeek types",
	Query:        "*",
	Input:        test.Trim(log),
	OutputFormat: "zeek",
	Expected:     test.Trim(log),
}
