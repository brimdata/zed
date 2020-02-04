package errors

import (
	"github.com/mccanne/zq/pkg/test"
	"github.com/mccanne/zq/zng/resolver"
)

const inputEmptyUnionType = `
#0:record[a:union[]]
0:[0:1;]
`

var EmptyUnionType = test.Internal{
	Name:        "emptyuniontype",
	Query:       "*",
	Input:       test.Trim(inputEmptyUnionType),
	InputFormat: "zng",
	ExpectedErr: resolver.ErrEmptyTypeList,
}

const inputEmptySetType = `
#0:record[a:set[]]
0:[0:1;]
`

var EmptySetType = test.Internal{
	Name:        "emptysettype",
	Query:       "*",
	Input:       test.Trim(inputEmptySetType),
	InputFormat: "zng",
	ExpectedErr: resolver.ErrEmptyTypeList,
}
