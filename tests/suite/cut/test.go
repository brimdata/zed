package cut

import (
	"github.com/mccanne/zq/tests/test"
)

var Test = test.Detail{
	Name:     "cut",
	Query:    "* | cut foo",
	Input:    input,
	Format:   "table",
	Expected: expected,
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
