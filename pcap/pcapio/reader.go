package pcapio

import (
	"errors"
	"fmt"
	"io"

	"github.com/brimdata/zq/pkg/nano"
	"github.com/brimdata/zq/zio/detector"
	"github.com/google/gopacket/layers"
	"go.uber.org/multierr"
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
	Packet([]byte) ([]byte, nano.Ts, layers.LinkType, error)
	Offset() uint64
}

// NewReader returns a Reader by trying both pcap and pcap-ng formats.
func NewReader(r io.Reader) (Reader, error) {
	return NewReaderWithWarnings(r, nil)
}

// NewReaderWithWarnings returns a Reader by trying both pcap and pcap-ng formats
// and arranges for warning messages to be sent over the given channel.  Different
// pcap implementations can have out-of-spec peculiarities that can be tolerated
// so we send warnings and try to keep going.
func NewReaderWithWarnings(r io.Reader, warningCh chan<- string) (Reader, error) {
	recorder := detector.NewRecorder(r)
	track := detector.NewTrack(recorder)
	_, err1 := NewPcapReader(track)
	if err1 == nil {
		return NewPcapReader(recorder)
	}
	track.Reset()
	_, err2 := NewNgReader(track)
	if err2 == nil {
		r, err := NewNgReader(recorder)
		r.SetWarningChan(warningCh)
		return r, err
	}
	var pcaperr, ngerr *ErrInvalidPcap
	if errors.As(err1, &pcaperr) && errors.As(err2, &ngerr) {
		err1 = fmt.Errorf("pcap: %w", pcaperr.err)
		err2 = fmt.Errorf("pcapng: %w", ngerr.err)
		err := multierr.Combine(err1, err2)
		return nil, NewErrInvalidPcap(err)
	}
	return nil, multierr.Combine(err1, err2)
}
