package zsio

import (
	"net"
	"strings"
	"testing"

	"github.com/mccanne/zq/pkg/zson/resolver"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func assertError(t *testing.T, err error, pattern, what string) {
	assert.Error(t, err, "Received error for %s", what)
	assert.Containsf(t, err.Error(), pattern, "error message for %s is as expected", what)
}

// Test things related to parsing zson
func TestZsonDescriptors(t *testing.T) {
	// Step 1 - Test a simple zson descriptor and corresponding value
	zson := "#1:record[s:string,n:int]\n"
	zson += "1:[foo;5;];\n"
	// Step 2 - Create a second descriptor of a different type
	zson += "#2:record[a:addr,p:port]\n"
	zson += "2:[10.5.5.5;443;];\n"
	// Step 3 - can still use the first descriptor
	zson += "1:[bar;100;];\n"
	// Step 4 - Test that referencing an invalid descriptor is an error.
	zson += "100:[something;somethingelse;]\n"

	r := NewReader(strings.NewReader(zson), resolver.NewTable())

	// Check Step 1
	record, err := r.Read()
	require.NoError(t, err)
	s, err := record.AccessString("s")
	require.NoError(t, err)
	assert.Equal(t, "foo", s, "Parsed string value properly")
	n, err := record.AccessInt("n")
	require.NoError(t, err)
	assert.Equal(t, 5, int(n), "Parsed int value properly")

	// Check Step 2
	record, err = r.Read()
	require.NoError(t, err)
	a, err := record.AccessIP("a")
	require.NoError(t, err)
	expectAddr := net.ParseIP("10.5.5.5")
	assert.Equal(t, expectAddr, a, "Parsed addr value properly")
	n, err = record.AccessInt("p")
	assert.Equal(t, 443, int(n), "Parsed port value properly")

	// Check Step 3
	record, err = r.Read()
	require.NoError(t, err)
	s, err = record.AccessString("s")
	require.NoError(t, err)
	assert.Equal(t, "bar", s, "Parsed another string properly")
	n, err = record.AccessInt("n")
	require.NoError(t, err)
	assert.Equal(t, 100, int(n), "Parsed another int properly")

	// XXX test other types, sets, vectors, etc.

	// Check Step 4 - Test that referencing an invalid descriptor is an error.
	_, err = r.Read()
	assert.Error(t, err, "invalid descriptor", "invalid descriptor")

	// Test various malformed zson:
	def1 := "#1:record[s:string,n:int]\n"
	zsons := []string{
		def1 + "1:string;123;\n",   // missing brackets
		def1 + "1:[string;123];\n", // missing semicolon
	}

	for _, z := range zsons {
		r := NewReader(strings.NewReader(z), resolver.NewTable())
		_, err = r.Read()
		assert.Error(t, err, "zson parse error", "invalid zson")
	}
	// Can't use a descriptor of non-record type
	r = NewReader(strings.NewReader("#3:string\n"), resolver.NewTable())
	_, err = r.Read()
	assertError(t, err, "bad value", "descriptor with non-record type")

	// Descriptor with an invalid type is rejected
	r = NewReader(strings.NewReader("#4:notatype\n"), resolver.NewTable())
	_, err = r.Read()
	assertError(t, err, "unknown type", "descriptor with invalid type")

	// Trying to redefine a descriptor is an error XXX this should be ok
	d := "#1:record[n:int]\n"
	r = NewReader(strings.NewReader(d+d), resolver.NewTable())
	_, err = r.Read()
	assertError(t, err, "descriptor already exists", "redefining //descriptor")
}

// XXX add test for mixing legacy and non-legacy
