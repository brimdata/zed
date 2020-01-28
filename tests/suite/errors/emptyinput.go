package errors

import (
	"github.com/mccanne/zq/pkg/test"
)

var Exec = test.Exec{
	Name:     "empty input",
	Command:  `echo "" | zq -`,
	Input:    test.Trim(input),
	Expected: "",
}

const input = `
#0:record[_path:string,ts:time]
0:[conn;1425565514.419939;]
`
