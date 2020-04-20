package detector

import (
	"errors"
	"os"

	"github.com/brimsec/zq/zbuf"
	"github.com/brimsec/zq/zio"
	"github.com/brimsec/zq/zng/resolver"
)

// OpenFile creates and returns zbuf.File for the indicated path.  If the path is
// a directory or can't otherwise be open as a file, then an error is returned.
// If path is empty, then os.Stdin is used as the file.
func OpenFile(zctx *resolver.Context, path string, flags *zio.ReaderFlags) (*zbuf.File, error) {
	var f *os.File
	var err error
	if path == "" {
		f = os.Stdin
	} else {
		info, err := os.Stat(path)
		if err != nil {
			return nil, err
		}
		if info.IsDir() {
			return nil, errors.New("is a directory")
		}
		f, err = os.Open(path)
		if err != nil {
			return nil, err
		}
	}
	r := GzipReader(f)
	var zr zbuf.Reader
	if flags.Format == "auto" {
		zr, err = NewReader(r, zctx)
	} else {
		zr, err = LookupReader(r, zctx, flags)
	}
	if err != nil {
		return nil, err
	}
	return zbuf.NewFile(zr, f), nil
}
