package queryio

import (
	"io"
	"net/http"

	"github.com/brimdata/zed/api"
	"github.com/brimdata/zed/pkg/nano"
	"github.com/brimdata/zed/zbuf"
	"github.com/brimdata/zed/zio"
	"github.com/brimdata/zed/zio/anyio"
	"github.com/brimdata/zed/zio/jsonio"
	"github.com/brimdata/zed/zio/zjsonio"
	"github.com/brimdata/zed/zio/zngio"
)

type controlWriter interface {
	zio.WriteCloser
	WriteControl(interface{}) error
}

type Writer struct {
	cid     int
	start   nano.Ts
	writer  zio.WriteCloser
	flusher http.Flusher
}

func NewWriter(w io.WriteCloser, format string, flusher http.Flusher, ctrl bool) (*Writer, error) {
	d := &Writer{
		cid:     -1,
		start:   nano.Now(),
		flusher: flusher,
	}
	var err error
	switch format {
	case "zng":
		if ctrl {
			d.writer = NewZNGWriter(w)
		} else {
			d.writer = zngio.NewWriter(w)
		}
	case "zjson":
		if ctrl {
			d.writer = NewZJSONWriter(w)
		} else {
			d.writer = zjsonio.NewWriter(w)
		}
	case "json":
		// A JSON response is always an array.
		d.writer = jsonio.NewArrayWriter(w)
	case "ndjson":
		d.writer = jsonio.NewWriter(w)
	default:
		d.writer, err = anyio.NewWriter(zio.NopCloser(w), anyio.WriterOpts{Format: format})
	}
	return d, err
}

func (w *Writer) WriteBatch(cid int, batch zbuf.Batch) error {
	if w.cid != cid {
		w.cid = cid
		if err := w.WriteControl(api.QueryChannelSet{cid}); err != nil {
			return err
		}
	}
	defer batch.Unref()
	return zbuf.WriteBatch(w.writer, batch)
}

func (w *Writer) WhiteChannelEnd(channelID int) error {
	return w.WriteControl(api.QueryChannelEnd{channelID})
}

func (w *Writer) WriteProgress(stats zbuf.Progress) error {
	v := api.QueryStats{
		StartTime:  w.start,
		UpdateTime: nano.Now(),
		Progress:   stats,
	}
	return w.WriteControl(v)
}

func (w *Writer) WriteError(err error) {
	w.WriteControl(api.QueryError{err.Error()})
}

func (w *Writer) WriteControl(value interface{}) error {
	var err error
	if ctrl, ok := w.writer.(controlWriter); ok {
		err = ctrl.WriteControl(value)
		if w.flusher != nil {
			w.flusher.Flush()
		}
	}
	return err
}

func (w *Writer) Close() error {
	return w.writer.Close()
}
