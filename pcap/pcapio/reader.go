package pcapio

import (
	"io"

	"github.com/brimsec/zq/pkg/nano"
	"github.com/brimsec/zq/zio/detector"
	"github.com/google/gopacket/layers"
)

type BlockType int

const (
	TypePacket = iota
	TypeSection
	TypeInterface
)

// Reader is an interface for reading data blocks from a pcap, either a legacy
// pcap or a next-gen pcap.  The Read method returns blocks of data that are
// one of: a pcap file header (TypeSection), a pcap packet including the capture
// header (TypePacket), a pcap-ng section block (TypeSection), a pcap-ng
// interface block (TypeInterface), or a pcap-ng packet block (TypePacket).
// For TypePacket, the capture timestamp and the link-layer type of the packet
// is indicated in the Info return value.
type Reader interface {
	Read() ([]byte, BlockType, error)
	Packet([]byte) ([]byte, nano.Ts, layers.LinkType)
	Offset() uint64
}

// NewReader returns a Reader by trying both pcap and pcap-ng formats.
func NewReader(r io.Reader) (Reader, error) {
	recorder := detector.NewRecorder(r)
	track := detector.NewTrack(recorder)
	_, err := NewPcapReader(track)
	if err == nil {
		return NewPcapReader(recorder)
	}
	track.Reset()
	_, err = NewNgReader(track)
	if err == nil {
		return NewNgReader(recorder)
	}
	return nil, ErrCorruptPcap
}
