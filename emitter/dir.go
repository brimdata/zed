package emitter

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strconv"

	"github.com/brimsec/zq/pkg/iosrc"
	"github.com/brimsec/zq/zbuf"
	"github.com/brimsec/zq/zio"
	"github.com/brimsec/zq/zng"
)

var (
	ErrNoPath = errors.New("no _path field in zng record")
)

type Emitter interface {
	zbuf.WriteCloser
	// ToStdio returns true if the emitter is writing to stdout or stderr.
	ToStdio() bool
}

// dir implements the Writer interface and sends all log lines with the
// same descriptor to a file named <prefix><path>.<ext> in the directory indicated,
// where <prefix> and <ext> are specificied and <path> is determined by the
// _path field in the boom descriptor.  Note that more than one descriptor
// can map to the same output file.
type dir struct {
	ctx     context.Context
	dir     iosrc.URI
	prefix  string
	ext     string
	stderr  io.Writer // XXX use warnings channel
	opts    zio.WriterOpts
	writers map[zng.Type]zbuf.WriteCloser
	paths   map[string]zbuf.WriteCloser
	source  iosrc.Source
}

func NewDir(ctx context.Context, dir, prefix string, stderr io.Writer, opts zio.WriterOpts) (Emitter, error) {
	uri, err := iosrc.ParseURI(dir)
	if err != nil {
		return nil, err
	}
	src, err := iosrc.GetSource(uri)
	if err != nil {
		return nil, err
	}
	return NewDirWithSource(ctx, uri, prefix, stderr, opts, src)
}

func NewDirWithSource(ctx context.Context, d iosrc.URI, prefix string, stderr io.Writer, opts zio.WriterOpts, source iosrc.Source) (Emitter, error) {
	if err := iosrc.MkdirAll(d, 0755); err != nil {
		return nil, err
	}
	e := zio.Extension(opts.Format)
	if e == "" {
		return nil, fmt.Errorf("unknown format: %s", opts.Format)
	}
	return &dir{
		ctx:     ctx,
		dir:     d,
		prefix:  prefix,
		ext:     e,
		stderr:  stderr,
		opts:    opts,
		writers: make(map[zng.Type]zbuf.WriteCloser),
		paths:   make(map[string]zbuf.WriteCloser),
		source:  source,
	}, nil
}

func (d *dir) Write(r *zng.Record) error {
	out, err := d.lookupOutput(r)
	if err != nil {
		return err
	}
	return out.Write(r)
}

func (d *dir) lookupOutput(rec *zng.Record) (zbuf.WriteCloser, error) {
	typ := rec.Type
	w, ok := d.writers[typ]
	if ok {
		return w, nil
	}
	w, err := d.newFile(rec)
	if err != nil {
		return nil, err
	}
	d.writers[typ] = w
	return w, nil
}

func (d *dir) ToStdio() bool {
	return false
}

// filename returns the name of the file for the specified path. This handles
// the case of two tds one _path, adding a # in the filename for every _path that
// has more than one td.
func (d *dir) filename(r *zng.Record) (iosrc.URI, string) {
	var _path string
	base, err := r.AccessString("_path")
	if err == nil {
		_path = base
	} else {
		base = strconv.Itoa(r.Type.ID())
	}
	name := d.prefix + base + d.ext
	return d.dir.AppendPath(name), _path
}

func (d *dir) newFile(rec *zng.Record) (zbuf.WriteCloser, error) {
	filename, path := d.filename(rec)
	if w, ok := d.paths[path]; ok {
		return w, nil
	}
	w, err := NewFileWithSource(d.ctx, filename, d.opts, d.source)
	if err != nil {
		return nil, err
	}
	if path != "" {
		d.paths[path] = w
	}
	return w, err
}

func (d *dir) Close() error {
	var cerr error
	for _, w := range d.writers {
		if err := w.Close(); err != nil {
			cerr = err
		}
	}
	return cerr
}
