package tests

import (
	"github.com/mccanne/zq/pkg/test"
	"github.com/mccanne/zq/tests/suite/count"
	"github.com/mccanne/zq/tests/suite/cut"
	"github.com/mccanne/zq/tests/suite/diropt"
	"github.com/mccanne/zq/tests/suite/errors"
	"github.com/mccanne/zq/tests/suite/filter"
	"github.com/mccanne/zq/tests/suite/format"
	"github.com/mccanne/zq/tests/suite/input"
	"github.com/mccanne/zq/tests/suite/reducer"
	"github.com/mccanne/zq/tests/suite/regexp"
	"github.com/mccanne/zq/tests/suite/sort"
	"github.com/mccanne/zq/tests/suite/time"
	"github.com/mccanne/zq/tests/suite/utf8"
)

var RootDir = "./test-root"

var internals = []test.Internal{
	count.Internal,
	cut.Internal,
	format.Internal,
	format.JsonTypes,
	input.JSON,
	input.Backslash,
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
	regexp.Internal,
	filter.EscapedEqual,
	filter.EscapedAsterisk,
	filter.UnescapedAsterisk,
	filter.NullWithNonexistentField,
	filter.NullWithUnsetField,
	reducer.UnsetAvg,
	reducer.UnsetCountDistinct,
	reducer.UnsetCount,
	reducer.UnsetFirst,
	reducer.UnsetLast,
	reducer.UnsetMax,
	reducer.UnsetMin,
	reducer.UnsetSum,
	sort.Internal1,
	sort.Internal2,
	sort.Internal3,
	sort.Internal4_1,
	sort.Internal4_2,
	sort.Internal4_3,
	sort.Internal4_4,
	time.Internal,
}

var commands = []test.Exec{
	cut.Exec,
	errors.Exec,
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
	diropt.Test,
	diropt.Test2,
}
