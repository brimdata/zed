package errors

import (
	"github.com/mccanne/zq/pkg/test"
	"github.com/mccanne/zq/zng"
)

const inputTypeNull = `
#0:record[foo:null]
0:[bleah;]
`

var TypeNull = test.Internal{
	Name:        "type null",
	Query:       "*",
	InputFormat: "zng",
	Input:       test.Trim(inputTypeNull),
	ExpectedErr: zng.ErrInstantiateNull,
}
