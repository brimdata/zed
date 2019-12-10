package proc_test

import (
	"strings"
	"testing"

	"github.com/mccanne/zq/pkg/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Data sets for tests:
const in = `
#0:record[key1:string,key2:string,n:int]
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

const unsetOut = `
0:[-;2;]
`

const groupSingleOut_unsetOut = `
#0:record[key1:string,count:count]
0:[-;2;]
0:[a;2;]
0:[b;1;]
`

const missingField = `
#1:record[key3:string,n:int]
1:[a;1;]
1:[b;2;]
`

const differentTypeIn = `
#1:record[key1:addr,n:int]
1:[10.0.0.1;1;]
1:[10.0.0.2;1;]
1:[10.0.0.1;1;]
`

const differentTypeOut = `
#1:record[key1:addr,count:count]
1:[10.0.0.1;2;]
1:[10.0.0.2;1;]
`

const reducersOut = `
#0:record[key1:string,first:int,last:int,sum:int,avg:double,min:int,max:int]
0:[a;1;2;3;1.5;1;2;]
0:[b;1;1;1;1;1;1;]
`

const arrayKeyIn = `
#0:record[vec:vector[int],val:int]
0:[-;2;]
0:[[1;2;]2;]
0:[[1;2;]3;]
`

const arrayKeyOut = `
#0:record[vec:vector[int],count:count]
0:[-;1;]
0:[[1;2;]2;]
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
		Name:     name,
		Query:    "* | " + cmd,
		Input:    input,
		Format:   "zson",
		Expected: test.Trim(output),
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
	s.add(New("array-out-of-bounds", arrayKeyIn, arrayKeyOut, "count() by vec"))

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
