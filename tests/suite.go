package tests

import (
	"github.com/brimsec/zq/pkg/test"
	"github.com/brimsec/zq/tests/suite/jsontype"
	"github.com/brimsec/zq/tests/suite/zeek"
)

var RootDir = "./test-root"

var internals = []test.Internal{
	zeek.UnionIncompat,
	zeek.UnionInput,
	zeek.RecordInput,
	zeek.ComplexArrayIncompat,
	zeek.ComplexArrayInput,
}

var scripts = []test.Shell{
	jsontype.Test,
	jsontype.TestInferPath,
	jsontype.TestSet,
	jsontype.TestNoTs,
	jsontype.TestTs,
}
