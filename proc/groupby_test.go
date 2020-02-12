package proc_test

import (
	"strings"
	"testing"

	"github.com/brimsec/zq/pkg/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Data sets for tests:
const in = `
#0:record[key1:string,key2:string,n:int32]
0:[a;x;1;]
0:[a;y;2;]
0:[b;z;1;]
`

const groupSingleOut = `
#0:record[key1:string,count:count]
0:[a;2;]
0:[b;1;]
`

const groupMultiOut = `
#0:record[key1:string,key2:string,count:count]
0:[a;x;1;]
0:[a;y;1;]
0:[b;z;1;]
`

const unsetIn = `
0:[-;-;3;]
0:[-;-;4;]
`

const groupSingleOut_unsetOut = `
#0:record[key1:string,count:count]
0:[-;2;]
0:[a;2;]
0:[b;1;]
`

const missingField = `
#1:record[key3:string,n:int32]
1:[a;1;]
1:[b;2;]
`

const differentTypeIn = `
#1:record[key1:ip,n:int32]
1:[10.0.0.1;1;]
1:[10.0.0.2;1;]
1:[10.0.0.1;1;]
`

const differentTypeOut = `
#1:record[key1:ip,count:count]
1:[10.0.0.1;2;]
1:[10.0.0.2;1;]
`

const reducersOut = `
#0:record[key1:string,first:int32,last:int32,sum:int64,avg:float64,min:int64,max:int64]
0:[a;1;2;3;1.5;1;2;]
0:[b;1;1;1;1;1;1;]
`

const arrayKeyIn = `
#0:record[arr:array[int32],val:int32]
0:[-;2;]
0:[[1;2;]2;]
0:[[1;2;]3;]
`

const arrayKeyOut = `
#0:record[arr:array[int32],count:count]
0:[-;1;]
0:[[1;2;]2;]
`

const nestedKeyIn = `
#0:record[rec:record[i:int32,s:string],val:int64]
0:[[1;bleah;]1;]
0:[[1;bleah;]2;]
0:[[2;bleah;]3;]
`

const nestedKeyOut = `
#0:record[rec:record[i:int32],count:count]
0:[[1;]2;]
0:[[2;]1;]
`

const nullIn = `
#0:record[key:string,val:int64]
0:[key1;5;]
0:[key2;-;]
`

const nullOut = `
#0:record[key:string,sum:int64]
0:[key1;5;]
0:[key2;-;]
`

const notPresentIn = `
#0:record[key:string]
0:[key1;]
`

const notPresentOut = `
#0:record[key:string,max:null,sum:null]
0:[key1;-;-;]
`

const mixedIn = `
#0:record[key:string,f:int32]
0:[k;5;]
#1:record[key:string,f:string]
1:[k;bleah;]
`

const mixedOut = `
#0:record[key:string,first:int32,last:string]
0:[k;5;bleah;]
`

const aliasIn = `
#ipaddr=ip
#0:record[host:ipaddr]
0:[127.0.0.1;]
#1:record[host:ip]
1:[127.0.0.1;]
`

const aliasOut = `
#ipaddr=ip
#0:record[host:ipaddr,count:count]
0:[127.0.0.1;1;]
#1:record[host:ip,count:count]
1:[127.0.0.1;1;]
`

//XXX this should go in a shared package
type suite []test.Internal

func (s suite) runSystem(t *testing.T) {
	t.Parallel()
	for _, d := range s {
		t.Run(d.Name, func(t *testing.T) {
			results, err := d.Run()
			require.NoError(t, err)
			assert.Exactly(t, d.Expected, results, "Wrong query results")
		})
	}
}

func (s *suite) add(t test.Internal) {
	*s = append(*s, t)
}

func New(name, input, output, cmd string) test.Internal {
	output = strings.ReplaceAll(output, "\n\n", "\n")
	return test.Internal{
		Name:         name,
		Query:        "* | " + cmd,
		Input:        input,
		OutputFormat: "zng",
		Expected:     test.Trim(output),
	}
}

func tests() suite {
	s := suite{}

	// Test a simple groupby
	s.add(New("simple", in, groupSingleOut, "count() by key1"))

	// Test that unset key values work correctly
	s.add(New("unset-keys", in+unsetIn, groupSingleOut_unsetOut, "count() by key1"))

	// Test grouping by multiple fields
	s.add(New("multiple-fields", in, groupMultiOut, "count() by key1,key2"))

	// Test that records missing groupby fields are ignored
	s.add(New("missing-fields", in+missingField, groupSingleOut, "count() by key1"))

	// Test that input with different key types works correctly
	s.add(New("different-key-types", in+differentTypeIn, groupSingleOut+differentTypeOut, "count() by key1"))

	// Test various reducers
	s.add(New("reducers", in, reducersOut, "first(n), last(n), sum(n), avg(n), min(n), max(n) by key1"))

	// Check out of bounds array indexes
	s.add(New("array-out-of-bounds", arrayKeyIn, arrayKeyOut, "count() by arr"))

	// Check groupby key inside a record
	s.add(New("key-in-record", nestedKeyIn, nestedKeyOut, "count() by rec.i"))

	// Test reducers with null inputs
	s.add(New("null-inputs", nullIn, nullOut, "sum(val) by key"))

	// Test reducers with missing operands
	s.add(New("not-present", notPresentIn, notPresentOut, "max(val), sum(val) by key"))

	// Test reducers with mixed-type inputs
	s.add(New("mixed-inputs", mixedIn, mixedOut, "first(f), last(f) by key"))

	s.add(New("aliases", aliasIn, aliasOut, "count() by host"))
	// XXX add coverage of time batching (every ..)

	return s
}

func TestGroupbySystem(t *testing.T) {
	tests().runSystem(t)
}

/* not yet
func TestGroupbyUnit(t *testing.T) {
	tests().runUnit(t)
}
*/
