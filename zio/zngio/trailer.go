package zngio

import (
	"encoding/binary"
	"errors"
)

var ErrTrailerNotFound = errors.New("trailer not found")

// FindTrailer finds the last valid, EOS-terminated ZNG stream in the
// buffer provided.
func FindTrailer(b []byte) ([]byte, error) {
	off := len(b) - 1
	if off < 0 {
		return nil, ErrTrailerNotFound
	}
	if b[off] != EOS {
		return nil, errors.New("trailer doesn't end with EOS")
	}
	for {
		off--
		if off < 0 {
			return nil, ErrTrailerNotFound
		}
		if off == 0 || b[off-1] == EOS {
			if ok := validStream(b, off); ok {
				return b[off:], nil
			}
		}
	}
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
