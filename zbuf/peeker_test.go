package zbuf_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/brimdata/zed/zbuf"
	"github.com/brimdata/zed/zio/tzngio"
	"github.com/brimdata/zed/zng/resolver"
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
	if !bytes.Equal(rec1.Bytes, rec2.Bytes) {
		t.Error("rec1 != rec2")
	}
	rec3, err := peeker.Read()
	if err != nil {
		t.Error(err)
	}
	if !bytes.Equal(rec1.Bytes, rec3.Bytes) {
		t.Error("rec1 != rec3")
	}
	rec4, err := peeker.Peek()
	if err != nil {
		t.Error(err)
	}
	if bytes.Equal(rec3.Bytes, rec4.Bytes) {
		t.Error("rec3 == rec4")
	}
	rec5, err := peeker.Read()
	if err != nil {
		t.Error(err)
	}
	if !bytes.Equal(rec4.Bytes, rec5.Bytes) {
		t.Error("rec4 != rec5")
	}
}
