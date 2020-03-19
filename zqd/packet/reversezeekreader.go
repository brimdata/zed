package packet

import (
	"bytes"
	"errors"
	"os"

	"github.com/brimsec/zq/zio/zeekio"
	"github.com/brimsec/zq/zng"
	"github.com/brimsec/zq/zng/resolver"
)

// reverseZeekReader reads a Zeek file's records in reverse order.
type reverseZeekReader struct {
	file   *os.File
	buf    []byte
	off    int64
	parser *zeekio.Parser
}

func newReverseZeekReader(f *os.File, zctx *resolver.Context) (*reverseZeekReader, error) {
	info, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if _, err := f.Seek(0, 0); err != nil {
		return nil, err
	}
	r, err := zeekio.NewReader(f, zctx)
	if err != nil {
		return nil, err
	}
	// Read one record to set the Zeek parser's descriptor.
	if _, err := r.Read(); err != nil {
		return nil, err
	}
	return &reverseZeekReader{
		file:   f,
		off:    info.Size(),
		parser: r.Parser(),
	}, nil
}

func (r *reverseZeekReader) Close() error {
	return r.file.Close()
}

func (r *reverseZeekReader) Read() (*zng.Record, error) {
	for {
		line, err := r.readLine()
		if err != nil || line == nil {
			return nil, err
		}
		if line[0] == '#' {
			continue
		}
		return r.parser.ParseValue(line)
	}
}

// readLine returns a single line without terminal newline.  Lines are returned
// from last to first.  The returned buffer is valid only until the next call.
// When lines remain, readLine returns a nil line and nil error.
func (r *reverseZeekReader) readLine() ([]byte, error) {
	for {
		// Skip blank lines and remove terminal newline.
		r.buf = bytes.TrimRight(r.buf, "\n")
		if i := bytes.LastIndexByte(r.buf, '\n'); i > 0 {
			line := r.buf[i+1:]
			r.buf = r.buf[:i]
			return line, nil
		}
		if r.off == 0 {
			if len(r.buf) == 0 {
				return nil, nil
			}
			line := r.buf
			r.buf = nil
			return line, nil
		}
		const readSize = 4096
		tmp := make([]byte, readSize)
		if r.off < int64(len(tmp)) {
			tmp = tmp[:r.off]
		}
		r.off -= int64(len(tmp))
		n, err := r.file.ReadAt(tmp, r.off)
		if err != nil {
			return nil, err
		}
		if n < len(tmp) {
			return nil, errors.New("short read")
		}
		r.buf = append(tmp, r.buf...)
	}
}
