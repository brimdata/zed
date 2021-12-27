package queryio

import (
	"errors"
	"fmt"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/api"
	"github.com/brimdata/zed/api/client"
	"github.com/brimdata/zed/zbuf"
	"github.com/brimdata/zed/zio/zngio"
	"github.com/brimdata/zed/zson"
)

type Query struct {
	reader *zngio.Reader
}

// NewQuery returns a Query that reads a ZNG-encoded query response
// from res and decodes it.
func NewQuery(res *client.Response) *Query {
	return &Query{
		reader: zngio.NewReader(res.Body, zed.NewContext()),
	}
}

func (q *Query) Read() (*zed.Value, error) {
	val, ctrl, err := q.reader.ReadPayload()
	if ctrl != nil {
		if ctrl.Encoding != zed.AppEncodingZSON {
			return nil, fmt.Errorf("unsupported app encoding: %v", ctrl.Encoding)
		}
		value, err := zson.ParseValue(zed.NewContext(), string(ctrl.Bytes))
		if err != nil {
			return nil, err
		}
		var v interface{}
		if err := unmarshaler.Unmarshal(value, &v); err != nil {
			return nil, err
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
