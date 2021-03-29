// The code in this source file is derived from
// https://github.com/google/gopacket/blob/master/pcapgo/ngread.go
// as of February 2020 and is covered by the copyright below.
// The changes are covered by the copyright and license in the
// LICENSE file in the root directory of this repository.

// Copyright 2018 The GoPacket Authors. All rights reserved.
// See acknowledgments.txt for full license text from:
// https://github.com/google/gopacket/LICENSE

package pcapio

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"time"

	"github.com/brimdata/zq/pkg/nano"
	"github.com/brimdata/zq/pkg/peeker"
	"github.com/google/gopacket/layers"
)

const PacketBlockHeaderLen = 28

// NgReader wraps an underlying bufio.NgReader to read packet data in pcapng.
type NgReader struct {
	*peeker.Reader
	ifaces    []NgInterface
	first     []byte
	bigEndian bool
	offset    uint64
	warningCh chan<- string
}

// NewNgReader initializes a new writer, reads the first section header,
// and if necessary according to the options the first interface.
func NewNgReader(r io.Reader) (*NgReader, error) {
	ret := &NgReader{
		Reader: peeker.NewReader(r, 32*1024, 1024*1024),
	}
	hdr, err := ret.Peek(12)
	if err != nil {
		if err == peeker.ErrTruncated {
			err = errInvalidf("pcap-ng file is too small to be valid")
		}
		return nil, err
	}
	// ensure first block is correct
	typ := ngBlockType(ret.getUint32(hdr[:4]))
	//pcapng _must_ start with a section header
	if typ != ngBlockTypeSectionHeader {
		return nil, errInvalidf("first block type not a section header: %d", typ)
	}
	if err := ret.parseSectionMagic(hdr[8:12]); err != nil {
		return nil, err
	}
	typ, block, err := ret.readBlock()
	if err != nil {
		return nil, err
	}
	if typ != ngBlockTypeSectionHeader {
		return nil, errInvalidf("unknown magic %x", typ)
	}
	ret.first = block
	return ret, nil
}

func (r *NgReader) SetWarningChan(c chan<- string) {
	r.warningCh = c
}

func (r *NgReader) warn(format string, a ...interface{}) {
	if c := r.warningCh; c != nil {
		c <- fmt.Sprintf(format, a...)
	}
}

func (r *NgReader) parsePacket(block []byte) ([]byte, int, error) {
	if len(block) < PacketBlockHeaderLen {
		return nil, 0, errInvalidf("packet buffer length less than minimum packet size")
	}
	ifno := int(r.getUint32(block[8:12]))
	if ifno >= len(r.ifaces) {
		return nil, 0, errInvalidf("packet references unknown interface no: %d", ifno)
	}
	caplen := int(r.getUint32(block[20:24]))
	packet := block[PacketBlockHeaderLen:]
	if len(packet) < caplen {
		return nil, 0, errInvalidf("invalid capture length")
	}
	return packet[:caplen], ifno, nil
}

// Packet returns the captured portion of a packet from an enhanced packet
// block returned by Read() (i.e., with BlockType equal to TypePacket) beginning
// with the link-layer header.  It also extracts the capture timestamp and link
// layer type and returns those values along with the packet.  If an error is
// encountered, zero values are returned for the three values.  PCAP-NG simple
// packet types aren't supported yet (we presume this type of trace is rare and
// does not fit our use case here as these traces do not include capture timestamps
// and the point here is to pull our ranges of packets from a large pcap based
// on timestamp).  We also do not support the original deprecated PCAP-NG packet
// format but could add support if users request this (it would only be because
// old pcaps with this deprecated format are sitting around).
func (r *NgReader) Packet(block []byte) ([]byte, nano.Ts, layers.LinkType, error) {
	packet, ifno, err := r.parsePacket(block)
	if err != nil {
		return nil, 0, 0, err
	}
	ts := uint64(r.getUint32(block[12:16]))<<32 | uint64(r.getUint32(block[16:20]))
	t := time.Unix(r.convertTime(ifno, ts)).UTC()
	return packet, nano.TimeToTs(t), r.ifaces[ifno].LinkType, nil
}

