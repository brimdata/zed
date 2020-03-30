package tests

import (
	"github.com/brimsec/zq/pkg/test"
	"github.com/brimsec/zq/tests/suite/diropt"
	"github.com/brimsec/zq/tests/suite/errors"
	"github.com/brimsec/zq/tests/suite/jsontype"
	"github.com/brimsec/zq/tests/suite/pcap"
	"github.com/brimsec/zq/tests/suite/utf8"
	"github.com/brimsec/zq/tests/suite/zeek"
)

var RootDir = "./test-root"

var internals = []test.Internal{
	errors.DuplicateFields,
	errors.ErrNotPrimitive,
	errors.ErrNotPrimitiveZJSON,
	errors.ErrNotContainer,
	errors.ErrNotContainerZJSON,
	errors.ErrMissingField,
	errors.ErrExtraField,
	errors.TypeNull,
	errors.EmptySetType,
	errors.EmptyUnionType,
	errors.RedefineAlias,
	zeek.UnionIncompat,
	zeek.UnionInput,
	zeek.RecordInput,
	zeek.ComplexArrayIncompat,
	zeek.ComplexArrayInput,
}

var commands = []test.Exec{
	utf8.Exec,
	// this test doesn't work in circleCI apparently because it does
	// a pipeline...
	//    zq ... | zq ...
	// Since we are reworking this test framework and I don't want to spend
	// time futzing with this, I will leave this here to be added when the
	// framework is reworked.
	//ndjson.Exec,
}

var scripts = []test.Shell{
	errors.Combiner,
	diropt.Test,
	diropt.Test2,
	jsontype.Test,
	jsontype.TestInferPath,
	jsontype.TestSet,
	jsontype.TestNoTs,
	pcap.Test1,
	pcap.Test2,
	pcap.Test3,
	pcap.Test4,
	pcap.Test5,
	pcap.Test6,
}
