package queryio

import (
	"io"

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
	cid    int
	ctrl   bool
	start  nano.Ts
	writer zio.WriteCloser
}

func NewDriver(w io.WriteCloser, format string, ctrl bool) (*Driver, error) {
	d := &Driver{
		cid:   -1,
		ctrl:  ctrl,
		start: nano.Now(),
	}
	var err error
	switch format {
	case "zng":
		d.writer = NewZNGWriter(w)
	case "zjson":
		d.writer = NewZJSONWriter(w)
	case "json":
		// The json response should always be an array, so force array.
		d.writer = jsonio.NewWriter(w, jsonio.WriterOpts{ForceArray: true})
	default:
		d.writer, err = anyio.NewWriter(w, anyio.WriterOpts{Format: format})
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
	for i := 0; i < batch.Length(); i++ {
		if err := d.writer.Write(batch.Index(i)); err != nil {
			return err
		}
	}
	batch.Unref()
	return nil
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
	if ctrl, ok := d.writer.(ControlWriter); ok && d.ctrl {
		return ctrl.WriteControl(value)
	}
	return nil
}

func (d *Driver) Error(err error) {
	d.WriteControl(api.QueryError{err.Error()})
}

func (d *Driver) Close() error {
	return d.writer.Close()
}
