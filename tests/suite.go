package tests

import (
	"github.com/brimsec/zq/pkg/test"
	"github.com/brimsec/zq/tests/suite/jsontype"
	"github.com/brimsec/zq/tests/suite/pcap"
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
	pcap.Test2,
	pcap.Test3,
	pcap.Test4,
	pcap.Test5,
	pcap.Test6,
	pcap.Test7,
}
