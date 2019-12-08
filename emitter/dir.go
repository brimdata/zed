package emitter

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"

	"github.com/mccanne/zq/pkg/zio"
	"github.com/mccanne/zq/pkg/zio/textio"
	"github.com/mccanne/zq/pkg/zq"
)

var (
	ErrNoPath = errors.New("no _path field in zson record")
)

// Dir implements the Writer interface and sends all log lines with the
// same descriptor to a file named <prefix><path>.<ext> in the directory indicated,
// where <prefix> and <ext> are specificied and <path> is determined by the
// _path field in the boom descriptor.  Note that more than one descriptor
// can map to the same output file.
type Dir struct {
	dir     string
	prefix  string
	ext     string
	format  string
	stderr  io.Writer // XXX use warnings channel
	tc      *textio.Config
	writers map[*zq.Descriptor]*zio.Writer
	paths   map[string]*zio.Writer
}

func unknownFormat(format string) error {
	return fmt.Errorf("unknown output format: %s", format)
}

func NewDir(dir, prefix, format string, stderr io.Writer, tc *textio.Config) (*Dir, error) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}
	e := zio.Extension(format)
	if e == "" {
		return nil, unknownFormat(format)
	}
	return &Dir{
		dir:     dir,
		prefix:  prefix,
		ext:     e,
		format:  format,
		stderr:  stderr,
		tc:      tc,
		writers: make(map[*zq.Descriptor]*zio.Writer),
		paths:   make(map[string]*zio.Writer),
	}, nil
}

func (d *Dir) Write(r *zq.Record) error {
	out, err := d.lookupOutput(r)
	if err != nil {
		return err
	}
	return out.Write(r)
}

func (d *Dir) lookupOutput(rec *zq.Record) (*zio.Writer, error) {
	descriptor := rec.Descriptor
	w, ok := d.writers[descriptor]
	if ok {
		return w, nil
	}
	w, err := d.newFile(rec)
	if err != nil {
		return nil, err
	}
	d.writers[descriptor] = w
	return w, nil
}

// filename returns the name of the file for the specified path. This handles
// the case of two tds one _path, adding a # in the filename for every _path that
// has more than one td.
func (d *Dir) filename(r *zq.Record) (string, string) {
	colno, ok := r.Descriptor.ColumnOfField("_path")
	var base, path string
	if ok {
		base = string(r.Slice(colno))
		path = base
	} else {
		base = strconv.Itoa(r.Descriptor.ID)
	}
	name := d.prefix + base + d.ext
	return filepath.Join(d.dir, name), path
}

func (d *Dir) newFile(rec *zq.Record) (*zio.Writer, error) {
	filename, path := d.filename(rec)
	if w, ok := d.paths[path]; ok {
		return w, nil
	}
	w, err := NewFile(filename, d.format, d.tc)
	if err != nil {
		return nil, err
	}
	if path != "" {
		d.paths[path] = w
	}
	return w, err
}

func (d *Dir) Close() error {
	var cerr error
	for _, w := range d.writers {
		if err := w.Close(); err != nil {
			cerr = err
		}
	}
	return cerr
}
