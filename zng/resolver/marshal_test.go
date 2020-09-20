package resolver_test

import (
	"net"
	"strings"
	"testing"

	"github.com/brimsec/zq/zcode"
	"github.com/brimsec/zq/zio"
	"github.com/brimsec/zq/zio/tzngio"
	"github.com/brimsec/zq/zng"
	"github.com/brimsec/zq/zng/resolver"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func trim(s string) string {
	return strings.TrimSpace(s) + "\n"
}

func rectzng(t *testing.T, rec *zng.Record) string {
	var b strings.Builder
	w := tzngio.NewWriter(zio.NopCloser(&b))
	err := w.Write(rec)
	require.NoError(t, err)
	return b.String()
}

func TestMarshal(t *testing.T) {
	type S2 struct {
		Field2 string `zng:"f2"`
		Field3 int
	}
	type S1 struct {
		Field1  string
		Sub1    S2
		PField1 *bool
	}
	zctx := resolver.NewContext()
	rec, err := resolver.MarshalRecord(zctx, S1{
		Field1: "value1",
		Sub1: S2{
			Field2: "value2",
			Field3: -1,
		},
	})
	require.NoError(t, err)
	require.NotNil(t, rec)

	exp := `
#0:record[Field1:string,Sub1:record[f2:string,Field3:int64],PField1:bool]
0:[value1;[value2;-1;]-;]
`
	assert.Equal(t, trim(exp), rectzng(t, rec))
}

type Thing struct {
	A string `zng:"a"`
	B int
}

type Foo struct {
	Thing
}

type Bar interface {
	MarshalZNG(*resolver.Context, *zcode.Builder) (zng.Type, error)
}

func (f *Foo) MarshalZNG(zctx *resolver.Context, b *zcode.Builder) (zng.Type, error) {
	return resolver.Marshal(zctx, b, f.Thing)
}

func TestMarshalInteface(t *testing.T) {
	f := &Foo{Thing{A: "hello", B: 123}}
	b := Bar(f)
	zctx := resolver.NewContext()
	rec, err := resolver.MarshalRecord(zctx, b)
	require.NoError(t, err)
	require.NotNil(t, rec)

	exp := `
#0:record[Thing:record[a:string,B:int64]]
0:[[hello;123;]]
`
	assert.Equal(t, trim(exp), rectzng(t, rec))
}

type Things struct {
	Things []Thing
}

func TestMarshalSlice(t *testing.T) {
	s := []Thing{{"hello", 123}, {"world", 0}}
	r := Things{s}
	zctx := resolver.NewContext()
	rec, err := resolver.MarshalRecord(zctx, r)
	require.NoError(t, err)
	require.NotNil(t, rec)

	exp := `
#0:record[Things:array[record[a:string,B:int64]]]
0:[[[hello;123;][world;0;]]]
`
	assert.Equal(t, trim(exp), rectzng(t, rec))

	var empty []Thing
	r2 := Things{empty}
	rec2, err := resolver.MarshalRecord(zctx, r2)
	require.NoError(t, err)
	require.NotNil(t, rec2)

	exp2 := `
#0:record[Things:array[record[a:string,B:int64]]]
0:[[]]
`
	assert.Equal(t, trim(exp2), rectzng(t, rec2))
}

type TestIP struct {
	Addr net.IP
}

func TestMarshalIP(t *testing.T) {
	s := TestIP{Addr: net.ParseIP("192.168.1.1")}
	zctx := resolver.NewContext()
	rec, err := resolver.MarshalRecord(zctx, s)
	require.NoError(t, err)
	require.NotNil(t, rec)

	exp := `
#0:record[Addr:ip]
0:[192.168.1.1;]
`
	assert.Equal(t, trim(exp), rectzng(t, rec))
}
