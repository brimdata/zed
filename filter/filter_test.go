package filter_test

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/mccanne/zq/ast"
	"github.com/mccanne/zq/filter"
	"github.com/mccanne/zq/zio/detector"
	"github.com/mccanne/zq/zng"
	"github.com/mccanne/zq/zng/resolver"
	"github.com/mccanne/zq/zql"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Execute one test of a filter by compiling the given filter and
// executing it against the given Record.  Returns an error if the filter
// result does not match expectedResult (or for any other error such as
// failure to parse or compile the filter)
func runTest(filt string, record *zng.Record, expectedResult bool) error {
	// Parse the filter.  Any filter is a valid full zql query,
	// it should parse to an AST with a top-level FilterProc node.
	parsed, err := zql.Parse("", []byte(filt))
	if err != nil {
		return err
	}

	filtProc, ok := parsed.(*ast.FilterProc)
	if !ok {
		return errors.New("expected FilterProc")
	}

	// Compile the filter...
	f, err := filter.Compile(filtProc.Filter)
	if err != nil {
		return err
	}

	// And execute it.
	result := f(record)
	if result == expectedResult {
		return nil
	}

	// Failure!  Try to assemble a useful error message.
	// Just use the zval pretty format of Raw.
	raw := record.Raw.String()
	if expectedResult {
		return fmt.Errorf("Filter \"%s\" should have matched \"%s\"", filt, raw)
	} else {
		return fmt.Errorf("Filter \"%s\" should not have matched \"%s\"", filt, raw)
	}
}

const zngsrc = `
#0:record[stringset:set[bstring]]
#1:record[stringvec:array[bstring]]
#2:record[intset:set[int]]
#3:record[intvec:array[int]]
#4:record[addrset:set[addr]]
#5:record[addrvec:array[addr]]
#6:record[nested:record[field:string]]
#7:record[nested:array[record[field:int]]]
#8:record[nested:record[vec:array[int]]]
#9:record[s:bstring]
#10:record[ts:time,ts2:time,ts3:time]
#11:record[s:string,srec:record[svec:array[string]]]
#12:record[s:bstring]
0:[[abc;xyz;]]
1:[[abc;xyz;]]
1:[[a\;b;xyz;]]
2:[[1;2;3;]]
3:[[1;2;3;]]
4:[[1.1.1.1;2.2.2.2;]]
5:[[1.1.1.1;2.2.2.2;]]
6:[[test;]]
7:[[[1;][2;]]]
8:[[[1;2;3;]]]
9:[begin\x01\x02\xffend;]
10:[1.001;1578411532;1578411533.01;]
11:[hello;[[world;worldz;1.1.1.1;]]]
9:[Buenos di\xcc\x81as sen\xcc\x83or;]
9:[Buenos d\xc3\xadas se\xc3\xb1or;]
12:[hello;]
0:[[a\;b;xyz;]]
`

func TestFilters(t *testing.T) {
	t.Parallel()

	ior := strings.NewReader(zngsrc)
	reader := detector.LookupReader("zng", ior, resolver.NewContext())

	nrecords := 17
	records := make([]*zng.Record, 0, nrecords)
	for {
		rec, err := reader.Read()
		require.NoError(t, err)
		if rec == nil {
			break
		}
		rec.Keep()
		records = append(records, rec)
	}

	assert.Equal(t, nrecords, len(records), fmt.Sprintf("zng parsed read %d records", nrecords))

	tests := []struct {
		filter         string
		record         *zng.Record
		expectedResult bool
	}{
		{"abc in stringset", records[0], true},
		{"xyz in stringset", records[0], true},
		{"ab in stringset", records[0], false},
		{"abcd in stringset", records[0], false},

		{"abc in stringvec", records[1], true},
		{"xyz in stringvec", records[1], true},
		{"ab in stringset", records[1], false},
		{"abcd in stringset", records[1], false},

		{"\"a;b\" in stringset", records[16], true},
		{"a in stringset", records[16], false},
		{"b in stringset", records[16], false},
		{"xyz in stringset", records[16], true},

		{"\"a;b\" in stringvec", records[2], true},
		{"a in stringvec", records[2], false},
		{"b in stringvec", records[2], false},
		{"xyz in stringvec", records[2], true},

		{"2 in intset", records[3], true},
		{"4 in intset", records[3], false},
		{"abc in intset", records[3], false},

		{"2 in intvec", records[4], true},
		{"4 in intvec", records[4], false},
		{"abc in intvec", records[4], false},

		{"1.1.1.1 in addrset", records[5], true},
		{"3.3.3.3 in addrset", records[5], false},
		{"1.1.1.1 in addrvec", records[6], true},
		{"3.3.3.3 in addrvec", records[6], false},
		{"len(addrvec) = 2", records[6], true},
		{"len(addrvec) = 3", records[6], false},
		{"len(addrvec) > 1", records[6], true},
		{"len(addrvec) >= 2", records[6], true},
		{"len(addrvec) < 5", records[6], true},
		{"len(addrvec) <= 2", records[6], true},

		{"nested.field = test", records[7], true},
		{"bogus.field = test", records[7], false},
		{"nested.bogus = test", records[7], false},
		{"* = test", records[7], false},
		{"** = test", records[7], true},

		{"nested[0].field = 1", records[8], true},
		{"nested[1].field = 2", records[8], true},
		{"nested[0].field = 2", records[8], false},
		{"nested[2].field = 2", records[8], false},
		{"nested.field = 2", records[8], false},

		{"1 in nested.vec", records[9], true},
		{"2 in nested.vec", records[9], true},
		{"4 in nested.vec", records[9], false},
		{"nested.vec[0] = 1", records[9], true},
		{"nested.vec[1] = 1", records[9], false},
		{"1 in nested", records[9], false},
		{"1", records[9], true},

		{"begin", records[10], true},
		{"s=begin", records[10], false},
		{"begin\\x01\\x02\\xffend", records[10], true},
		{"s=begin\\x01\\x02\\xffend", records[10], true},

		{"ts<2", records[11], true},
		{"ts2=1578411532", records[11], true},
		{"ts3=1578411533", records[11], false},
		{"T", records[11], false}, // 0x54 matches binary encoding of 1.001 but naked string search shouldn't

		{"ts=1.001", records[11], true},
		{"ts<1.002", records[11], true},
		{"ts<2.0", records[11], true},

		{"hello", records[12], true},
		{"worldz", records[12], true},
		{"1.1.1.1", records[12], true},

		// Test unicode string comparison.  The two records used in
		// these tests both have the string "Buenos días señor" but
		// one uses combining characters (e.g., plain n plus combining
		// tilde) and the other uses composed characters.  Test both
		// strings against queries written with both formats.
		{`s = "Buenos di\u{0301}as sen\u{0303}or"`, records[13], true},
		{`s = "Buenos d\u{ed}as se\u{f1}or"`, records[13], true},
		{`s = "Buenos di\u{0301}as sen\u{0303}or"`, records[14], true},
		{`s = "Buenos d\u{ed}as se\u{f1}or"`, records[14], true},

		// Test coercion between string/bstring
		{"s = hello", records[15], true},

		// Smoke test for globs...
		{"s = hell*", records[15], true},
		{"s = ell*", records[15], false},
	}

	for _, tt := range tests {
		t.Run(tt.filter, func(t *testing.T) {
			err := runTest(tt.filter, tt.record, tt.expectedResult)
			require.NoError(t, err)
		})
	}
}
