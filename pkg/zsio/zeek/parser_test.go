package zeek

import (
	"fmt"
	"net"
	"strings"
	"testing"

	"github.com/mccanne/zq/pkg/nano"
	"github.com/mccanne/zq/pkg/zson"
	"github.com/mccanne/zq/pkg/zson/resolver"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	separator    = "#separator \\x09"
	setSeparator = "#set_separator\t,"
	empty        = "#empty_field\t(empty)"
	unset        = "#unset_field\t-"
)

var (
	standardHeaders = []string{separator, setSeparator, empty, unset}
	fields          = []string{"str", "num", "addr", "ss", "sa"}
	types           = []string{"string", "int", "addr", "set[string]", "set[addr]"}
	values          = []string{"foo", "123", "10.5.5.5", "foo,bar,baz", "10.1.1.0,10.1.1.1,10.1.1.2"}
)

func makeHeader(name string, rest []string) string {
	return strings.Join(append([]string{name}, rest...), "\t")
}

// startTest() creates a new parser and sends all the provided
// directives, expecting them to all be parsed successfully.
// A parser object ready for further testing is returned.
func startTest(t *testing.T, headers []string) *parser {
	p := newParser(resolver.NewTable())
	for _, h  := range headers {
		require.NoError(t, p.parseDirective([]byte(h)))
	}

	return p
}

// startLegacyTest() creates a new parser and sends the standard
// zeek legacy directives.  If any of fields, types, path are provided,
// corresponding #files, #types, and #path directives are also sent.
// A parser object ready for further testing is returned.
func startLegacyTest(t *testing.T, fields, types []string, path string) *parser {
	headers := standardHeaders
	if len(path) > 0 {
		headers = append(headers, fmt.Sprintf("#path\t%s", path))
	}
	if len(fields) > 0 {
		headers = append(headers, makeHeader("#fields", fields))
	}
	if len(types) > 0 {
		headers = append(headers, makeHeader("#types", types))
	}

	return startTest(t, headers)
}

// sendLegacyValues() formats the array of values as a legacy zeek log line
// and parses it.
func sendLegacyValues(p *parser, vals []string) (*zson.Record, error) {
	return p.parseValue([]byte(strings.Join(vals, "\t")))
}

func assertError(t *testing.T, err error, pattern, what string) {
	assert.NotNilf(t, err, "Received error for %s", what)
	assert.Containsf(t, err.Error(), pattern, "error message for %s is as expected", what)
}

// Test things related to legacy zeek records that the parser should
// handle successfully.
func TestLegacyZeekValid(t *testing.T) {
	// Test standard headers but no #path and no timestamp in records
	parser := startLegacyTest(t, fields, types, "")
	record, err := sendLegacyValues(parser, values)
	require.NoError(t, err)

	assert.Equal(t, record.Ts, nano.MinTs, "Record has MinTs")
	assert.False(t, record.HasField("_path"), "Record does not have a _path field")
	// XXX check contents of other fields?

	// Test standard headers with a timestamp in records
	fieldsWithTs := append(fields, "ts")
	typesWithTs := append(types, "time")
	parser = startLegacyTest(t, fieldsWithTs, typesWithTs, "")

	timestamp := "1573588318384.000"
	valsWithTs := append(values, timestamp)
	record, err = sendLegacyValues(parser, valsWithTs)
	require.NoError(t, err)

	ts, err := nano.Parse([]byte(timestamp))
	require.NoError(t, err)
	assert.Equal(t, record.Ts, ts, "Timestamp is correct")

	// Test standard headers including a #path header
	parser = startLegacyTest(t, fieldsWithTs, typesWithTs, "testpath")
	record, err = sendLegacyValues(parser, valsWithTs)
	require.NoError(t, err)

	path, err := record.AccessString("_path")
	require.NoError(t, err)
	assert.Equal(t, path, "testpath", "Legacy _path field was set properly")

	// XXX test overriding separator, setSeparator
}

