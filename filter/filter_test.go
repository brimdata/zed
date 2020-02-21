package filter_test

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/brimsec/zq/ast"
	"github.com/brimsec/zq/filter"
	"github.com/brimsec/zq/zio/detector"
	"github.com/brimsec/zq/zng"
	"github.com/brimsec/zq/zng/resolver"
	"github.com/brimsec/zq/zql"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func compileFilter(filt string) (filter.Filter, error) {
	// Parse the filter.  Any filter is a valid full zql query,
	// it should parse to an AST with a top-level FilterProc node.
	parsed, err := zql.Parse("", []byte(filt))
	if err != nil {
		return nil, err
	}

	filtProc, ok := parsed.(*ast.FilterProc)
	if !ok {
		return nil, errors.New("expected FilterProc")
	}

	// Compile the filter...
	return filter.Compile(filtProc.Filter)
}

// Execute one test of a filter by compiling the given filter and
// executing it against the given Record.  Returns an error if the filter
// result does not match expectedResult (or for any other error such as
// failure to parse or compile the filter)
func runTest(filt string, record *zng.Record, expectedResult bool) error {
	f, err := compileFilter(filt)
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

func parseOneRecord(zngsrc string) (*zng.Record, error) {
	ior := strings.NewReader(zngsrc)
	reader, err := detector.LookupReader("zng", ior, resolver.NewContext())
	if err != nil {
		return nil, err
	}

	rec, err := reader.Read()
	if err != nil {
		return nil, err
	}
	if rec == nil {
		return nil, errors.New("expected to read one record")
	}
	rec.Keep()

	rec2, err := reader.Read()
	if err != nil {
		return nil, err
	}
	if rec2 != nil {
		return nil, errors.New("got more than one record")
	}
	return rec, nil
}

type testcase struct {
	filter         string
	expectedResult bool
}

func runCases(t *testing.T, record *zng.Record, cases []testcase) {
	for _, tt := range cases {
		t.Run(tt.filter, func(t *testing.T) {
			err := runTest(tt.filter, record, tt.expectedResult)
			require.NoError(t, err)
		})
	}
}

func TestFilters(t *testing.T) {
	t.Parallel()

	// Test set membership with "in"
	record, err := parseOneRecord(`
#0:record[stringset:set[bstring]]
0:[[abc;xyz;]]`)
	require.NoError(t, err)
	runCases(t, record, []testcase{
		{"abc in stringset", true},
		{"xyz in stringset", true},
		{"ab in stringset", false},
		{"abcd in stringset", false},
	})

	// Test escaped bstrings inside a set
	record, err = parseOneRecord(`#0:record[stringset:set[bstring]]
0:[[a\x3bb;xyz;]]`)
	require.NoError(t, err)
	runCases(t, record, []testcase{
		{"\"a;b\" in stringset", true},
		{"a in stringset", false},
		{"b in stringset", false},
		{"xyz in stringset", true},
	})

	// Test array membership with "in"
	record, err = parseOneRecord(`
#0:record[stringvec:array[bstring]]
0:[[abc;xyz;]]`)
	require.NoError(t, err)
	runCases(t, record, []testcase{
		{"abc in stringvec", true},
		{"xyz in stringvec", true},
		{"ab in stringvec", false},
		{"abcd in stringvec", false},
	})

	// Test escaped bstrings inside an array
	record, err = parseOneRecord(`
#0:record[stringvec:array[bstring]]
0:[[a\x3bb;xyz;]]`)
	require.NoError(t, err)
	runCases(t, record, []testcase{
		{"\"a;b\" in stringvec", true},
		{"a in stringvec", false},
		{"b in stringvec", false},
		{"xyz in stringvec", true},
	})

	// Test membership in set of integers
	record, err = parseOneRecord(`
#0:record[intset:set[int32]]
0:[[1;2;3;]]`)
	require.NoError(t, err)
	runCases(t, record, []testcase{
		{"2 in intset", true},
		{"4 in intset", false},
		{"abc in intset", false},
	})

	// Test membership in array of integers
	record, err = parseOneRecord(`
#0:record[intvec:array[int32]]
0:[[1;2;3;]]`)
	require.NoError(t, err)
	runCases(t, record, []testcase{
		{"2 in intvec", true},
		{"4 in intvec", false},
		{"abc in intvec", false},
	})

	// Test membership in set of ip addresses
	record, err = parseOneRecord(`
#0:record[addrset:set[ip]]
0:[[1.1.1.1;2.2.2.2;]]`)
	require.NoError(t, err)
	runCases(t, record, []testcase{
		{"1.1.1.1 in addrset", true},
		{"3.3.3.3 in addrset", false},
	})

	// Test membership and len() on array of ip addresses
	record, err = parseOneRecord(`
#0:record[addrvec:array[ip]]
0:[[1.1.1.1;2.2.2.2;]]`)
	require.NoError(t, err)
	runCases(t, record, []testcase{
		{"1.1.1.1 in addrvec", true},
		{"3.3.3.3 in addrvec", false},
		{"len(addrvec) = 2", true},
		{"len(addrvec) = 3", false},
		{"len(addrvec) > 1", true},
		{"len(addrvec) >= 2", true},
		{"len(addrvec) < 5", true},
		{"len(addrvec) <= 2", true},
	})

	// Test comparing fields in nested records
	record, err = parseOneRecord(`
#0:record[nested:record[field:string]]
0:[[test;]]`)
	require.NoError(t, err)
	runCases(t, record, []testcase{
		{"nested.field = test", true},
		{"bogus.field = test", false},
		{"nested.bogus = test", false},
		{"* = test", false},
		{"** = test", true},
	})

	// Test array of records
	record, err = parseOneRecord(`
#0:record[nested:array[record[field:int32]]]
0:[[[1;][2;]]]`)
	require.NoError(t, err)
	runCases(t, record, []testcase{
		{"nested[0].field = 1", true},
		{"nested[1].field = 2", true},
		{"nested[0].field = 2", false},
		{"nested[2].field = 2", false},
		{"nested.field = 2", false},
	})

	// Test array inside a record
	record, err = parseOneRecord(`
#0:record[nested:record[vec:array[int32]]]
0:[[[1;2;3;]]]`)
	require.NoError(t, err)
	runCases(t, record, []testcase{
		{"1 in nested.vec", true},
		{"2 in nested.vec", true},
		{"4 in nested.vec", false},
		{"nested.vec[0] = 1", true},
		{"nested.vec[1] = 1", false},
		{"1 in nested", false},
		{"1", true},
	})

	// Test escaped chars in a bstring
	record, err = parseOneRecord(`
#0:record[s:bstring]
0:[begin\x01\x02\xffend;]`)
	require.NoError(t, err)
	runCases(t, record, []testcase{
		{"begin", true},
		{"s=begin", false},
		{"begin\\x01\\x02\\xffend", true},
		{"s=begin\\x01\\x02\\xffend", true},
		{"s=*\\x01\\x02*", true},
	})

	// Test unicode string comparison.  The following two records
	// both have the string "Buenos días señor" but one uses
	// combining characters (e.g., plain n plus combining
	// tilde) and the other uses composed characters.  Test both
	// strings against queries written with both formats.
	record, err = parseOneRecord(`
#0:record[s:bstring]
0:[Buenos di\xcc\x81as sen\xcc\x83or;]`)
	require.NoError(t, err)
	runCases(t, record, []testcase{
		{`s = "Buenos di\u{0301}as sen\u{0303}or"`, true},
		{`s = "Buenos d\u{ed}as se\u{f1}or"`, true},
	})
	record, err = parseOneRecord(`
#0:record[s:bstring]
0:[Buenos d\xc3\xadas se\xc3\xb1or;]`)
	require.NoError(t, err)
	runCases(t, record, []testcase{
		{`s = "Buenos di\u{0301}as sen\u{0303}or"`, true},
		{`s = "Buenos d\u{ed}as se\u{f1}or"`, true},
	})

	// Test searching inside containers
	record, err = parseOneRecord(`
#0:record[s:string,srec:record[svec:array[string]]]
0:[hello;[[world;worldz;1.1.1.1;]]]`)
	require.NoError(t, err)
	runCases(t, record, []testcase{
		{"hello", true},
		{"worldz", true},
		{"1.1.1.1", true},
	})

	// Test time coercion
	record, err = parseOneRecord(`
#0:record[ts:time,ts2:time,ts3:time]
0:[1.001;1578411532;1578411533.01;]`)
	require.NoError(t, err)
	runCases(t, record, []testcase{
		{"ts<2", true},
		{"ts=1.001", true},
		{"ts<1.002", true},
		{"ts<2.0", true},
		{"ts2=1578411532", true},
		{"ts3=1578411533", false},
		// The ASCII value of 'T' (0x54) is present inside the binary
		// encoding of 1.001.  But naked string search should not match.
		{"T", false},
	})

	// Test integer conditions.  These are really testing 2 things:
	// 1. that the full range of values are correctly parsed
	// 2. that coercion to int64 works properly (in all the filters
	//    with integers on the RHS)
	// 3. that coercion to float64 works properly (in the filters
	//    with floats on the RHS)
	record, err = parseOneRecord(`
#0:record[b:byte,i16:int16,u16:uint16,i32:int32,u32:uint32,i64:int64,u64:uint64]
0:[0;-32768;0;-2147483648;0;-9223372036854775808;0;]
`)
	require.NoError(t, err)
	runCases(t, record, []testcase{
		{"b > -1", true},
		{"b = 0", true},
		{"b < 1", true},
		{"b > 1", false},
		{"b = 0.0", true},
		{"b < 0.5", true},

		{"i16 = -32768", true},
		{"i16 < 0", true},
		{"i16 > 0", false},
		{"i16 = -32768.0", true},
		{"i16 < 0.0", true},

		{"u16 > -1", true},
		{"u16 = 0", true},
		{"u16 < 1", true},
		{"u16 > 1", false},
		{"u16 = 0.0", true},
		{"u16 < 0.5", true},

		{"i32 = -2147483648", true},
		{"i32 < 0", true},
		{"i32 > 0", false},
		{"i32 = -2147483648.0", true},
		{"i32 < 0.5", true},

		{"u32 > -1", true},
		{"u32 = 0", true},
		{"u32 < 1", true},
		{"u32 > 1", false},
		{"u32 = 0.0", true},
		{"u32 < 0.5", true},

		{"i64 = -9223372036854775808", true},
		{"i64 < 0", true},
		{"i64 > 0", false},
		{"i64 < 0.0", true},
		// MinInt64 can't be represented precisely as a float64

		{"u64 > -1", true},
		{"u64 = 0", true},
		{"u64 < 1", true},
		{"u64 > 1", false},
		{"u64 = 0.0", true},
		{"u64 < 0.5", true},

		// Can't coerce an integer to a port
		{"u16 = :0", false},
		{"u16 != :0", false},
	})

	record, err = parseOneRecord(`
#0:record[b:byte,i16:int16,u16:uint16,i32:int32,u32:uint32,i64:int64,u64:uint64]
0:[255;32767;65535;2147483647;4294967295;9223372036854775807;18446744073709551615;]
`)
	require.NoError(t, err)
	runCases(t, record, []testcase{
		{"b = 255", true},
		{"i16 = 32767", true},
		{"u16 = 65535", true},
		{"i32 = 2147483647", true},
		{"u32 = 4294967295", true},
		{"i64 = 9223372036854775807", true},
		// can't represent large unsigned 64 bit values in zql...
		// {"u64 = 18446744073709551615", true},
	})

	// Test comparisons with field of type port (can compare with
	// a port literal or an integer literal)
	record, err = parseOneRecord(`
#0:record[p:port]
0:[443;]
`)
	require.NoError(t, err)
	runCases(t, record, []testcase{
		{"p = :443", true},
		{"p = 443", true},
		{"p = 80", false},
		{"p = :80", false},
	})

	// Test coercion from string to bstring
	record, err = parseOneRecord(`
#0:record[s:bstring]
0:[hello;]
`)
	require.NoError(t, err)
	runCases(t, record, []testcase{
		{"s = hello", true},

		// Also smoke test that globs work...
		{"s = hell*", true},
		{"s = ell*", false},
	})

	// Test ip comparisons
	record, err = parseOneRecord(`
#0:record[a:ip]
0:[192.168.1.50;]
`)
	require.NoError(t, err)
	runCases(t, record, []testcase{
		{"a = 192.168.1.50", true},
		{"a = 50.1.168.192", false},
		{"a != 50.1.168.192", true},
		{"a = 192.168.0.0/16", true},
		{"a != 192.168.0.0/16", false},
		{"a = 10.0.0.0/16", false},
		{"a != 10.0.0.0/16", true},
	})

	// Test comparisons with an aliased type
	record, err = parseOneRecord(`
#myint=int32
#0:record[i:myint]
0:[100;]`)
	require.NoError(t, err)
	runCases(t, record, []testcase{
		{"i = 100", true},
		{"i > 0", true},
		{"i < 50", false},
	})
}

func TestBadFilter(t *testing.T) {
	filter := `s = \xa8*`
	_, err := compileFilter(filter)
	assert.Error(t, err, "Received error for bad glob")
	assert.Contains(t, err.Error(), "invalid UTF-8", "Received good error message for invalid UTF-8 in a regexp")
}
