package emitter

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/mccanne/zq/pkg/zsio"
	"github.com/mccanne/zq/pkg/zsio/text"
	"github.com/mccanne/zq/pkg/zson"
)

var (
	ErrNoPath = errors.New("no _path field in zson record")
)

// Dir implements the Writer interface and sends all log lines with the
// same descriptor to a file named <prefix><path>.<ext> in the directory indicated,
// where <prefix> and <ext> are specificied and <path> is determined by the
// _path field in the boom descriptor.  If more than one path exists with
// different descriptors, the later records are ignored.
type Dir struct {
	dir     string
	prefix  string
	ext     string
	format  string
	stderr  io.Writer // XXX use warnings channel
	tc      *text.Config
	writers map[*zson.Descriptor]*zsio.Writer
	paths   map[string]int
}

func NewDir(dir, prefix, format string, stderr io.Writer, tc *text.Config) (*Dir, error) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}
	e := zsio.Extension(format)
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
		writers: make(map[*zson.Descriptor]*zsio.Writer),
		paths:   make(map[string]int),
	}, nil
}

func (d *Dir) Write(r *zson.Record) error {
	out, err := d.lookupOutput(r)
	if err != nil {
		return err
	}
	if out != nil {
		return out.Write(r)
	}
	// The descriptor is blocked if file exists.  Drop the record.
	return nil
}

func (d *Dir) lookupOutput(rec *zson.Record) (*zsio.Writer, error) {
	descriptor := rec.Descriptor
	w, ok := d.writers[descriptor]
	if ok {
		return w, nil
	}
	w, _, err := d.newFile(rec)
	if err == ErrNoPath {
		fmt.Fprintf(d.stderr, "warning: no path for descriptor %d (dropping all related records)", rec.Descriptor.ID)
		// Block this descriptor.
		d.writers[descriptor] = nil
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	d.writers[descriptor] = w
	return w, nil
}

// filename returns the name of the file for the specified path. This handles
// the case of two tds one _path, adding a # in the filename for every _path that
// has more than one td.
func (d *Dir) filename(path string) string {
	filename := d.prefix + path
	if d.paths[path]++; d.paths[path] > 1 {
		filename += fmt.Sprintf("-%d", d.paths[path])
	}
	filename += d.ext
	return filepath.Join(d.dir, filename)
}

func (d *Dir) newFile(rec *zson.Record) (*zsio.Writer, string, error) {
	// get path name from descriptor.  the td at column 0
	// has already been stripped out.
	i, ok := rec.Descriptor.ColumnOfField("_path")
	if !ok {
		return nil, "", ErrNoPath
	}
	path := string(rec.Slice(i))
	filename := d.filename(path)
	w, err := NewFile(filename, d.format, d.tc)
	if err != nil {
		return nil, filename, err
	}
	return w, filename, err
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
