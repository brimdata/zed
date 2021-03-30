package zio_test

import (
	"net"
	"strings"
	"testing"

	"github.com/brimdata/zed/zio/tzngio"
	"github.com/brimdata/zed/zng/resolver"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func assertError(t *testing.T, err error, pattern, what string) {
	assert.Error(t, err, "Received error for %s", what)
	if err != nil {
		assert.Containsf(t, err.Error(), pattern, "error message for %s is as expected", what)
	}
}

// Test things related to parsing tzng
func TestTzngDescriptors(t *testing.T) {
	// Step 1 - Test a simple zng descriptor and corresponding value
	src := "#port=uint16\n"
	src += "#1:record[s:string,n:int32]\n"
	src += "1:[foo;5;]\n"
	// Step 2 - Create a second descriptor of a different type
	src += "#2:record[a:ip,p:port]\n"
	src += "2:[10.5.5.5;443;]\n"
	// Step 3 - can still use the first descriptor
	src += "1:[bar;100;]\n"
	// Step 4 - Test that referencing an invalid descriptor is an error.
	src += "100:[something;somethingelse;]\n"

	r := tzngio.NewReader(strings.NewReader(src), resolver.NewContext())

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
	expectAddr := net.ParseIP("10.5.5.5").To4()
	assert.Equal(t, expectAddr, a, "Parsed addr value properly")
	n, err = record.AccessInt("p")
	require.NoError(t, err)
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

	// XXX test other types, sets, arrays, etc.

	// Check Step 4 - Test that referencing an invalid descriptor is an error.
	_, err = r.Read()
	assert.Error(t, err, "invalid descriptor", "invalid descriptor")

	// Test various malformed zng:
	def1 := "#1:record[s:string,n:int32]\n"
	zngs := []string{
		def1 + "1:string;123;\n",  // missing brackets
		def1 + "1:[string;123]\n", // missing semicolon
	}

	for _, z := range zngs {
		r := tzngio.NewReader(strings.NewReader(z), resolver.NewContext())
		_, err = r.Read()
		assert.Error(t, err, "tzng parse error", "invalid tzng")
	}

	// Descriptor with an invalid type is rejected
	r = tzngio.NewReader(strings.NewReader("#4:notatype\n"), resolver.NewContext())
	_, err = r.Read()
	assertError(t, err, "unknown type", "descriptor with invalid type")
}

// XXX add test for mixing legacy and non-legacy
