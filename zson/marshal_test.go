package zson_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/brimdata/zed/zio"
	"github.com/brimdata/zed/zio/zngio"
	"github.com/brimdata/zed/zio/zsonio"
	"github.com/brimdata/zed/zng"
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
	expectedRose := `{MyColor:"red"} (=Plant)`
	flamingo := Thing(&Animal{"pink"})
	expectedFlamingo := `{MyColor:"pink"} (=Animal)`

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
	require.NoError(t, err)

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
	assert.Equal(t, `[1 (int8),2 (int8),3 (int8)] (=0)`, z)

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
	assert.Equal(t, `true (=Roll)`, z)
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

func recToZSON(t *testing.T, rec *zng.Record) string {
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
{A:0x00010203 (=ID),B:0x04050607 (ID)} (=IDRecord)
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
{B:null (bytes)}
`
	assert.Equal(t, trim(exp), recToZSON(t, rec))

	s := SliceRecord{S: nil}
	m = zson.NewZNGMarshaler()
	rec, err = m.MarshalRecord(s)
	require.NoError(t, err)
	require.NotNil(t, rec)

	exp = `
{S:null (0=([bytes]))}
	`
	assert.Equal(t, trim(exp), recToZSON(t, rec))

}

type RecordWithInterfaceSlice struct {
	X string
	S []Thing
}

func TestBug2575(t *testing.T) {
	// Bug #2575 is preventing "-f lake" from working on transactions that
	// have arrays of actions.Interface.  This test repros the problem.
	// Skipping until #2575 is addressed.
	t.Skip()
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
	writer := zngio.NewWriter(zio.NopCloser(&buffer), zngio.WriterOpts{})
	recExpected := zng.NewRecord(zv.Type, zv.Bytes)
	writer.Write(recExpected)
	writer.Close()

	r := bytes.NewReader(buffer.Bytes())
	reader := zngio.NewReader(r, zson.NewContext())
	recActual, err := reader.Read()
	exp, err := zson.FormatValue(recExpected.Value)
	require.NoError(t, err)
	actual, err := zson.FormatValue(recActual.Value)
	require.NoError(t, err)
	assert.Equal(t, trim(exp), actual)
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
	Field zng.Value `zng:"field"`
}

func TestZNGValueField(t *testing.T) {
	// Embed a Zed int64 inside a go struct as a zng.Value
	zngValueField := &ZNGValueField{
		Name:  "test1",
		Field: zng.Value{zng.TypeInt64, zng.EncodeInt(123)},
	}
	m := zson.NewZNGMarshaler()
	m.Decorate(zson.StyleSimple)
	zv, err := m.Marshal(zngValueField)
	require.NoError(t, err)
	expected := `{Name:"test1",field:123} (=ZNGValueField)`
	actual, err := zson.FormatValue(zv)
	require.NoError(t, err)
	assert.Equal(t, trim(expected), trim(actual))
	u := zson.NewZNGUnmarshaler()
	var out ZNGValueField
	err = u.Unmarshal(zv, &out)
	require.NoError(t, err)
	assert.Equal(t, *zngValueField, out)
	// Embed a Zed record inside a go struct as a zng.Value
	z := `{s:"foo",a:[1,2,3]}`
	zv2, err := zson.ParseValue(zson.NewContext(), z)
	require.NoError(t, err)
	zngValueField2 := &ZNGValueField{
		Name:  "test2",
		Field: zv2,
	}
	m2 := zson.NewZNGMarshaler()
	m2.Decorate(zson.StyleSimple)
	zv3, err := m2.Marshal(zngValueField2)
	require.NoError(t, err)
	expected2 := `{Name:"test2",field:{s:"foo",a:[1,2,3]}} (=ZNGValueField)`
	actual2, err := zson.FormatValue(zv3)
	require.NoError(t, err)
	assert.Equal(t, trim(expected2), trim(actual2))
	u2 := zson.NewZNGUnmarshaler()
	var out2 ZNGValueField
	err = u2.Unmarshal(zv3, &out2)
	require.NoError(t, err)
	assert.Equal(t, *zngValueField2, out2)
}
