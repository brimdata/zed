package queryio

import (
	"context"
	"errors"
	"fmt"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/api"
	"github.com/brimdata/zed/api/client"
	"github.com/brimdata/zed/driver"
	"github.com/brimdata/zed/zbuf"
	"github.com/brimdata/zed/zio/zngio"
)

func RunClientResponse(ctx context.Context, d driver.Driver, res *client.Response) (zbuf.ScannerStats, error) {
	format, err := api.MediaTypeToFormat(res.ContentType)
	if err != nil {
		return zbuf.ScannerStats{}, err
	}
	if format != "zng" {
		return zbuf.ScannerStats{}, fmt.Errorf("unsupported format: %s", format)
	}
	run := &runner{driver: d}
	r := NewZNGReader(zngio.NewReader(res.Body, zed.NewContext()))
	for ctx.Err() == nil {
		rec, ctrl, err := r.ReadPayload()
		if err != nil {
			return run.stats, err
		}
		if ctrl != nil {
			if err := run.handleCtrl(ctrl); err != nil {
				return run.stats, err
			}
			continue
		}
		if rec != nil {
			if err := run.Write(rec); err != nil {
				return run.stats, err
			}
			continue
		}
		return run.stats, nil
	}
	return run.stats, ctx.Err()
}

type runner struct {
	driver driver.Driver
	cid    int
	recs   []*zed.Record
	stats  zbuf.ScannerStats
}

func (r *runner) Write(rec *zed.Record) error {
	return r.driver.Write(r.cid, &zbuf.Array{rec})
}

func (r *runner) handleCtrl(ctrl interface{}) error {
	switch ctrl := ctrl.(type) {
	case *api.QueryChannelSet:
		r.cid = ctrl.ChannelID
	case *api.QueryChannelEnd:
		return r.driver.ChannelEnd(ctrl.ChannelID)
	case *api.QueryStats:
		r.stats = zbuf.ScannerStats(ctrl.ScannerStats)
		return r.driver.Stats(ctrl.ScannerStats)
	case *api.QueryWarning:
		return r.driver.Warn(ctrl.Warning)
	case *api.QueryError:
		return errors.New(ctrl.Error)
	default:
		return fmt.Errorf("unsupported control message: %T", ctrl)
	}
	return nil
}