func (r *NgReader) InterfaceDescriptor(block []byte) (NgInterface, error) {
	return r.parseInterfaceDescriptor(block)
}

func (r *NgReader) SectionHeader(block []byte) NgSectionInfo {
	return NgSectionInfo{
		MajorVersion: r.getUint16(block[12:14]),
		MinorVersion: r.getUint16(block[14:16]),
	}
}

func (r *NgReader) Offset() uint64 {
	return r.offset
}

// The following functions make the binary.* functions inlineable (except for getUint64, which is too big, but not in any hot path anyway)
// Compared to storing binary.*Endian in a binary.ByteOrder this shaves off about 20% for (ZeroCopy)ReadPacketData, which is caused by the needed itab lookup + indirect go call
func (r *NgReader) getUint16(buffer []byte) uint16 {
	if r.bigEndian {
		return binary.BigEndian.Uint16(buffer)
	}
	return binary.LittleEndian.Uint16(buffer)
}

func (r *NgReader) getUint32(buffer []byte) uint32 {
	if r.bigEndian {
		return binary.BigEndian.Uint32(buffer)
	}
	return binary.LittleEndian.Uint32(buffer)
}

func (r *NgReader) getUint64(buffer []byte) uint64 {
	if r.bigEndian {
		return binary.BigEndian.Uint64(buffer)
	}
	return binary.LittleEndian.Uint64(buffer)
}

// Now the pcapng implementation

// readBlock reads a the blocktype and length from the file.
// If the type is a section header, endianess is also read.
func (r *NgReader) readBlock() (ngBlockType, []byte, error) {
	hdr, err := r.Peek(12)
	if err != nil {
		if err == peeker.ErrTruncated {
			r.warn("pcap-ng has extra bytes at eof: %s", hex.EncodeToString(hdr))
			// read the bytes to discard and reach eof
			r.Reader.Read(len(hdr))
			err = nil
		}
		return 0, nil, err
	}
	typ := ngBlockType(r.getUint32(hdr[:4]))
	// The first thing we do when reading any block is check for a
	// section header.  If so, we parse it so we get endianess right for
	// the remaining fields (note the section header type is robust to
	// either byte order).
	if typ == ngBlockTypeSectionHeader {
		if err := r.parseSectionMagic(hdr[8:12]); err != nil {
			return 0, nil, err
		}
	}
	length := r.getUint32(hdr[4:8])
	if length < 20 {
		// avoid infinite loop for bad input
		return 0, nil, errInvalidf("pcap-ng block too small: %d bytes", length)
	}
	b, err := r.Reader.Read(int(length))
	if err != nil {
		return 0, nil, err
	}
	if uint32(len(b)) < length {
		return 0, nil, errInvalidf("truncated pcap-ng block")
	}
	if r.getUint32(b[length-4:length]) != uint32(length) {
		return 0, nil, errInvalidf("pcap-ng trailer length mismatch")
	}
	return typ, b, err
}

func (r *NgReader) parseSectionMagic(b []byte) error {
	if binary.BigEndian.Uint32(b) == ngByteOrderMagic {
		r.bigEndian = true
	} else if binary.LittleEndian.Uint32(b) == ngByteOrderMagic {
		r.bigEndian = false
	} else {
		return errInvalidf("Wrong byte order value in Section Header")
	}
	r.ifaces = r.ifaces[:0]
	return nil
}

// parseOption parses and returns a single arbitrary option (type, value, and
// actual length of option buffer) or returns an error if the option is not
// valid.
func (r *NgReader) readOption(b []byte) (ngOptionCode, []byte, int, error) {
	code := ngOptionCode(r.getUint16(b[:2]))
	if code == ngOptionCodeEndOfOptions {
		return code, nil, 4, nil
	}
	length := r.getUint16(b[2:4])
	b = b[4:]
	if int(length) > len(b) {
		return 0, nil, 0, errInvalidf("bad option length")
	}
	// Determine padding. The option value field is always padded up to 32 bits.
	padding := length % 4
	if padding > 0 {
		padding = 4 - padding
	}
	return code, b[:length], 4 + int(length+padding), nil
}

