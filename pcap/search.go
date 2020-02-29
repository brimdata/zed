package pcap

import (
	"errors"
	"fmt"
	"io"
	"net"

	"github.com/brimsec/zq/pkg/nano"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
)

var (
	// ErrNoPacketsFound is an error indicating no packets have been found.
	ErrNoPacketsFound = errors.New("no packets found")
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

func matchIP(packet gopacket.Packet) (net.IP, net.IP) {
	network := packet.NetworkLayer()
	if ip, ok := network.(*layers.IPv4); ok {
		return ip.SrcIP, ip.DstIP
	} else if ip, ok := network.(*layers.IPv6); ok {
		return ip.SrcIP, ip.DstIP
	}
	return nil, nil
}

func genFlowFilter(flow Flow) func(Socket, Socket) bool {
	return func(s0, s1 Socket) bool {
		return s0.IP.Equal(flow.S0.IP) && s1.IP.Equal(flow.S1.IP) && s0.Port == flow.S0.Port && s1.Port == flow.S1.Port
	}
}

func genTCPFilter(flow Flow) func(gopacket.Packet) bool {
	match := genFlowFilter(flow)
	return func(packet gopacket.Packet) bool {
		srcIP, dstIP := matchIP(packet)
		if srcIP == nil {
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

func genUDPFilter(flow Flow) func(gopacket.Packet) bool {
	match := genFlowFilter(flow)
	return func(packet gopacket.Packet) bool {
		srcIP, dstIP := matchIP(packet)
		if srcIP == nil {
			return false
		}
		transport := packet.TransportLayer()
		tcp, ok := transport.(*layers.UDP)
		if !ok {
			return false
		}
		src := Socket{srcIP, int(tcp.SrcPort)}
		dst := Socket{dstIP, int(tcp.DstPort)}
		return match(src, dst) || match(dst, src)
	}
}

func genICMPFilter(src, dst net.IP) func(gopacket.Packet) bool {
	return func(packet gopacket.Packet) bool {
		if packet.LayerClass(layers.LayerClassIPControl) == nil {
			return false
		}
		srcIP, dstIP := matchIP(packet)
		if srcIP == nil {
			return false
		}
		return (src.Equal(srcIP) && dst.Equal(dstIP)) || (src.Equal(dstIP) && dst.Equal(srcIP))
	}
}

// XXX currently assumes legacy pcap is produced by the input reader
// XXX need to handle searching over multiple pcap files
func (s *Search) Run(w io.Writer, r io.Reader) error {
	pcap, err := NewReader(r)
	if err != nil {
		return err
	}
	hdr, err := pcap.ReadBlock(s.span)
	if err != nil {
		return err
	}
	if len(hdr) != fileHeaderLen {
		return errors.New("bad pcap file")
	}
	//XXX the .LayerType() method is returning Unknown for some reason
	//outerLayer := pcap.LinkType().LayerType()
	outerLayer := layers.LayerTypeEthernet
	opts := gopacket.DecodeOptions{Lazy: true, NoCopy: true}
	for {
		block, err := pcap.ReadBlock(s.span)
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
		if block == nil {
			break
		}
		pktBuf := block[packetHeaderLen:]
		packet := gopacket.NewPacket(pktBuf, outerLayer, opts)
		if s.filter != nil && !s.filter(packet) {
			continue
		}
		if hdr != nil {
			w.Write(hdr)
			hdr = nil
		}
		if _, err = w.Write(block); err != nil {
			return err
		}
	}
	if hdr != nil {
		return ErrNoPacketsFound
	}
	return nil
}
