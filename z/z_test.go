package z_test

import (
	"testing"

	"github.com/brimsec/zq/z"
	"github.com/stretchr/testify/assert"
)

func TestZ(t *testing.T) {
	var zc z.Context
	rec := zc.NewRecord(
		z.Int64("cnt", 12),
		zc.Array("a", z.Int64v(1), z.Int64v(2), z.Int64v(3)),
		z.String("s", "hello"),
		zc.Record("r",
			z.String("a", "foo"),
			z.String("b", "bar")))

	assert.Equal(t, rec.Type.String(), "record[cnt:int64,a:array[int64],s:string,r:record[a:string,b:string]]")
	assert.Equal(t, rec.String(), "record[12,array[1,2,3],hello,record[foo,bar]]")
}
