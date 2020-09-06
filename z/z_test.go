package z_test

import (
	"testing"

	"github.com/brimsec/zq/z"
	"github.com/stretchr/testify/assert"
)

func TestZ(t *testing.T) {
	var zc z.Context
	rec := zc.Record(
		z.Int64("cnt", 12),
		z.String("s", "hello"),
		zc.RecordField("rec",
			z.String("a", "foo"),
			z.String("b", "bar")))

	assert.Equal(t, rec.String(), "record[12,hello,record[foo,bar]]")
}
