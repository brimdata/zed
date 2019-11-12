package zeek

import (
	"fmt"
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
	assert.NotNil(t, err, "#empty_field header caused an error")
	assert.Contains(t, err.Error(), "encountered bad header field", "error emssage for bad #empty_field header is as expected")

	// Test that a non-standard value for unset_field is rejected
	parser = startTest(t, []string{separator, setSeparator})
	err = parser.parseDirective([]byte("#unset_field\tboo"))
	assert.NotNil(t, err, "#unset_field header caused an error")
	assert.Contains(t, err.Error(), "encountered bad header field", "error emssage for bad #unset_field header is as expected")

	// Test that missing #fields/#values headers is an error
	parser = startTest(t, standardHeaders)
	_, err = sendLegacyValues(parser, values)
	assert.NotNil(t, err, "values without #fields/#types caused an error")
	assert.Contains(t, err.Error(), "bad types/fields definition", "error emssage for missing #fields/#types header is as expected")

	// Test that #fields header without #values is an error
	fh := makeHeader("#fields", fields)
	parser = startTest(t, append(standardHeaders, fh))
	_, err = sendLegacyValues(parser, values)
	assert.NotNil(t, err, "values without #types caused an error")
	assert.Contains(t, err.Error(), "bad types/fields definition", "error emssage for missing #types header is as expected")

	// Test that #types header without #fields is an error
	th := makeHeader("#types", types)
	parser = startTest(t, append(standardHeaders, th))
	_, err = sendLegacyValues(parser, values)
	assert.NotNil(t, err, "values without #fields caused an error")
	assert.Contains(t, err.Error(), "bad types/fields definition", "error emssage for missing #fields header is as expected")

	// Test that mismatched #fields/#types headers is an error
	/* XXX fixme
	parser = startTest(t, append(standardHeaders, fh))
	err = parser.parseDirective([]byte(makeHeader("#types", append(types, "int"))))
	assert.NotNil(t, err, "mismatched #fields/#types caused an error")
	assert.Contains(t, err.Error(), "bad types/fields definition", "error emssage for mismatched #fields/#types header is as expected")
	*/

	// Test that the wrong number of values is an error
	parser = startTest(t, append(standardHeaders, fh, th))
	_, err = sendLegacyValues(parser, append(values, "extra"))
	assert.NotNil(t, err, "mismatched #fields/#types caused an error")
	assert.Contains(t, err.Error(), "got 6 values, expected 5", "error message for wrong numbers of values is as expected")

	// XXX check invalid types?
}

// XXX Add non-legacy zson test cases