// readInterfaceDescriptor parses an interface descriptor, prepares timing
// calculation, and adds the interface details to the current list.
func (r *NgReader) parseInterfaceDescriptor(b []byte) (NgInterface, error) {
	var intf NgInterface
	if len(b) < 20 {
		return intf, errInvalidf("bad interface descriptor block")
	}
	intf.LinkType = layers.LinkType(r.getUint16(b[8:10]))
	intf.SnapLength = r.getUint32(b[12:16])
	b = b[16:]

	// loop until we get to the 4-byte trailer length field
	for len(b) > 4 {
		code, body, length, err := r.readOption(b)
		if err != nil {
			return intf, err
		}
		b = b[length:]
		switch code {
		case ngOptionCodeInterfaceName:
			intf.Name = string(body)
		case ngOptionCodeComment:
			intf.Comment = string(body)
		case ngOptionCodeInterfaceDescription:
			intf.Description = string(body)
		case ngOptionCodeInterfaceFilter:
			// ignore filter type (first byte) since it is not specified
			intf.Filter = string(body[1:])
		case ngOptionCodeInterfaceOS:
			intf.OS = string(body)
		case ngOptionCodeInterfaceTimestampOffset:
			if len(body) != 8 {
				return intf, errInvalidf("bad option value: ngOptionCodeInterfaceTimestampOffset")
			}
			intf.TimestampOffset = r.getUint64(body[:8])
		case ngOptionCodeInterfaceTimestampResolution:
			if len(body) != 1 {
				return intf, errInvalidf("bad option value: ngOptionCodeInterfaceTimestampResolution")
			}
			intf.TimestampResolution = NgResolution(body[0])
		}
	}
	if intf.TimestampResolution == 0 {
		intf.TimestampResolution = 6
	}

	//parse options
	if intf.TimestampResolution.Binary() {
		//negative power of 2
		intf.secondMask = 1 << intf.TimestampResolution.Exponent()
	} else {
		//negative power of 10
		intf.secondMask = 1
		for j := uint8(0); j < intf.TimestampResolution.Exponent(); j++ {
			intf.secondMask *= 10
		}
	}
	intf.scaleDown = 1
	intf.scaleUp = 1
	if intf.secondMask < 1e9 {
		intf.scaleUp = 1e9 / intf.secondMask
	} else {
		intf.scaleDown = intf.secondMask / 1e9
	}
	r.ifaces = append(r.ifaces, intf)
	return intf, nil
}

// convertTime adds offset + shifts the given time value according to the given interface
func (r *NgReader) convertTime(ifaceID int, ts uint64) (int64, int64) {
	iface := r.ifaces[ifaceID]
	return int64(ts/iface.secondMask + iface.TimestampOffset), int64(ts % iface.secondMask * iface.scaleUp / iface.scaleDown)
}

// readPacketHeader looks for a packet (enhanced, simple, or packet) and parses the header.
// If an interface descriptor, an interface statistics block, or a section header is encountered, those are handled accordingly.
// All other block types are skipped. New block types must be added here.
func (r *NgReader) Read() ([]byte, BlockType, error) {
	block := r.first
	if block != nil {
		r.first = nil
		r.offset += uint64(len(block))
		return block, TypeSection, nil
	}
	for {
		typ, block, err := r.readBlock()
		if err != nil {
			return nil, 0, err
		}
		r.offset += uint64(len(block))
		switch typ {
		case ngBlockTypeEnhancedPacket:
			packet, _, err := r.parsePacket(block)
			if packet == nil {
				return nil, 0, err
			}
			return block, TypePacket, nil
		case ngBlockTypeSimplePacket:
			return nil, 0, errInvalidf("pcap-ng simple packets not supported")
		case ngBlockTypeInterfaceDescriptor:
			_, err := r.parseInterfaceDescriptor(block)
			return block, TypeInterface, err
		case ngBlockTypeInterfaceStatistics:
			// ignore and drop
		case ngBlockTypeSectionHeader:
			return block, TypeSection, err
		case ngBlockTypePacket:
			return nil, 0, errInvalidf("pcap-ng deprecated type packet not supported")
		}
	}
}
