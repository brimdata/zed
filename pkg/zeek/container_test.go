package zeek_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/mccanne/zq/filter"
	"github.com/mccanne/zq/pkg/zeek"
	"github.com/mccanne/zq/pkg/zson"
	"github.com/looky-cloud/lookytalk/ast"
	"github.com/stretchr/testify/require"
)

// Execute one test of a filter on a container type (ie set or vector).
// Does the following:
// 1. Compiles a filter that corresponds to a "value in field" query for
//    the given value.
// 2. Executes the filter against a set/vector with the given type and value.
// 3. Checks the filter result against the expected result.
//
// Returns an error if #3 does not match (or for any other error), or
// nil if the result matches.
func runTest(valType string, valRaw string, containerType string, containerRaw string, expectedResult bool) error {
	// Build and compile the filter.
	expr := &ast.CompareField{
		Comparator: "in",
		Field:      &ast.FieldRead{Field: "f"},
		Value:      ast.TypedValue{Type: valType, Value: valRaw},
	}

	filt, err := filter.Compile(expr)
	if err != nil {
		return err
	}

	// Mock up a tuple with a single column that holds the set.
	containerTyp, err := zeek.LookupType(containerType)
	if err != nil {
		return err
	}
	columns := []zeek.Column{{"f", containerTyp}}
	d := zson.NewDescriptor(zeek.LookupTypeRecord(columns))
	r, err := zson.NewRecordZeekStrings(d, containerRaw)
	if err != nil {
		return err
	}

	// Apply the filter.
	result := filt(r)

	if result == expectedResult {
		return nil
	}
	if expectedResult {
		return fmt.Errorf("Should have found %s in %s", valRaw, containerRaw)
	} else {
		return fmt.Errorf("Should not have found %s in %s", valRaw, containerRaw)
	}
}

func containerLen(val string) int {
	return len(strings.Split(val, ","))
}

func recordType(typ string, n int) string {
	s := "record["
	comma := ""
	for k := 0; k < n; k++ {
		s += fmt.Sprintf("%sfld%d:%s", comma, k, typ)
		comma = ","
	}
	return s + "]"
}

func TestContainers(t *testing.T) {
	t.Parallel()

	tests := []struct {
		valType        string
		valRaw         string
		elementType    string
		containerRaw   string
		containerLen   int
		expectedResult bool
	}{
		{"string", "abc", "string", "abc,xyz", 2, true},
		{"string", "xyz", "string", "abc,xyz", 2, true},
		{"string", "abcd", "string", "abc,xyz", 2, false},
		{"string", "ab", "string", "abc,xyz", 2, false},
		{"string", "a,b", "string", `a\x2cb,c`, 2, true},
		{"string", "c", "string", `a\x2cb,c`, 2, true},
		{"string", "b", "string", `a\x2cb,c`, 2, false},
		{"string", "abc", "int", "1,2,3", 3, false},
		{"int", "2", "int", "1,2,3", 3, true},
		{"int", "4", "int", "1,2,3", 3, false},
		{"addr", "1.1.1.1", "addr", "1.1.1.1,2.2.2.2", 2, true},
		{"addr", "3.3.3.3", "addr", "1.1.1.1,2.2.2.2", 2, false},
	}

	for _, tt := range tests {
		// Run each test case against both set and vector
		containerType := fmt.Sprintf("set[%s]", tt.elementType)
		err := runTest(tt.valType, tt.valRaw, containerType, tt.containerRaw, tt.expectedResult)
		require.NoError(t, err)

		containerType = fmt.Sprintf("vector[%s]", tt.elementType)
		err = runTest(tt.valType, tt.valRaw, containerType, tt.containerRaw, tt.expectedResult)
		require.NoError(t, err)

		//XXX for records, just make sure they parse for now.
		// there is no ast/filter support for them yet
		containerType = recordType(tt.elementType, tt.containerLen)
		parsedType, err := zeek.LookupType(containerType)
		require.NoError(t, err)
		require.Exactly(t, containerType, parsedType.String())
	}
}
