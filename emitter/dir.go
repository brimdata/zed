package emitter

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

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
	stderr  io.Writer
	writers map[*zson.Descriptor]*zson.WriteCloser
	paths   map[string]int
}

func NewDir(dir, prefix, ext string, stderr io.Writer) (*Dir, error) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}
	return &Dir{
		dir:     dir,
		prefix:  prefix,
		ext:     ext,
		stderr:  stderr,
		writers: make(map[*zson.Descriptor]*zson.WriteCloser),
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
	// this descriptor is blocked so drop the record
	return nil
}

func (d *Dir) lookupOutput(rec *zson.Record) (*zson.WriteCloser, error) {
	descriptor := rec.Descriptor
	w, ok := d.writers[descriptor]
	if ok {
		if w == nil {
			// avoid non-nil interface pointer to nil
			return nil, nil
		}
		return w, nil
	}
	w, fname, err := d.newFile(rec)
	if os.IsNotExist(err) {
		//XXX fix Phil's error message
		// warn that file exists
		warning := fmt.Sprintf("%s: file exists (blocking writes to this file, but continuing)\n", fname)
		d.stderr.Write([]byte(warning))
		// block future writes to this descriptor since file exists
		// XXX fix this since zeek writing now handles interleaved stuff
		d.writers[descriptor] = nil
	} else if err != nil {
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

func (d *Dir) newFile(rec *zson.Record) (*zson.WriteCloser, string, error) {
	// get path name from descriptor.  the td at column 0
	// has already been stripped out.
	i, ok := rec.Descriptor.ColumnOfField("_path")
	if !ok {
		return nil, "", ErrNoPath
	}
	path := string(rec.Slice(i))
	filename := d.filename(path)
	w, err := OpenOutputFile("zeek", filename)
	if err == os.ErrExist {
		// The files exists and we're not going to overwrite it.
		// Return nil for the file with no error indicating that
		// we should track this path but block all such records.
		return nil, filename, nil
	}
	return w, filename, nil
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
