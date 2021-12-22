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
)

type ControlWriter interface {
	zio.WriteCloser
	WriteControl(interface{}) error
}

type Driver struct {
	cid     int
	ctrl    bool
	start   nano.Ts
	writer  io.Writer
	zwriter zio.WriteCloser
}

func NewDriver(w io.Writer, format string, ctrl bool) (*Driver, error) {
	d := &Driver{
		cid:    -1,
		ctrl:   ctrl,
		start:  nano.Now(),
		writer: w,
	}
	var err error
	switch format {
	case "zng":
		d.zwriter = NewZNGWriter(w)
	case "zjson":
		d.zwriter = NewZJSONWriter(w)
	case "json":
		// The json response should always be an array, so force array.
		d.zwriter = jsonio.NewWriter(zio.NopCloser(w), jsonio.WriterOpts{ForceArray: true})
	default:
		d.zwriter, err = anyio.NewWriter(zio.NopCloser(w), anyio.WriterOpts{Format: format})
	}
	return d, err
}

func (d *Driver) Warn(warning string) error {
	return d.WriteControl(api.QueryWarning{Warning: warning})
}

func (d *Driver) Write(cid int, batch zbuf.Batch) error {
	if d.cid == -1 || d.cid != cid {
		d.cid = cid
		if err := d.WriteControl(api.QueryChannelSet{cid}); err != nil {
			return err
		}
	}
	defer batch.Unref()
	return zbuf.WriteBatch(d.zwriter, batch)
}

func (d *Driver) ChannelEnd(channelID int) error {
	return d.WriteControl(api.QueryChannelEnd{channelID})
}

func (d *Driver) Stats(stats zbuf.ScannerStats) error {
	v := api.QueryStats{
		StartTime:    d.start,
		UpdateTime:   nano.Now(),
		ScannerStats: stats,
	}
	return d.WriteControl(v)
}

func (d *Driver) WriteControl(value interface{}) error {
	if ctrl, ok := d.zwriter.(ControlWriter); ok && d.ctrl {
		err := ctrl.WriteControl(value)
		if flusher, ok := d.writer.(http.Flusher); ok {
			flusher.Flush()
		}
		return err
	}
	return nil
}

func (d *Driver) Error(err error) {
	d.WriteControl(api.QueryError{err.Error()})
}

func (d *Driver) Close() error {
	return d.zwriter.Close()
}
