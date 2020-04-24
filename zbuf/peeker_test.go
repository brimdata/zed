package zbuf_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/brimsec/zq/zbuf"
	"github.com/brimsec/zq/zio/tzngio"
	"github.com/brimsec/zq/zng/resolver"
)

func newTextReader(logs string) *tzngio.Reader {
	logs = strings.TrimSpace(logs) + "\n"
	return tzngio.NewReader(strings.NewReader(logs), resolver.NewContext())
}

const input = `
#0:record[key:string,value:string]
0:[key1;value1;]
0:[key2;value2;]
0:[key3;value3;]
0:[key4;value4;]
0:[key5;value5;]
0:[key6;value6;]`

func TestPeeker(t *testing.T) {
	stream := newTextReader(input)
	peeker := zbuf.NewPeeker(stream)
	rec1, err := peeker.Peek()
	if err != nil {
		t.Error(err)
	}
	rec2, err := peeker.Peek()
	if err != nil {
		t.Error(err)
	}
	if !bytes.Equal(rec1.Raw, rec2.Raw) {
		t.Error("rec1 != rec2")
	}
	rec3, err := peeker.Read()
	if err != nil {
		t.Error(err)
	}
	if !bytes.Equal(rec1.Raw, rec3.Raw) {
		t.Error("rec1 != rec3")
	}
	rec4, err := peeker.Peek()
	if err != nil {
		t.Error(err)
	}
	if bytes.Equal(rec3.Raw, rec4.Raw) {
		t.Error("rec3 == rec4")
	}
	rec5, err := peeker.Read()
	if err != nil {
		t.Error(err)
	}
	if !bytes.Equal(rec4.Raw, rec5.Raw) {
		t.Error("rec4 != rec5")
	}
}