// Test things related to legacy zeek records that should cause the
// parser to generate errors.
func TestLegacyZeekInvalid(t *testing.T) {
	// Test that a non-standard value for empty_field is rejected
	parser := startTest(t, []string{separator, setSeparator})
	err := parser.parseDirective([]byte("#empty_field\tboo"))
	assertError(t, err, "encountered bad header field", "#empty_field header")

	// Test that a non-standard value for unset_field is rejected
	parser = startTest(t, []string{separator, setSeparator})
	err = parser.parseDirective([]byte("#unset_field\tboo"))
	assertError(t, err, "encountered bad header field", "#unset header")

	// Test that missing #fields/#values headers is an error
	parser = startTest(t, standardHeaders)
	_, err = sendLegacyValues(parser, values)
	assertError(t, err, "bad types/fields definition", "missing #fields/#types header")

	// Test that #fields header without #values is an error
	fh := makeHeader("#fields", fields)
	parser = startTest(t, append(standardHeaders, fh))
	_, err = sendLegacyValues(parser, values)
	assertError(t, err, "bad types/fields definition", "missing #types header")

	// Test that #types header without #fields is an error
	th := makeHeader("#types", types)
	parser = startTest(t, append(standardHeaders, th))
	_, err = sendLegacyValues(parser, values)
	assertError(t, err, "bad types/fields definition", "values without #fields")

	// Test that mismatched #fields/#types headers is an error
	/* XXX fixme
	parser = startTest(t, append(standardHeaders, fh))
	err = parser.parseDirective([]byte(makeHeader("#types", append(types, "int"))))
	assertError(t, err, "bad types/fields definition", "mismatched #fields/#types headers")
	*/

	// Test that the wrong number of values is an error
	parser = startTest(t, append(standardHeaders, fh, th))
	_, err = sendLegacyValues(parser, append(values, "extra"))
	assertError(t, err, "got 6 values, expected 5", "wrong number of values")

	// XXX check invalid types?
}

// Test things related to parsing zson
func TestZsonDescriptors(t *testing.T) {
	parser := startTest(t, nil)

	// Test a simple zson descriptor and corresponding value
	desc1 := "#1:record[s:string,n:int]"
	val1 := "1:[foo;5;]"
	require.NoError(t, parser.parseDirective([]byte(desc1)))
	record, err := parser.parseValue([]byte(val1))
	require.NoError(t, err)
	s, err := record.AccessString("s")
	require.NoError(t, err)
	assert.Equal(t, "foo", s, "Parsed string value properly")
	n, err := record.AccessInt("n")
	require.NoError(t, err)
	assert.Equal(t, 5, int(n), "Parsed int value properly")

	// Create a second descriptor of a different type
	desc2 := "#2:record[a:addr,p:port]"
	val2 := "2:[10.5.5.5;443;]"
	require.NoError(t, parser.parseDirective([]byte(desc2)))
	record, err = parser.parseValue([]byte(val2))
	require.NoError(t, err)
	a, err := record.AccessIP("a")
	require.NoError(t, err)
	expectAddr := net.ParseIP("10.5.5.5")
	assert.Equal(t, expectAddr, a, "Parsed addr value properly")
	n, err = record.AccessInt("p")
	assert.Equal(t, 443, int(n), "Parsed port value properly")

	// Can still use the first descriptor
	val3 := "1:[bar;100;]"
	record, err = parser.parseValue([]byte(val3))
	require.NoError(t, err)
	s, err = record.AccessString("s")
	require.NoError(t, err)
	assert.Equal(t, "bar", s, "Parsed another string properly")
	n, err = record.AccessInt("n")
	require.NoError(t, err)
	assert.Equal(t, 100, int(n), "Parsed another int properly")

	// XXX test other types, sets, vectors, etc.

	// Test that referencing an invalid descriptor is an error.
	_, err = parser.parseValue([]byte("100:[something;somethingelse;]"))
	assert.Error(t, err, "invalid descriptor", "invalid descriptor")

	// Test various malformed zson:
	zsons := []string{
		"1:string;123;",  // missing brackets
		"1:[string;123]", // missing semicolon
	}

	for _, z := range zsons {
		_, err = parser.parseValue([]byte(z))
		assert.Error(t, err, "zson parse error", "invalid zson")
	}

	// Can't use a descriptor of non-record type
	err = parser.parseDirective([]byte("#3:string"))
	assertError(t, err, "bad value", "descriptor with non-record type")

	// Descriptor with an invalid type is rejected
	err = parser.parseDirective([]byte("#4:notatype"))
	assertError(t, err, "unknown type", "descriptor with invalid type")

	// Trying to redefine a descriptor is an error
	err = parser.parseDirective([]byte("#1:record[n:int]"))
	assertError(t, err, "descriptor already exists", "redefining descriptor")
}

// XXX add test for mixing legacy and non-legacy
