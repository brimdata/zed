package zbuf

import (
	"fmt"

	"github.com/brimdata/zq/zng"
)

type Warner interface {
	Warn(msg string) error
}

type WarningReader struct {
	zr Reader
	wn Warner
}

// NewWarningReader returns a Reader that reads from zr.  Any error
// encountered results in a call to w with the warning as prameter, and
// then a nil *zng.Record and nil error are returned.
func NewWarningReader(zr Reader, w Warner) Reader {
	return &WarningReader{zr: zr, wn: w}
}

func (w *WarningReader) Read() (*zng.Record, error) {
	rec, err := w.zr.Read()
	if err != nil {
		w.wn.Warn(fmt.Sprintf("%s: %s", w.zr, err))
		return nil, nil
	}
	return rec, nil
}
