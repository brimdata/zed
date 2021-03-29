// The code in this source file is derived from
// https://github.com/google/gopacket/blob/master/pcapgo/read.go
// as of February 2020 and is covered by the copyright below.
// The changes are covered by the copyright and license in the
// LICENSE file in the root directory of this repository.

// Copyright 2014 Damjan Cvetko. All rights reserved.
// See acknowledgments.txt for full license text from:
// https://github.com/fitzgen/glob-to-regexp#license

package pcapio

import (
	"encoding/binary"
	"fmt"
	"io"

	"github.com/brimdata/zq/pkg/nano"
	"github.com/brimdata/zq/pkg/peeker"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
)

// PcapReader implements the Reader interface to read packet data in PCAP
// format.  See http://wiki.wireshark.org/Development/LibpcapFileFormat
// for information on the file format.
//
// We currenty read v2.4 file format with nanosecond and microsecond
// timestamp resolution in little-endian and big-endian encoding.
//
// If the PCAP data is gzip compressed it is transparently uncompressed
// by wrapping the given io.Reader with a gzip.Reader.
type PcapReader struct {
	*peeker.Reader
	LinkType layers.LinkType

	byteOrder      binary.ByteOrder
	nanoSecsFactor uint32
	// timezone
	// sigfigs
	versionMajor uint16
	versionMinor uint16
	snaplen      uint32
	offset       uint64
	header       []byte
}

//XXX from ngwrite
const magicMicroseconds = 0xA1B2C3D4
const versionMajor = 2
const versionMinor = 4

const magicNanoseconds = 0xA1B23C4D
const magicMicrosecondsBigendian = 0xD4C3B2A1
const magicNanosecondsBigendian = 0x4D3CB2A1

const magicGzip1 = 0x1f
const magicGzip2 = 0x8b

const (
	fileHeaderLen   = 24
	packetHeaderLen = 16
)

// NewPcapReader returns a new reader object, for reading packet data from
// the given reader. The reader must be open and header data is
// read from it at this point.
// If the file format is not supported an error is returned.
//
//  // Create new reader:
//  f, _ := fs.Open("/tmp/file.pcap")
//  defer f.Close()
//  r, err := NewReader(f)
//  data, info, err := r.Read()
func NewPcapReader(r io.Reader) (*PcapReader, error) {
	reader := &PcapReader{
		Reader: peeker.NewReader(r, 32*1024, 1024*1024),
	}
	if err := reader.readHeader(); err != nil {
		return nil, err
	}
	return reader, nil
}

func (r *PcapReader) Packet(block []byte) ([]byte, nano.Ts, layers.LinkType, error) {
	if len(block) <= packetHeaderLen {
		return nil, 0, 0, errInvalidf("packet buffer length less than minimum packet size")
	}
	caplen := int(r.byteOrder.Uint32(block[8:12]))
	if caplen+packetHeaderLen > len(block) {
		return nil, 0, 0, errInvalidf("invalid capture length")
	}
	ts := r.TsFromHeader(block)
	pkt := block[packetHeaderLen:]
	return pkt[:caplen], ts, r.LinkType, nil
}

func (r *PcapReader) readHeader() error {
	hdr, err := r.Reader.Read(fileHeaderLen)
	if err != nil {
		return err
	}
	r.header = make([]byte, fileHeaderLen)
	copy(r.header, hdr)
	if magic := binary.LittleEndian.Uint32(hdr[0:4]); magic == magicNanoseconds {
		r.byteOrder = binary.LittleEndian
		r.nanoSecsFactor = 1
	} else if magic == magicNanosecondsBigendian {
		r.byteOrder = binary.BigEndian
		r.nanoSecsFactor = 1
	} else if magic == magicMicroseconds {
		r.byteOrder = binary.LittleEndian
		r.nanoSecsFactor = 1000
	} else if magic == magicMicrosecondsBigendian {
		r.byteOrder = binary.BigEndian
		r.nanoSecsFactor = 1000
	} else {
		return errInvalidf("unknown magic %x", magic)
	}
	if r.versionMajor = r.byteOrder.Uint16(hdr[4:6]); r.versionMajor != versionMajor {
		return errInvalidf("unknown major version %d", r.versionMajor)
	}
	if r.versionMinor = r.byteOrder.Uint16(hdr[6:8]); r.versionMinor != versionMinor {
		return errInvalidf("unknown minor version %d", r.versionMinor)
	}
	// ignore timezone 8:12 and sigfigs 12:16
	r.snaplen = r.byteOrder.Uint32(hdr[16:20])
	r.LinkType = layers.LinkType(r.byteOrder.Uint32(hdr[20:24]))
	return nil
}

func (r *PcapReader) TsFromHeader(hdr []byte) nano.Ts {
	ns := int64(r.byteOrder.Uint32(hdr[0:4])) * 1_000_000_000
	ns += int64(r.byteOrder.Uint32(hdr[4:8]) * r.nanoSecsFactor)
	return nano.Ts(ns)
}

