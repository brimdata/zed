package emitter

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/pkg/storage"
	"github.com/brimdata/zed/zio"
	"github.com/brimdata/zed/zio/anyio"
)

var (
	ErrNoPath = errors.New("no _path field in zng record")
)

type Split struct {
	ctx     context.Context
	dir     *storage.URI
	prefix  string
	ext     string
	opts    anyio.WriterOpts
	writers map[zed.Type]zio.WriteCloser
	seen    map[string]struct{}
	engine  storage.Engine
}

var _ zio.Writer = (*Split)(nil)

func NewSplit(ctx context.Context, engine storage.Engine, dir *storage.URI, prefix string, opts anyio.WriterOpts) (*Split, error) {
	e := zio.Extension(opts.Format)
	if e == "" {
		return nil, fmt.Errorf("unknown format: %s", opts.Format)
	}
	if prefix != "" {
		prefix = prefix + "-"
	}
	return &Split{
		ctx:     ctx,
		dir:     dir,
		prefix:  prefix,
		ext:     e,
		opts:    opts,
		writers: make(map[zed.Type]zio.WriteCloser),
		seen:    make(map[string]struct{}),
		engine:  engine,
	}, nil
}

func (s *Split) Write(r *zed.Value) error {
	out, err := s.lookupOutput(r)
	if err != nil {
		return err
	}
	return out.Write(r)
}

func (s *Split) lookupOutput(val *zed.Value) (zio.WriteCloser, error) {
	typ := val.Type
	w, ok := s.writers[typ]
	if ok {
		return w, nil
	}
	w, err := NewFileFromURI(s.ctx, s.engine, s.path(val), s.opts)
	if err != nil {
		return nil, err
	}
	s.writers[typ] = w
	return w, nil
}

// path returns the storage URI given the prefix combined with a type ID
// to make a unique path for each Zed type. If the _path field is present,
// we use that for the unique ID but add the type ID if there any _path
// string appears with different Zed types.
func (s *Split) path(r *zed.Value) *storage.URI {
	uniq := strconv.Itoa(r.Type.ID())
	if _path, err := r.AccessString("_path"); err == nil {
		if _, ok := s.seen[_path]; ok {
			uniq = _path + "-" + uniq
		} else {
			uniq = _path
			s.seen[_path] = struct{}{}
		}
	}
	return s.dir.AppendPath(s.prefix + uniq + s.ext)
}

func (s *Split) Close() error {
	var cerr error
	for _, w := range s.writers {
		if err := w.Close(); err != nil {
			cerr = err
		}
	}
	return cerr
}
