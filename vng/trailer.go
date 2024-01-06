package vng

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
)

const (
	Version     = 4
	HeaderSize  = 16
	MaxMetaSize = 100 * 1024 * 1024
	MaxDataSize = 2 * 1024 * 1024 * 1024
)

type Header struct {
	Version  uint32
	MetaSize uint32
	DataSize uint32
}

func (h Header) Serialize() []byte {
	var bytes [HeaderSize]byte
	bytes[0] = 'V'
	bytes[1] = 'N'
	bytes[2] = 'G'
	binary.LittleEndian.PutUint32(bytes[4:], h.Version)
	binary.LittleEndian.PutUint32(bytes[8:], h.MetaSize)
	binary.LittleEndian.PutUint32(bytes[12:], h.DataSize)
	return bytes[:]
}

func (h *Header) Deserialize(bytes []byte) error {
	if len(bytes) != HeaderSize || bytes[0] != 'V' || bytes[1] != 'N' || bytes[2] != 'G' || bytes[3] != 0 {
		return errors.New("invalid VNG header")
	}
	h.Version = binary.LittleEndian.Uint32(bytes[4:])
	h.MetaSize = binary.LittleEndian.Uint32(bytes[8:])
	h.DataSize = binary.LittleEndian.Uint32(bytes[12:])
	if h.Version != Version {
		return fmt.Errorf("unsupport VNG version %d: expected version %d", h.Version, Version)
	}
	if h.MetaSize > MaxMetaSize {
		return fmt.Errorf("VNG metadata section too big: %d bytes", h.MetaSize)
	}
	if h.MetaSize > MaxDataSize {
		return fmt.Errorf("VNG data section too big: %d bytes", h.DataSize)
	}
	return nil
}

func ReadHeader(r io.Reader) (Header, error) {
	var bytes [HeaderSize]byte
	cc, err := r.Read(bytes[:])
	if err != nil {
		return Header{}, err
	}
	if cc < HeaderSize {
		return Header{}, fmt.Errorf("short VNG file: %d bytes read", cc)
	}
	var h Header
	if err := h.Deserialize(bytes[:]); err != nil {
		return Header{}, err
	}
	return h, nil
}
