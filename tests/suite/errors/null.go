package errors

import (
	"github.com/brimsec/zq/pkg/test"
	"github.com/brimsec/zq/zng"
)

const inputTypeNull = `
#0:record[foo:null]
0:[bleah;]
`

var TypeNull = test.Internal{
	Name:        "type null",
	Query:       "*",
	InputFormat: "tzng",
	Input:       test.Trim(inputTypeNull),
	ExpectedErr: zng.ErrInstantiateNull,
}
