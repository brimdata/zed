package errors

import (
	"github.com/mccanne/zq/pkg/test"
	"github.com/mccanne/zq/zbuf"
)

const inputErrNotScalar = `
#0:record[a:string]
0:[[1;]]
`

// Container/scalar type checks are done while parsing, so
// ErrNotScalar and ErrNotContainer get dual zng and zjson tests. The
// other type checks are done after parsing and dont the dual tests.

var ErrNotScalar = test.Internal{
	Name:        "container where scalar expected",
	Query:       "*",
	Input:       test.Trim(inputErrNotScalar),
	InputFormat: "zng",
	ExpectedErr: zbuf.ErrNotScalar,
}

const inputErrNotScalarZJSON = `{"id":0,"type":[{"name":"a","type":"string"}],"values":[["1"]]}`

var ErrNotScalarZJSON = test.Internal{
	Name:        "container where scalar expected (zjson)",
	Query:       "*",
	Input:       test.Trim(inputErrNotScalarZJSON),
	InputFormat: "zjson",
	ExpectedErr: zbuf.ErrNotScalar,
}

const inputErrNotContainer = `
#0:record[a:record[b:string]]
0:[1;]
`

var ErrNotContainer = test.Internal{
	Name:        "scalar where container expected",
	Query:       "*",
	Input:       test.Trim(inputErrNotContainer),
	InputFormat: "zng",
	ExpectedErr: zbuf.ErrNotContainer,
}

const inputErrNotContainerZJSON = `{"id":0,"type":[{"name":"a","type":[{"name":"b","type":"string"}]}],"values":["1"]}`

var ErrNotContainerZJSON = test.Internal{
	Name:        "scalar where container expected (zjson)",
	Query:       "*",
	Input:       test.Trim(inputErrNotContainerZJSON),
	InputFormat: "zjson",
	ExpectedErr: zbuf.ErrNotContainer,
}

const inputErrExtraField = `
#0:record[a:string]
0:[1;2;]
`

var ErrExtraField = test.Internal{
	Name:        "extra field",
	Query:       "*",
	Input:       test.Trim(inputErrExtraField),
	InputFormat: "zng",
	ExpectedErr: zbuf.ErrExtraField,
}

const inputErrMissingField = `
#0:record[a:string,b:string]
0:[1;]
`

var ErrMissingField = test.Internal{
	Name:        "missing field",
	Query:       "*",
	Input:       test.Trim(inputErrMissingField),
	InputFormat: "zng",
	ExpectedErr: zbuf.ErrMissingField,
}
