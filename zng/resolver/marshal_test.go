package resolver_test

import (
	"errors"
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

func tzngToRec(t *testing.T, zctx *resolver.Context, tzng string) *zng.Record {
	r := tzngio.NewReader(strings.NewReader(tzng), zctx)
	rec, err := r.Read()
	require.NoError(t, err)
	return rec
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

func TestIPType(t *testing.T) {
	addr := net.ParseIP("192.168.1.1").To4()
	require.NotNil(t, addr)
	s := TestIP{Addr: addr}
	zctx := resolver.NewContext()
	rec, err := resolver.MarshalRecord(zctx, s)
	require.NoError(t, err)
	require.NotNil(t, rec)

	exp := `
#0:record[Addr:ip]
0:[192.168.1.1;]
`
	assert.Equal(t, trim(exp), rectzng(t, rec))

	var tip TestIP
	err = resolver.UnmarshalRecord(zctx, rec, &tip)
	require.NoError(t, err)
	require.Equal(t, s, tip)
}

func TestUnmarshalRecord(t *testing.T) {
	type T3 struct {
		T3f1 int32
		T3f2 float32
	}
	type T2 struct {
		T2f1 T3
		T2f2 string
	}
	type T1 struct {
		T1f1 *T2 `zng:"top"`
	}
	v1 := T1{
		T1f1: &T2{T2f1: T3{T3f1: 1, T3f2: 1.0}, T2f2: "t2f2-string1"},
	}
	zctx := resolver.NewContext()
	rec, err := resolver.MarshalRecord(zctx, v1)
	require.NoError(t, err)
	require.NotNil(t, rec)

	exp := `
#0:record[top:record[T2f1:record[T3f1:int32,T3f2:float64],T2f2:string]]
0:[[[1;1;]t2f2-string1;]]
`
	require.Equal(t, trim(exp), rectzng(t, rec))

	zctx = resolver.NewContext()
	rec = tzngToRec(t, zctx, exp)

	var v2 T1
	err = resolver.UnmarshalRecord(zctx, rec, &v2)
	require.NoError(t, err)
	require.Equal(t, v1, v2)

	type T4 struct {
		T4f1 *T2 `zng:"top"`
	}
	var v3 *T4
	err = resolver.UnmarshalRecord(zctx, rec, &v3)
	require.NoError(t, err)
	require.NotNil(t, v3)
	require.NotNil(t, v3.T4f1)
	require.Equal(t, *v1.T1f1, *v3.T4f1)
}

func TestUnmarshalSlice(t *testing.T) {
	type T1 struct {
		T1f1 []bool
	}
	v1 := T1{
		T1f1: []bool{true, false, true},
	}
	zctx := resolver.NewContext()
	rec, err := resolver.MarshalRecord(zctx, v1)
	require.NoError(t, err)
	require.NotNil(t, rec)

	var v2 T1
	err = resolver.UnmarshalRecord(zctx, rec, &v2)
	require.NoError(t, err)
	require.Equal(t, v1, v2)

	type T2 struct {
		Field1 []*int
	}
	intp := func(x int) *int { return &x }
	v3 := T2{
		Field1: []*int{intp(1), intp(2)},
	}
	zctx = resolver.NewContext()
	rec, err = resolver.MarshalRecord(zctx, v3)
	require.NoError(t, err)
	require.NotNil(t, rec)

	var v4 T2
	err = resolver.UnmarshalRecord(zctx, rec, &v4)
	require.NoError(t, err)
	require.Equal(t, v1, v2)
}

type testMarshaler string

func (m testMarshaler) MarshalZNG(zctx *resolver.Context, b *zcode.Builder) (zng.Type, error) {
	return resolver.Marshal(zctx, b, "marshal-"+string(m))
}

func (m *testMarshaler) UnmarshalZNG(zctx *resolver.Context, zt zng.Type, zb zcode.Bytes) error {
	var s string
	if err := resolver.Unmarshal(zctx, zt, zb, &s); err != nil {
		return err
	}
	ss := strings.Split(s, "-")
	if len(ss) != 2 && ss[0] != "marshal" {
		return errors.New("bad value")
	}
	*m = testMarshaler(ss[1])
	return nil
}

func TestMarshalInterface(t *testing.T) {
	type rectype struct {
		M1 *testMarshaler
		M2 testMarshaler
	}
	m1 := testMarshaler("m1")
	r1 := rectype{M1: &m1, M2: testMarshaler("m2")}
	rec, err := resolver.MarshalRecord(resolver.NewContext(), r1)
	require.NoError(t, err)
	require.NotNil(t, rec)

	exp := `
#0:record[M1:string,M2:string]
0:[marshal-m1;marshal-m2;]
`
	assert.Equal(t, trim(exp), rectzng(t, rec))

	var r2 rectype
	err = resolver.UnmarshalRecord(resolver.NewContext(), rec, &r2)
	require.NoError(t, err)
	assert.Equal(t, "m1", string(*r1.M1))
	assert.Equal(t, "m2", string(r1.M2))
}
