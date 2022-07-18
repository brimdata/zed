package zson_test

import (
	"bytes"
	"net"
	"net/netip"
	"strings"
	"testing"
	"time"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/zio"
	"github.com/brimdata/zed/zio/zngio"
	"github.com/brimdata/zed/zio/zsonio"
	"github.com/brimdata/zed/zson"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func trim(s string) string {
	return strings.TrimSpace(s) + "\n"
}

type Thing interface {
	Color() string
}

type Plant struct {
	MyColor string
}

func (p *Plant) Color() string { return p.MyColor }

type Animal struct {
	MyColor string
}

func (a *Animal) Color() string { return a.MyColor }

func TestInterfaceMarshal(t *testing.T) {
	rose := Thing(&Plant{"red"})
	expectedRose := `{MyColor:"red"}(=Plant)`
	flamingo := Thing(&Animal{"pink"})
	expectedFlamingo := `{MyColor:"pink"}(=Animal)`

	m := zson.NewMarshaler()
	m.Decorate(zson.StyleSimple)

	zsonRose, err := m.Marshal(rose)
	require.NoError(t, err)
	assert.Equal(t, trim(expectedRose), trim(zsonRose))

	zsonFlamingo, err := m.Marshal(flamingo)
	require.NoError(t, err)
	assert.Equal(t, trim(expectedFlamingo), trim(zsonFlamingo))

	u := zson.NewUnmarshaler()
	u.Bind(Plant{}, Animal{})
	var thing Thing

	err = u.Unmarshal(zsonRose, &thing)
	require.NoError(t, err)
	assert.Equal(t, "red", thing.Color())

	err = u.Unmarshal(zsonFlamingo, &thing)
	require.NoError(t, err)
	assert.Equal(t, "pink", thing.Color())
}

type Roll bool

func TestMarshal(t *testing.T) {
	z, err := zson.Marshal("hello, world")
	require.NoError(t, err)
	assert.Equal(t, `"hello, world"`, z)

	aIn := []int8{1, 2, 3}
	z, err = zson.Marshal(aIn)
	require.NoError(t, err)
	assert.Equal(t, `[1(int8),2(int8),3(int8)]`, z)

	var v interface{}
	err = zson.Unmarshal(z, &v)
	require.NoError(t, err)
	aOut, ok := v.([]int8)
	assert.Equal(t, ok, true)
	assert.Equal(t, aIn, aOut)

	m := zson.NewMarshaler()
	m.Decorate(zson.StyleSimple)
	z, err = m.Marshal(Roll(true))
	require.NoError(t, err)
	assert.Equal(t, `true(=Roll)`, z)
}

type BytesRecord struct {
	B []byte
}

type BytesArrayRecord struct {
	A [3]byte
}

type ID [4]byte

type IDRecord struct {
	A ID
	B ID
}

type IDSlice []byte

type SliceRecord struct {
	S []IDSlice
}

func recToZSON(t *testing.T, rec *zed.Value) string {
	var b strings.Builder
	w := zsonio.NewWriter(zio.NopCloser(&b), zsonio.WriterOpts{})
	err := w.Write(rec)
	require.NoError(t, err)
	return b.String()
}

func TestBytes(t *testing.T) {
	b := BytesRecord{B: []byte{1, 2, 3}}
	m := zson.NewZNGMarshaler()
	rec, err := m.MarshalRecord(b)
	require.NoError(t, err)
	require.NotNil(t, rec)

	exp := `
{B:0x010203}
`
	assert.Equal(t, trim(exp), recToZSON(t, rec))

	a := BytesArrayRecord{A: [3]byte{4, 5, 6}}
	rec, err = m.MarshalRecord(a)
	require.NoError(t, err)
	require.NotNil(t, rec)

	exp = `
{A:0x040506}
`
	assert.Equal(t, trim(exp), recToZSON(t, rec))

	id := IDRecord{A: ID{0, 1, 2, 3}, B: ID{4, 5, 6, 7}}
	m = zson.NewZNGMarshaler()
	m.Decorate(zson.StyleSimple)
	rec, err = m.MarshalRecord(id)
	require.NoError(t, err)
	require.NotNil(t, rec)

	exp = `
{A:0x00010203(=ID),B:0x04050607(ID)}(=IDRecord)
	`
	assert.Equal(t, trim(exp), recToZSON(t, rec))

	var id2 IDRecord
	u := zson.NewZNGUnmarshaler()
	u.Bind(IDRecord{}, ID{})
	err = zson.UnmarshalZNGRecord(rec, &id2)
	require.NoError(t, err)
	assert.Equal(t, id, id2)

	b2 := BytesRecord{B: nil}
	m = zson.NewZNGMarshaler()
	rec, err = m.MarshalRecord(b2)
	require.NoError(t, err)
	require.NotNil(t, rec)

	exp = `
{B:null(bytes)}
`
	assert.Equal(t, trim(exp), recToZSON(t, rec))

	s := SliceRecord{S: nil}
	m = zson.NewZNGMarshaler()
	rec, err = m.MarshalRecord(s)
	require.NoError(t, err)
	require.NotNil(t, rec)

	exp = `
{S:null([bytes])}
	`
	assert.Equal(t, trim(exp), recToZSON(t, rec))

}

type RecordWithInterfaceSlice struct {
	X string
	S []Thing
}

func TestMixedTypeArray(t *testing.T) {
	x := &RecordWithInterfaceSlice{
		X: "hello",
		S: []Thing{
			&Plant{"red"},
			&Animal{"blue"},
		},
	}
	m := zson.NewZNGMarshaler()
	m.Decorate(zson.StyleSimple)

	zv, err := m.Marshal(x)
	require.NoError(t, err)

	var buffer bytes.Buffer
	writer := zngio.NewWriter(zio.NopCloser(&buffer))
	recExpected := zed.NewValue(zv.Type, zv.Bytes)
	writer.Write(recExpected)
	writer.Close()

	reader := zngio.NewReader(zed.NewContext(), &buffer)
	defer reader.Close()
	recActual, err := reader.Read()
	exp, err := zson.FormatValue(recExpected)
	require.NoError(t, err)
	actual, err := zson.FormatValue(recActual)
	require.NoError(t, err)
	assert.Equal(t, trim(exp), trim(actual))
	// Double check that all the proper typing made it into the implied union.
	assert.Equal(t, `{X:"hello",S:[[{MyColor:"red"}(=Plant),{MyColor:"blue"}(=Animal)]]}(=RecordWithInterfaceSlice)`, actual)
}

type Foo struct {
	A int
	a int
}

func TestUnexported(t *testing.T) {
	f := &Foo{1, 2}
	m := zson.NewZNGMarshaler()
	_, err := m.Marshal(f)
	require.NoError(t, err)
}

type ZNGValueField struct {
	Name  string
	Field zed.Value `zed:"field"`
}

func TestZNGValueField(t *testing.T) {
	// Include a Zed int64 inside a Go struct as a zed.Value field.
	zngValueField := &ZNGValueField{
		Name:  "test1",
		Field: zed.Value{zed.TypeInt64, zed.EncodeInt(123)},
	}
	m := zson.NewZNGMarshaler()
	m.Decorate(zson.StyleSimple)
	zv, err := m.Marshal(zngValueField)
	require.NoError(t, err)
	expected := `{Name:"test1",field:123}(=ZNGValueField)`
	actual, err := zson.FormatValue(zv)
	require.NoError(t, err)
	assert.Equal(t, trim(expected), trim(actual))
	u := zson.NewZNGUnmarshaler()
	var out ZNGValueField
	err = u.Unmarshal(zv, &out)
	require.NoError(t, err)
	assert.Equal(t, *zngValueField, out)
	// Include a Zed record inside a Go struct in a zed.Value field.
	z := `{s:"foo",a:[1,2,3]}`
	zv2, err := zson.ParseValue(zed.NewContext(), z)
	require.NoError(t, err)
	zngValueField2 := &ZNGValueField{
		Name:  "test2",
		Field: *zv2,
	}
	m2 := zson.NewZNGMarshaler()
	m2.Decorate(zson.StyleSimple)
	zv3, err := m2.Marshal(zngValueField2)
	require.NoError(t, err)
	expected2 := `{Name:"test2",field:{s:"foo",a:[1,2,3]}}(=ZNGValueField)`
	actual2, err := zson.FormatValue(zv3)
	require.NoError(t, err)
	assert.Equal(t, trim(expected2), trim(actual2))
	u2 := zson.NewZNGUnmarshaler()
	var out2 ZNGValueField
	err = u2.Unmarshal(zv3, &out2)
	require.NoError(t, err)
	assert.Equal(t, *zngValueField2, out2)
}

func TestJSONFieldTag(t *testing.T) {
	const expected = `{value:"test"}`
	type jsonTag struct {
		Value string `json:"value"`
	}
	s, err := zson.Marshal(jsonTag{Value: "test"})
	require.NoError(t, err)
	assert.Equal(t, expected, s)
	var j jsonTag
	require.NoError(t, zson.Unmarshal(s, &j))
	assert.Equal(t, jsonTag{Value: "test"}, j)
}

func TestIgnoreField(t *testing.T) {
	type s struct {
		Value  string       `zed:"value"`
		Ignore func() error `zed:"-"`
	}
	b, err := zson.Marshal(s{Value: "test"})
	require.NoError(t, err)
	assert.Equal(t, `{value:"test"}`, b)
	var v s
	require.NoError(t, zson.Unmarshal(b, &v))
	assert.Equal(t, s{Value: "test"}, v)
}

func TestMarshalNetIP(t *testing.T) {
	before := net.ParseIP("10.0.0.1")
	b, err := zson.Marshal(before)
	require.NoError(t, err)
	assert.Equal(t, `10.0.0.1`, b)
	var after net.IP
	err = zson.Unmarshal(string(b), &after)
	require.NoError(t, err)
	assert.Equal(t, before, after)
}

func TestMarshalNetipAddr(t *testing.T) {
	before := netip.MustParseAddr("10.0.0.1")
	b, err := zson.Marshal(before)
	require.NoError(t, err)
	assert.Equal(t, `10.0.0.1`, b)
	var after netip.Addr
	err = zson.Unmarshal(string(b), &after)
	require.NoError(t, err)
	assert.Equal(t, before, after)
}

func TestMarshalDecoratedIPs(t *testing.T) {
	m := zson.NewMarshaler()
	// Make sure IPs don't get decorated with Go type and just
	// appear as native Zed IPs.
	m.Decorate(zson.StyleSimple)
	b, err := m.Marshal(net.ParseIP("142.250.72.142"))
	require.NoError(t, err)
	assert.Equal(t, `142.250.72.142`, b)
	b, err = m.Marshal(netip.MustParseAddr("142.250.72.142"))
	require.NoError(t, err)
	assert.Equal(t, `142.250.72.142`, b)
}

func TestMarshalGoTime(t *testing.T) {
	tm, _ := time.Parse(time.RFC3339, "2006-01-02T15:04:05.123Z")
	b, err := zson.Marshal(tm)
	require.NoError(t, err)
	assert.Equal(t, `2006-01-02T15:04:05.123Z`, b)
}
