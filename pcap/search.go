package pcap

import (
	"context"
	"fmt"
	"io"
	"net"

	zqe "github.com/brimsec/zq/errors"
	"github.com/brimsec/zq/pcap/pcapio"
	"github.com/brimsec/zq/pkg/ctxio"
	"github.com/brimsec/zq/pkg/nano"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
)

var (
	// ErrNoPacketsFound is an error indicating no packets have been found.
	ErrNoPacketsFound = zqe.E(zqe.NotFound, "no packets found")
)

type PacketFilter func(gopacket.Packet) bool

// Search describes the parameters for a packet search over a pcap file.
type Search struct {
	span   nano.Span
	filter PacketFilter
	id     string
}

func NewTCPSearch(span nano.Span, flow Flow) *Search {
	id := fmt.Sprintf("%s_tcp_%s", span.Ts.StringFloat(), flow)
	return &Search{
		span:   span,
		filter: genTCPFilter(flow),
		id:     id,
	}
}

func NewUDPSearch(span nano.Span, flow Flow) *Search {
	id := fmt.Sprintf("%s_udp_%s", span.Ts.StringFloat(), flow)
	return &Search{
		span:   span,
		filter: genUDPFilter(flow),
		id:     id,
	}
}

func NewICMPSearch(span nano.Span, src, dst net.IP) *Search {
	id := fmt.Sprintf("icmp_%s_%s_%s", span.Ts.StringFloat(), src.String(), dst.String())
	return &Search{
		span:   span,
		filter: genICMPFilter(src, dst),
		id:     id,
	}
}

func NewRangeSearch(span nano.Span) *Search {
	id := fmt.Sprintf("%s_%s_%s", span.Ts.StringFloat(), "none", "no-filter")
	return &Search{
		span: span,
		id:   id,
	}
}

func (s Search) Span() nano.Span {
	return s.span
}

// ID returns an identifier for the search performed.
func (s Search) ID() string {
	return s.id
}

func matchIP(packet gopacket.Packet) (net.IP, net.IP, bool) {
	network := packet.NetworkLayer()
	if ip, ok := network.(*layers.IPv4); ok {
		return ip.SrcIP, ip.DstIP, true
	} else if ip, ok := network.(*layers.IPv6); ok {
		return ip.SrcIP, ip.DstIP, true
	}
	return nil, nil, false
}

func genFlowFilter(flow Flow) func(Socket, Socket) bool {
	return func(s0, s1 Socket) bool {
		return s0.IP.Equal(flow.S0.IP) && s1.IP.Equal(flow.S1.IP) && s0.Port == flow.S0.Port && s1.Port == flow.S1.Port
	}
}

func genTCPFilter(flow Flow) PacketFilter {
	match := genFlowFilter(flow)
	return func(packet gopacket.Packet) bool {
		srcIP, dstIP, ok := matchIP(packet)
		if !ok {
			return false
		}
		transport := packet.TransportLayer()
		tcp, ok := transport.(*layers.TCP)
		if !ok {
			return false
		}
		src := Socket{srcIP, int(tcp.SrcPort)}
		dst := Socket{dstIP, int(tcp.DstPort)}
		return match(src, dst) || match(dst, src)
	}
}

func genUDPFilter(flow Flow) PacketFilter {
	match := genFlowFilter(flow)
	return func(packet gopacket.Packet) bool {
		srcIP, dstIP, ok := matchIP(packet)
		if !ok {
			return false
		}
		transport := packet.TransportLayer()
		udp, ok := transport.(*layers.UDP)
		if !ok {
			return false
		}
		src := Socket{srcIP, int(udp.SrcPort)}
		dst := Socket{dstIP, int(udp.DstPort)}
		return match(src, dst) || match(dst, src)
	}
}

func genICMPFilter(src, dst net.IP) PacketFilter {
	return func(packet gopacket.Packet) bool {
		if packet.LayerClass(layers.LayerClassIPControl) == nil {
			return false
		}
		srcIP, dstIP, ok := matchIP(packet)
		if !ok {
			return false
		}
		return (src.Equal(srcIP) && dst.Equal(dstIP)) || (src.Equal(dstIP) && dst.Equal(srcIP))
	}
}

// XXX need to handle searching over multiple pcap files
func (s *Search) Run(ctx context.Context, w io.Writer, r pcapio.Reader) error {
	_, err := ctxio.Copy(ctx, w, s.Reader(r))
	return err
}

type SearchReader struct {
	*Search
	reader pcapio.Reader
	opts   gopacket.DecodeOptions
	npkt   int
	buf    []byte
}

func (s *Search) Reader(r pcapio.Reader) *SearchReader {
	opts := gopacket.DecodeOptions{Lazy: true, NoCopy: true}
	return &SearchReader{Search: s, reader: r, opts: opts}
}

func (s *SearchReader) Read(p []byte) (n int, err error) {
	if len(s.buf) == 0 {
		s.buf, err = s.next()
		if err != nil {
			return 0, err
		}
		if len(s.buf) == 0 {
			return 0, io.EOF
		}
	}
	n = copy(p, s.buf)
	s.buf = s.buf[n:]
	return n, err
}

func (s *SearchReader) next() ([]byte, error) {
	for {
		block, typ, err := s.reader.Read()
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
		if block == nil {
			break
		}
		// Write any blocks that aren't packets.  For pcap-ng, this has
		// the effect of writing out all the sections in the underlying
		// pcap, with empty sections where there a no matches.  This is a
		// bit ugly but perfectly valid output.  We could clean this up
		// by looking for sections headers, a buffering unnwritten sections
		// until we get to the first packet and never writing the blocksa
		// for sections that have no packets.
		if typ != pcapio.TypePacket {
			return block, nil
		}
		pktBuf, ts, linkType := s.reader.Packet(block)
		if pktBuf == nil {
			return nil, pcapio.ErrCorruptPcap
		}
		if !s.span.ContainsClosed(ts) {
			continue
		}
		packet := gopacket.NewPacket(pktBuf, linkType, s.opts)
		if s.filter != nil && !s.filter(packet) {
			continue
		}
		s.npkt++
		return block, nil
	}
	if s.npkt == 0 {
		return nil, ErrNoPacketsFound
	}
	return nil, nil
}
