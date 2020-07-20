package zbuf_test

import (
	"strings"
	"testing"

	"github.com/brimsec/zq/zbuf"
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
	rec, err := zbuf.Marshal(zctx, S1{
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
