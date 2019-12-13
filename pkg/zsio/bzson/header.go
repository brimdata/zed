// Package bzson provides an API for reading and writing zson values and
// directives in binary zson format.  The Reader and Writer types implement the
// the zson.Reader and zson.Writer interfaces.  Since these methods
// read and write only zson.Records, but the bzson format includes additional
// functionality, other methods are available to read/write zson comments
// and include virtual channel numbers in the stream.  Virtual channels
// provide a way to indicate which output of a flowgraph a result came from
// when a flowgraph computes multiple output channels.  The bzson values in
// this zson value (will be) are "machine format" (encoded in a architecture
// independent binary format).  The vanilla zson.Reader and zson.Writer
// implementations ignore comments and channels.
package bzson

import (
	"encoding/binary"
	"errors"
	"io"

	"github.com/mccanne/zq/pkg/zson"
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
	ErrBadHeader = errors.New("malformed bzson header")
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
		return 0, errors.New("machine-format bzson not yet implemented")
	}
	typ &= TypeMask
	h.typ = typ
	if typ != TypeControl {
		id, n := binary.Uvarint(b[off:])
		if n <= 0 {
			return 0, ErrBadHeader
		}
		if id > zson.MaxDescriptor {
			return 0, zson.ErrDescriptorInvalid
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
