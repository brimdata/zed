package zngio

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/zson"
)

var ErrTrailerNotFound = errors.New("trailer not found")

const (
	Magic          = "ZNG Trailer"
	TrailerMaxSize = 4096
)

type Trailer struct {
	Magic    string    `zed:"magic"`
	Type     string    `zed:"type"`
	Version  int       `zed:"version"`
	Sections []int64   `zed:"sections"`
	Meta     zed.Value `zed:"meta"`
}

func MarshalTrailer(typ string, version int, sections []int64, meta interface{}) (zed.Value, error) {
	m := zson.NewZNGMarshaler()
	m.Decorate(zson.StylePackage)
	metaVal, err := m.Marshal(meta)
	if err != nil {
		return zed.Value{}, err
	}
	return m.Marshal(&Trailer{
		Magic:    Magic,
		Type:     typ,
		Version:  version,
		Sections: sections,
		Meta:     metaVal,
	})
}

func ReadTrailer(r io.ReaderAt, fileSize int64) (*Trailer, error) {
	b, err := readTail(r, fileSize)
	if err != nil {
		return nil, err
	}
	trailer, _, err := findTrailer(b)
	return trailer, err
}

func ReadTrailerAsBytes(r io.ReaderAt, fileSize int64) ([]byte, error) {
	b, err := readTail(r, fileSize)
	if err != nil {
		return nil, err
	}
	_, bytes, err := findTrailer(b)
	return bytes, err
}

func readTail(r io.ReaderAt, fileSize int64) ([]byte, error) {
	n := fileSize
	if n > TrailerMaxSize {
		n = TrailerMaxSize
	}
	buf := make([]byte, n)
	cc, err := r.ReadAt(buf, fileSize-n)
	if err != nil {
		return nil, err
	}
	if int64(cc) != n {
		// This shouldn't happen but maybe could occur under a corner case
		// or I/O problems.
		return nil, fmt.Errorf("couldn't read trailer: expected %d bytes but read %d", n, cc)
	}
	return buf, nil
}

// FindTrailer finds the last valid, EOS-terminated ZNG stream in the
// buffer provided.
func findTrailer(b []byte) (*Trailer, []byte, error) {
	u := zson.NewZNGUnmarshaler()
	err := ErrTrailerNotFound
	off := len(b) - 1
	for {
		off = findCandidate(b, off)
		if off < 0 {
			return nil, nil, err
		}
		if val := decodeTrailer(b[off:]); val != nil {
			var trailer Trailer
			uErr := u.Unmarshal(*val, &trailer)
			if uErr == nil {
				if trailer.Magic != Magic {
					return nil, nil, errors.New("bad trailer magic")
				}
				return &trailer, b[off:], nil
			}
			// If unmarshal fails, keep looking for candidates but
			// remember the error if we never succeed as we prefer this
			// more specific unmarshaling error over ErrTrailerNotFound.
			if err == ErrTrailerNotFound {
				err = uErr
			}
		}
	}
}

func findCandidate(b []byte, off int) int {
	for {
		off--
		if off < 0 {
			return -1
		}
		if off == 0 || b[off-1] == EOS {
			if ok := validStream(b, off); ok {
				return off
			}
		}
	}
}
func decodeTrailer(b []byte) *zed.Value {
	val, _ := NewReader(bytes.NewReader(b), zed.NewContext()).Read()
	return val
}

func validStream(b []byte, off int) bool {
	for off < len(b) {
		code := b[off]
		if code == EOS {
			return true
		}
		if (code & 0x80) != 0 {
			// Bad format
			return false
		}
		typ := (code >> 4) & 3
		if typ == 3 {
			// bad message block type
			return false
		}
		len, ok := decodeLength(b[off:], code)
		if !ok {
			return false
		}
		off += len
	}
	return false
}

func decodeLength(b []byte, code byte) (int, bool) {
	if len(b) < 2 {
		return 0, false
	}
	v, n := binary.Uvarint(b[1:])
	if n == 0 {
		return 0, false
	}
	return ((int(v) << 4) | (int(code) & 0xf)) + n + 1, true
}
