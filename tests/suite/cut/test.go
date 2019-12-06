package cut

import (
	"github.com/mccanne/zq/pkg/test"
)

var Internal = test.Internal{
	Name:     "cut",
	Query:    "* | cut foo",
	Input:    test.Trim(input),
	Format:   "table",
	Expected: test.Trim(expected),
}

var Exec = test.Exec{
	Name:     "cut",
	Command:  `zq -f table "* | cut foo" -`,
	Input:    test.Trim(input),
	Expected: test.Trim(expected),
}

const input = `
#separator \x09
#set_separator	,
#empty_field	(empty)
#unset_field	-
#path	conn
#fields	foo	bar
#types	string	string
key1 value1	key2 value1
key1 value2	key2 value2
`
const expected = `
FOO
key1 value1
key1 value2`
