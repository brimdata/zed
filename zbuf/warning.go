package zbuf

import (
	"fmt"

	"github.com/brimsec/zq/zng"
)

type WarningReader struct {
	zr Reader
	ch chan string
}

// WarningReader returns a Reader that reads from zr.  Any error encountered is
// sent to ch, and then a nil *zng.Record and nil error are returned.
func NewWarningReader(zr Reader, ch chan string) *WarningReader {
	return &WarningReader{zr: zr, ch: ch}
}

func (w *WarningReader) Read() (*zng.Record, error) {
	rec, err := w.zr.Read()
	if err != nil {
		w.ch <- fmt.Sprintf("%s: %s", w.zr, err)
		return nil, nil
	}
	return rec, nil
}
