package proc_test

import (
	"strings"
	"testing"

	"github.com/mccanne/zq/pkg/nano"
	"github.com/mccanne/zq/pkg/zsio"
	"github.com/mccanne/zq/pkg/zson"
	"github.com/mccanne/zq/pkg/zson/resolver"
	"github.com/mccanne/zq/proc"
	"github.com/stretchr/testify/require"
)

func parse(resolver *resolver.Table, src string) (*zson.Array, error) {
	reader := zsio.LookupReader("zson", strings.NewReader(src), resolver)
	records := make([]*zson.Record, 0)
	for {
		rec, err := reader.Read()
		if err != nil {
			return nil, err
		}
		if rec == nil {
			break
		}
		records = append(records, rec)
	}

	return zson.NewArray(records, nano.MaxSpan), nil
}

// testOne() runs one test of a sort proc with limit, fields, and dir
// as the parameters to sort (XXX should just use the parser for this).
// Parses zsonin, runs the resulting records through sort, then asserts
// that the output matches zsonout.
func testOne(t *testing.T, zsonin, zsonout string, cmd string) {
	resolver := resolver.NewTable()
	recsin, err := parse(resolver, zsonin)
	require.NoError(t, err)
	recsout, err := parse(resolver, zsonout)
	require.NoError(t, err)

	test, err := proc.NewProcTestFromSource(cmd, resolver, []zson.Batch{recsin})
	require.NoError(t, err)

	result, err := test.Pull()
	require.NoError(t, err)
	require.NoError(t, test.ExpectEOS())
	require.NoError(t, test.Finish())

	require.Equal(t, recsout.Length(), result.Length())
	for i := 0; i < result.Length(); i++ {
		r1 := recsout.Index(i)
		r2 := result.Index(i)
		// XXX could print something a lot pretter if/when this fails.
		require.Equalf(t, r2.Raw, r1.Raw, "Expected record %d to match", i)
	}
}

// Data sets for tests:
const unsortedInts = `
#0:record[foo:int]
0:[100;]
0:[2;]
0:[9100;]
`

const ascendingInts = `
#0:record[foo:int]
0:[2;]
0:[100;]
0:[9100;]
`

const descendingInts = `
#0:record[foo:int]
0:[9100;]
0:[100;]
0:[2;]
`

const unsortedStrings = `
#1:record[foo:string]
1:[zzz;]
1:[hello;]
1:[abc;]
1:[abcd;]
`
const sortedStrings = `
#1:record[foo:string]
1:[abc;]
1:[abcd;]
1:[hello;]
1:[zzz;]
`

// A point that can be included with unsortedInts
const unsetInt = `
0:[-;]
`

// Some records that don't include the field "foo".  These are combined
// with sets that include foo to test mixed records.
const nofoo = `
#2:record[notfoo:int]
2:[1;]
2:[2;]
2:[3;]
`

// Records for testing sorting on multiple fields.
const multiIn = `
#3:record[foo:int,bar:int]
3:[5;10;]
3:[10;10;]
3:[10;5;]
3:[5;5;]
`

const foobarOut = `
#3:record[foo:int,bar:int]
3:[5;5;]
3:[5;10;]
3:[10;5;]
3:[10;10;]
`

const barfooOut = `
#3:record[foo:int,bar:int]
3:[5;5;]
3:[10;5;]
3:[5;10;]
3:[10;10;]
`

// Test cases for sort without a field list, in which case sort chooses
// the field to sort on.
// First case: prefer an int-valued field (n in this case)
const chooseIn1 = `
#4:record[s:string,n:int,ts:time]
4:[a;300;1574610096.000000;]
4:[b;100;1574610095.000000;]
4:[c;200;1574610094.000000;]
`

const chooseOut1 = `
#4:record[s:string,n:int,ts:time]
4:[b;100;1574610095.000000;]
4:[c;200;1574610094.000000;]
4:[a;300;1574610096.000000;]
`

// Second case: prefer a non-time-valued field.
const chooseIn2 = `
#4:record[s:string,ts:time]
4:[c;1574610094.000000;]
4:[a;1574610096.000000;]
4:[b;1574610095.000000;]
`

const chooseOut2 = `
#4:record[s:string,ts:time]
4:[a;1574610096.000000;]
4:[b;1574610095.000000;]
4:[c;1574610094.000000;]
`

// This case: no numeric fields, just take the very first one.
const chooseIn3 = `
#4:record[s:string,s2:string]
4:[a;c;]
4:[c;a;]
4:[b;b;]
`
const chooseOut3 = `
#4:record[s:string,s2:string]
4:[a;c;]
4:[b;b;]
4:[c;a;]
`

func TestSort(t *testing.T) {
	// Test simple sorting of integers.
	testOne(t, unsortedInts, ascendingInts, "sort foo")

	// Test sorting ints in reverse.
	testOne(t, unsortedInts, descendingInts, "sort -r foo")

	// Test sorting strings.
	testOne(t, unsortedStrings, sortedStrings, "sort foo")

	// Test that unset values are sorted to the end
	testOne(t, unsortedInts+unsetInt, ascendingInts+unsetInt, "sort foo")

	// Test sorting records that don't all have the requested field.
	// XXX sort.Stable() is apparently re-ordering the nofoo records?
	// const missingFields = nofoo + unsortedStrings
	// const missingSorted = sortedStrings + nofoo
	// testOne(t, missingFields, missingSorted, "sort foo")

	// Test sorting records with different types.
	const mixedTypesIn = unsortedStrings + unsortedInts
	const mixedTypesOut = ascendingInts + sortedStrings
	testOne(t, mixedTypesIn, mixedTypesOut, "sort foo")

	// Test sorting on multiple fields.
	testOne(t, multiIn, foobarOut, "sort foo, bar")
	testOne(t, multiIn, barfooOut, "sort bar, foo")

	// Test that choosing a field when none is provided works.
	testOne(t, chooseIn1, chooseOut1, "sort")
	testOne(t, chooseIn2, chooseOut2, "sort")
	testOne(t, chooseIn3, chooseOut3, "sort")
}
