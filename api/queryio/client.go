package queryio

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/api"
	"github.com/brimdata/zed/zbuf"
	"github.com/brimdata/zed/zio/zngio"
	"github.com/brimdata/zed/zson"
)

type scanner struct {
	channel  string
	scanner  zbuf.Scanner
	closer   io.Closer
	progress zbuf.Progress
}

func NewScanner(ctx context.Context, rc io.ReadCloser) zbuf.Scanner {
	s, err := zngio.NewReader(zed.NewContext(), rc).NewScanner(ctx, nil)
	if err != nil {
		// XXX This shouldn't happen since we don't have a filter.
		panic(err)
	}
	return &scanner{
		scanner: s,
		closer:  rc,
	}
}

func (s *scanner) Progress() zbuf.Progress {
	return s.progress
}

func (s *scanner) Pull(done bool) (zbuf.Batch, error) {
again:
	batch, err := s.scanner.Pull(done)
	if err == nil {
		if batch != nil {
			return zbuf.Label(s.channel, batch), nil
		}
		return nil, s.closer.Close()
	}
	zctrl, ok := err.(*zbuf.Control)
	if !ok {
		return nil, err
	}
	v, err := s.marshalControl(zctrl)
	if err != nil {
		return nil, err
	}
	switch ctrl := v.(type) {
	case *api.QueryChannelSet:
		s.channel = ctrl.Channel
		goto again
	case *api.QueryChannelEnd:
		eoc := zbuf.EndOfChannel(ctrl.Channel)
		return &eoc, nil
	case *api.QueryStats:
		s.progress.Add(ctrl.Progress)
		goto again
	case *api.QueryError:
		return nil, errors.New(ctrl.Error)
	default:
		return nil, fmt.Errorf("unsupported control message: %T", ctrl)
	}
}

func (s *scanner) marshalControl(zctrl *zbuf.Control) (any, error) {
	ctrl, ok := zctrl.Message.(*zngio.Control)
	if !ok {
		return nil, fmt.Errorf("unknown control type: %T", zctrl.Message)
	}
	if ctrl.Format != zngio.ControlFormatZSON {
		return nil, fmt.Errorf("unsupported app encoding: %v", ctrl.Format)
	}
	arena := zed.NewArena()
	defer arena.Unref()
	value, err := zson.ParseValue(zed.NewContext(), arena, string(ctrl.Bytes))
	if err != nil {
		return nil, fmt.Errorf("unable to parse Zed control message: %w (%s)", err, string(ctrl.Bytes))
	}
	var v interface{}
	if err := unmarshaler.Unmarshal(value, &v); err != nil {
		return nil, fmt.Errorf("unable to unmarshal Zed control message: %w (%s)", err, string(ctrl.Bytes))
	}
	return v, nil
}