func (r *PcapReader) Read() ([]byte, BlockType, error) {
	header := r.header
	if header != nil {
		r.header = nil
		r.offset += uint64(len(header))
		return header, TypeSection, nil
	}
	hdr, err := r.Reader.Peek(packetHeaderLen)
	if err != nil {
		if err == io.EOF {
			err = nil
		}
		return nil, 0, err
	}
	caplen := int(r.byteOrder.Uint32(hdr[8:12]))
	if r.snaplen != 0 && caplen > int(r.snaplen) {
		return nil, 0, errInvalidf("capture length exceeds snap length: %d > %d", caplen, r.snaplen)
	}
	// Some pcaps have the bug that captures exceed the size of the actual packet.
	// Wireshark seems to handle these ok, so instead of failing and raising
	// an error, we should ignore this condition and pass the packet along.
	//fullLength := int(r.byteOrder.Uint32(hdr[12:16]))
	//if caplen > fullLength {
	//	return nil, 0, fmt.Errorf("capture length exceeds original packet length: %d > %d", caplen, fullLength)
	//}
	n := caplen + packetHeaderLen
	block, err := r.Reader.Read(n)
	if err != nil {
		return nil, 0, err
	}
	r.offset += uint64(n)
	return block, TypePacket, nil
}

// Snaplen returns the snapshot length of the capture file.
func (r *PcapReader) Snaplen() uint32 {
	return r.snaplen
}

func (r *PcapReader) Offset() uint64 {
	return r.offset
}

func (r *PcapReader) Version() string {
	return fmt.Sprintf("%d.%d", r.versionMajor, r.versionMinor)
}

// SetSnaplen sets the snapshot length of the capture file.
//
// This is useful when a pcap file contains packets bigger than then snaplen.
// Pcapgo will error when reading packets bigger than snaplen, then it dumps those
// packets and reads the next 16 bytes, which are part of the "faulty" packet's payload, but pcapgo
// thinks it's the next header, which is probably also faulty because it's not really a packet header.
// This can lead to a lot of faulty reads.
//
// The SetSnaplen function can be used to set a bigger snaplen to prevent those read errors.
//
// This snaplen situation can happen when a pcap writer doesn't truncate packets to the snaplen size while writing packets to file.
// E.g. In Python, dpkt.pcap.Writer sets snaplen by default to 1500 (https://dpkt.readthedocs.io/en/latest/api/api_auto.html#dpkt.pcap.Writer)
// but doesn't enforce this when writing packets (https://dpkt.readthedocs.io/en/latest/_modules/dpkt/pcap.html#Writer.writepkt).
// When reading, tools like tcpdump, tcpslice, mergecap and wireshark ignore the snaplen and use
// their own defined snaplen.
// E.g. When reading packets, tcpdump defines MAXIMUM_SNAPLEN (https://github.com/the-tcpdump-group/tcpdump/blob/6e80fcdbe9c41366df3fa244ffe4ac8cce2ab597/netdissect.h#L290)
// and uses it (https://github.com/the-tcpdump-group/tcpdump/blob/66384fa15b04b47ad08c063d4728df3b9c1c0677/print.c#L343-L358).
//
// For further reading:
//  - https://github.com/the-tcpdump-group/tcpdump/issues/389
//  - https://bugs.wireshark.org/bugzilla/show_bug.cgi?id=8808
//  - https://www.wireshark.org/lists/wireshark-dev/201307/msg00061.html
//  - https://github.com/wireshark/wireshark/blob/bfd51199e707c1d5c28732be34b44a9ee8a91cd8/wiretap/pcap-common.c#L723-L742
//    - https://github.com/wireshark/wireshark/blob/f07fb6cdfc0904905627707b88450054e921f092/wiretap/libpcap.c#L592-L598
//    - https://github.com/wireshark/wireshark/blob/f07fb6cdfc0904905627707b88450054e921f092/wiretap/libpcap.c#L714-L727
//  - https://github.com/the-tcpdump-group/tcpdump/commit/d033c1bc381c76d13e4aface97a4f4ec8c3beca2
//  - https://github.com/the-tcpdump-group/tcpdump/blob/88e87cb2cb74c5f939792171379acd9e0efd8b9a/netdissect.h#L263-L290
func (r *PcapReader) SetSnaplen(newSnaplen uint32) {
	r.snaplen = newSnaplen
}

// Reader formatter
func (r *PcapReader) String() string {
	return fmt.Sprintf("PcapFile  maj: %x min: %x snaplen: %d linktype: %s", r.versionMajor, r.versionMinor, r.snaplen, r.LinkType)
}

// Resolution returns the timestamp resolution of acquired timestamps before scaling to NanosecondTimestampResolution.
func (r *PcapReader) Resolution() gopacket.TimestampResolution {
	if r.nanoSecsFactor == 1 {
		return gopacket.TimestampResolutionMicrosecond
	}
	return gopacket.TimestampResolutionNanosecond
}
