package tests

import (
	"github.com/brimsec/zq/pkg/test"
	"github.com/brimsec/zq/tests/suite/errors"
	"github.com/brimsec/zq/tests/suite/jsontype"
	"github.com/brimsec/zq/tests/suite/pcap"
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

var scripts = []test.Shell{
	errors.Combiner,
	errors.StopErrStop,
	errors.StopErrContinue,
	errors.StopErrContinueMid,
	jsontype.Test,
	jsontype.TestInferPath,
	jsontype.TestSet,
	jsontype.TestNoTs,
	jsontype.TestTs,
	pcap.Test2,
	pcap.Test3,
	pcap.Test4,
	pcap.Test5,
	pcap.Test6,
	pcap.Test7,
}
