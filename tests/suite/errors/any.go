package errors

import (
	"github.com/mccanne/zq/pkg/test"
	"github.com/mccanne/zq/zng"
)

const inputTypeAny = `
#0:record[foo:any]
0:[bleah;]
`

var TypeAny = test.Internal{
	Name:        "type any",
	Query:       "*",
	InputFormat: "zng",
	Input:       test.Trim(inputTypeAny),
	ExpectedErr: zng.ErrInstantiateAny,
}
