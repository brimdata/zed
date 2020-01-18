// Package bzngio provides an API for reading and writing zng values and
// directives in binary zng format.  The Reader and Writer types implement the
// the zbuf.Reader and zbuf.Writer interfaces.  Since these methods
// read and write only zbuf.Records, but the bzng format includes additional
// functionality, other methods are available to read/write zng comments
// and include virtual channel numbers in the stream.  Virtual channels
// provide a way to indicate which output of a flowgraph a result came from
// when a flowgraph computes multiple output channels.  The bzng values in
// this zng value (will be) are "machine format" (encoded in a architecture
// independent binary format).  The vanilla zbuf.Reader and zbuf.Writer
// implementations ignore comments and channels.
package bzngio

import (
	"encoding/binary"
	"errors"
	"io"

	"github.com/mccanne/zq/zng"
)

const (
	TypeDescriptor = iota
	TypeValue
	TypeControl
)

const (
	MachineFlag = 0x80
	TypeMask    = 0x3f
)

var (
	ErrBadHeader = errors.New("malformed bzng header")
)

type header struct {
	typ    int
	id     int
	length int
}

const maxHeaderSize = 1 + 3*binary.MaxVarintLen64
const minHeaderSize = 3

func writeHeader(w io.Writer, typ, id, length int) (int, error) {
	var hdr [maxHeaderSize]byte
	hdr[0] = byte(typ)
	off := 1
	if typ != TypeControl {
		off += binary.PutUvarint(hdr[off:], uint64(id))
	}
	off += binary.PutUvarint(hdr[off:], uint64(length))
	_, err := w.Write(hdr[:off])
	return off, err
}

func parseHeader(b []byte, h *header) (int, error) {
	if len(b) < 3 {
		return 0, ErrBadHeader
	}
	typ := int(b[0])
	off := 1
	if typ&MachineFlag != 0 {
		return 0, errors.New("machine-format bzng not yet implemented")
	}
	typ &= TypeMask
	h.typ = typ
	if typ != TypeControl {
		id, n := binary.Uvarint(b[off:])
		if n <= 0 {
			return 0, ErrBadHeader
		}
		// XXX this will go away with the update to the ZNG spec
		const MaxDescriptor = 1000000
		if id > MaxDescriptor {
			return 0, zng.ErrDescriptorInvalid
		}
		off += n
		h.id = int(id)
	}
	length, n := binary.Uvarint(b[off:])
	if n <= 0 {
		return 0, ErrBadHeader
	}
	off += n
	h.length = int(length)
	return off, nil
}
