package api

import (
	"github.com/brimdata/zed/api"
	"github.com/brimdata/zed/zbuf"
	"github.com/brimdata/zed/zson"
)

type queryDriver struct {
	unmarshaler *zson.UnmarshalZNGContext
	results     []interface{}
}

func newQueryDriver(types ...interface{}) *queryDriver {
	u := zson.NewZNGUnmarshaler()
	u.Bind(types...)
	return &queryDriver{unmarshaler: u}
}

func (d *queryDriver) Write(channelID int, batch zbuf.Batch) error {
	for _, rec := range batch.Records() {
		var v interface{}
		if err := d.unmarshaler.Unmarshal(rec.Value, &v); err != nil {
			return err
		}
		d.results = append(d.results, v)
	}
	return nil
}

func (*queryDriver) Warn(msg string) error {
	return nil
}

func (*queryDriver) ChannelEnd(channelID int) error {
	return nil
}

func (*queryDriver) Stats(api.ScannerStats) error {
	return nil
}
