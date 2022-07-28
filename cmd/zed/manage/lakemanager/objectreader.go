package lakemanager

import (
	"context"

	"github.com/brimdata/zed/lake/api"
	"github.com/brimdata/zed/lake/data"
	"github.com/brimdata/zed/lakeparse"
	"github.com/brimdata/zed/order"
	"github.com/brimdata/zed/zio"
	"github.com/brimdata/zed/zson"
)

type PoolObjectReader struct {
	reader    zio.ReadCloser
	unmarshal *zson.UnmarshalZNGContext
}

func NewPoolObjectReader(ctx context.Context, lake api.Interface, head *lakeparse.Commitish,
	layout order.Layout) (*PoolObjectReader, error) {
	query, err := head.FromSpec("objects")
	if err != nil {
		return nil, err
	}
	if layout.Order == order.Asc {
		query += " | sort meta.first"
	} else {
		query += " | sort meta.last"
	}
	r, err := lake.Query(ctx, nil, query)
	if err != nil {
		return nil, err
	}
	return &PoolObjectReader{
		reader:    r,
		unmarshal: zson.NewZNGUnmarshaler(),
	}, nil
}

func (r *PoolObjectReader) Next() (*data.Object, error) {
	val, err := r.reader.Read()
	if val == nil || err != nil {
		return nil, err
	}
	var o data.Object
	if err := r.unmarshal.Unmarshal(val, &o); err != nil {
		return nil, err
	}
	return &o, nil
}

func (r *PoolObjectReader) Close() error {
	return r.reader.Close()
}
