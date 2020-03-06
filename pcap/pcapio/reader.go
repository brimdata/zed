package pcapio

import (
	"github.com/brimsec/zq/pkg/nano"
	"github.com/google/gopacket/layers"
)

type BlockType int

const (
	TypePacket = iota
	TypeSection
	TypeInterface
)

type Info struct {
	Type BlockType
	Link layers.LinkType
	Ts   nano.Ts
}

// Reader is an interface for reading data blocks from a pcap, either a legacy
// pcap or a next-gen pcap.  The Read method returns blocks of data that are
// one of: a pcap file header (TypeSection), a pcap packet including the capture
// header (TypePacket), a pcap-ng section block (TypeSection), a pcap-ng
// interface block (TypeInterface), or a pcap-ng packet block (TypePacket).
// For TypePacket, the capture timestamp and the link-layer type of the packet
// is indicated in the Info return value.
type Reader interface {
	Read() ([]byte, Info, error)
	Offset() uint64
}
