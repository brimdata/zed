package queryio

import (
	"errors"
	"fmt"
	"io"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/api"
	"github.com/brimdata/zed/zbuf"
	"github.com/brimdata/zed/zio/zngio"
	"github.com/brimdata/zed/zson"
)

type Query struct {
	reader *zngio.Reader
	closer io.Closer
}

// NewQuery returns a Query that reads a ZNG-encoded query response
// from rc and decodes it.  Closing the Query also closes rc.
func NewQuery(rc io.ReadCloser) *Query {
	return &Query{
		reader: zngio.NewReader(rc, zed.NewContext()),
		closer: rc,
	}
}

func (q *Query) Close() error {
	err := q.reader.Close()
	q.closer.Close()
	return err
}

func (q *Query) Read() (*zed.Value, error) {
	val, ctrl, err := q.reader.ReadPayload()
	if ctrl != nil {
		if ctrl.Format != zngio.ControlFormatZSON {
			return nil, fmt.Errorf("unsupported app encoding: %v", ctrl.Format)
		}
		value, err := zson.ParseValue(zed.NewContext(), string(ctrl.Bytes))
		if err != nil {
			return nil, fmt.Errorf("unable to parse Zed control message: %w (%s)", err, string(ctrl.Bytes))
		}
		var v interface{}
		if err := unmarshaler.Unmarshal(*value, &v); err != nil {
			return nil, fmt.Errorf("unable to unmarshal Zed control message: %w (%s)", err, string(ctrl.Bytes))
		}
		return nil, controlToError(v)
	}
	return val, err
}

func controlToError(ctrl interface{}) error {
	switch ctrl := ctrl.(type) {
	case *api.QueryChannelSet:
		return &zbuf.Control{zbuf.SetChannel(ctrl.ChannelID)}
	case *api.QueryChannelEnd:
		return &zbuf.Control{zbuf.EndChannel(ctrl.ChannelID)}
	case *api.QueryStats:
		return &zbuf.Control{zbuf.Progress(ctrl.Progress)}
	case *api.QueryError:
		return errors.New(ctrl.Error)
	default:
		return fmt.Errorf("unsupported control message: %T", ctrl)
	}
}
