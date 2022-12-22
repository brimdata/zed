package zeekio

import (
	"fmt"
	"strings"
	"testing"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/pkg/nano"
	"github.com/brimdata/zed/zio/zngio"
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
func startTest(t *testing.T, headers []string) *Parser {
	p := NewParser(zed.NewContext())
	for _, h := range headers {
		require.NoError(t, p.ParseDirective([]byte(h)))
	}

	return p
}

// startLegacyTest() creates a new parser and sends the standard
// zeek legacy directives.  If any of fields, types, path are provided,
// corresponding #files, #types, and #path directives are also sent.
// A parser object ready for further testing is returned.
func startLegacyTest(t *testing.T, fields, types []string, path string) *Parser {
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
func sendLegacyValues(p *Parser, vals []string) (*zed.Value, error) {
	return p.ParseValue([]byte(strings.Join(vals, "\t")))
}

func assertError(t *testing.T, err error, pattern, what string) {
	assert.NotNilf(t, err, "Received error for %s", what)
	assert.Containsf(t, err.Error(), pattern, "error message for %s is as expected", what)
}

// Test things related to legacy zeek records that the parser should
// handle successfully.
func TestLegacyZeekValid(t *testing.T) {
	// Test standard headers but no timestamp in records
	parser := startLegacyTest(t, fields, types, "")
	record, err := sendLegacyValues(parser, values)
	require.NoError(t, err)

	assert.Equal(t, record.Deref("ts").MissingAsNull(), zed.Null)
	assert.Equal(t, record.Deref("ts").MissingAsNull().AsTime(), nano.Ts(0))
	// XXX check contents of other fields?

	// Test standard headers with a timestamp in records
	fieldsWithTs := append(fields, "ts")
	typesWithTs := append(types, "time")
	parser = startLegacyTest(t, fieldsWithTs, typesWithTs, "")

	timestamp := "1573588318384.000"
	valsWithTs := append(values, timestamp)
	record, err = sendLegacyValues(parser, valsWithTs)
	require.NoError(t, err)

	expectedTs, err := parseTime([]byte(timestamp))
	require.NoError(t, err)
	x := record.Deref("ts").AsTime()
	assert.Equal(t, expectedTs, x, "Timestamp is correct")

	// Test the #path header
	parser = startLegacyTest(t, fieldsWithTs, typesWithTs, "testpath")
	record, err = sendLegacyValues(parser, valsWithTs)
	require.NoError(t, err)

	path := record.Deref("_path").AsString()
	assert.Equal(t, path, "testpath", "Legacy _path field was set properly")

	// XXX test overriding separator, setSeparator
}

func assertInt64(t *testing.T, i int64, val *zed.Value, what string) {
	ok := val.Type == zed.TypeInt64
	assert.Truef(t, ok, "%s is type int", what)
	assert.Equalf(t, i, zed.DecodeInt(val.Bytes), "%s has value %d", what, i)
}

func TestNestedRecords(t *testing.T) {
	// Test the parser handling of nested records.
	// The schema used here touches several edge cases:
	//  - nested records separated by a regular field
	//  - adjacent nested records (nest2, nest3)
	//  - nested record containing nonadjacent fields (nest1)
	//  - nested record as the final column
	fields := []string{"a", "nest1.a", "nest1.b", "b", "nest2.y", "nest1.nestnest.c", "nest3.z"}
	types := []string{"int", "int", "int", "int", "int", "int", "int"}
	vals := []string{"1", "2", "3", "4", "5", "6", "7"}

	parser := startLegacyTest(t, fields, types, "")
	record, err := sendLegacyValues(parser, vals)
	require.NoError(t, err)
	require.NoError(t, zngio.Validate(record))

	// First check that the descriptor was created correctly
	cols := zed.TypeRecordOf(record.Type).Columns
	assert.Equal(t, 5, len(cols), "Descriptor has 5 columns")
	//assert.Equal(t, "_path", cols[0].Name, "Column 0 is _path")
	assert.Equal(t, "a", cols[0].Name, "Column 0 is a")
	assert.Equal(t, "nest1", cols[1].Name, "Column 1 is nest1")
	nest1Type, ok := cols[1].Type.(*zed.TypeRecord)
	assert.True(t, ok, "Columns nest1 is a record")
	assert.Equal(t, 3, len(nest1Type.Columns), "nest1 has 3 columns")
	assert.Equal(t, "a", nest1Type.Columns[0].Name, "First column in nest1 is a")
	assert.Equal(t, "b", nest1Type.Columns[1].Name, "Second column in nest1 is b")
	assert.Equal(t, "nestnest", nest1Type.Columns[2].Name, "Third column in nest1 is nestnest")
	nestnestType, ok := nest1Type.Columns[2].Type.(*zed.TypeRecord)
	assert.True(t, ok, "nest1.nestnest is a record")
	assert.Equal(t, 1, len(nestnestType.Columns), "nest1.nestnest has 1 column")
	assert.Equal(t, "c", nestnestType.Columns[0].Name, "First column in nest1.nestnest is c")
	assert.Equal(t, "b", cols[2].Name, "Column 2 is b")
	assert.Equal(t, "nest2", cols[3].Name, "Column 3 is nest2")
	nest2Type, ok := cols[3].Type.(*zed.TypeRecord)
	assert.True(t, ok, "Columns nest2 is a record")
	assert.Equal(t, 1, len(nest2Type.Columns), "nest2 has 1 column")
	assert.Equal(t, "y", nest2Type.Columns[0].Name, "column in nest2 is y")
	assert.Equal(t, "nest3", cols[4].Name, "Column 4 is nest3")
	nest3Type, ok := cols[4].Type.(*zed.TypeRecord)
	assert.True(t, ok, "Column nest3 is a record")
	assert.Equal(t, 1, len(nest3Type.Columns), "nest3 has 1 column")
	assert.Equal(t, "z", nest3Type.Columns[0].Name, "column in nest3 is z")

	// Now check the actual values
	assert.Equal(t, 1, int(record.Deref("a").AsInt()), "Field a has value 1")

	e := record.Deref("nest1")
	assert.Equal(t, nest1Type, e.Type, "Got right type for field nest1")
	assert.Equal(t, 2, int(e.Deref("a").AsInt()), "nest1.a")
	assert.Equal(t, 3, int(e.Deref("b").AsInt()), "nest1.b")

	e = e.Deref("nestnest")
	assert.Equal(t, nestnestType, e.Type, "Got right type for field nest1.nestnest")
	assert.Equal(t, 6, int(e.Deref("c").AsInt()), "nest1.nestnest.c")

	assert.Equal(t, 4, int(record.Deref("b").AsInt()), "Field b has value 4")

	e = record.Deref("nest2")
	assert.Equal(t, nest2Type, e.Type, "Got right type for field nest2")
	assert.Equal(t, 5, int(e.Deref("y").AsInt()), "nest2.y")

	e = record.Deref("nest3")
	assert.Equal(t, nest3Type, e.Type, "Got right type for field nest3")
	assert.Equal(t, 7, int(e.Deref("z").AsInt()), "nest3.z")
}

// Test things related to legacy zeek records that should cause the
// parser to generate errors.
func TestLegacyZeekInvalid(t *testing.T) {
	// Test that a non-standard value for empty_field is rejected
	parser := startTest(t, []string{separator, setSeparator})
	err := parser.ParseDirective([]byte("#empty_field\tboo"))
	assertError(t, err, "encountered bad header field", "#empty_field header")

	// Test that a non-standard value for unset_field is rejected
	parser = startTest(t, []string{separator, setSeparator})
	err = parser.ParseDirective([]byte("#unset_field\tboo"))
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

	// Test that too many values is an error
	parser = startTest(t, append(standardHeaders, fh, th))
	_, err = sendLegacyValues(parser, append(values, "extra"))
	assertError(t, err, "too many values", "wrong number of values")

	// Test that too few values is an error
	parser = startTest(t, append(standardHeaders, fh, th))
	_, err = sendLegacyValues(parser, values[:len(values)-2])
	assertError(t, err, "too few values", "wrong number of values")

	// XXX check invalid types?
}
