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

//XXX bool true = TCP, else UDP, change this
type Predicate func(bool, Socket, Socket) bool

// Search describes the parameters for a packet search over a pcap file.
type Search struct {
	span    nano.Span
	matcher Predicate
	id      string
}

// NewSearch creates a new search object.
func NewSearch(span nano.Span, proto string, srcHost string, srcPort *uint16, dstHost string, dstPort *uint16) (*Search, error) {
	switch proto {
	//case "icmp", "tcp", "udp":
	case "tcp", "udp":
	default:
		return nil, fmt.Errorf("unsupported proto type: %s", proto)
	}
	// convert ips
	src := net.ParseIP(srcHost)
	if src == nil {
		return nil, fmt.Errorf("invalid ip: %s", srcHost)
	}
	dst := net.ParseIP(dstHost)
	if dst == nil {
		return nil, fmt.Errorf("invalid ip: %s", dstHost)
	}
	if srcPort == nil || dstPort == nil {
		return nil, fmt.Errorf("%s connections must have src port and dst port", proto)
	}
	return NewFlowSearch(span, proto, Flow{Socket{src, int(*srcPort)}, Socket{dst, int(*dstPort)}}), nil
}

func NewFlowSearch(span nano.Span, proto string, flow Flow) *Search {
	var tcp bool
	if proto == "tcp" {
		tcp = true
	}
	id := fmt.Sprintf("%s_%s_%s", span.Ts.StringFloat(), proto, flow)
	return &Search{
		span:    span,
		matcher: makeMatcher(tcp, flow),
		id:      id,
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

func makeMatcher(tcp_ bool, flow Flow) Predicate {
	match := func(s0, s1 Socket) bool {
		return s0.IP.Equal(flow.S0.IP) && s1.IP.Equal(flow.S1.IP) && s0.Port == flow.S0.Port && s1.Port == flow.S1.Port
	}
	return func(tcp bool, src, dst Socket) bool {
		if tcp != tcp_ {
			return false
		}
		return match(src, dst) || match(dst, src)
	}
}

// ID returns an identifier for the search performed.
func (s Search) ID() string {
	return s.id
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
	w.Write(hdr)
	var n int
	//XXX the .LayerType() method is returning Unknown for some reason
	//outerLayer := pcap.LinkType().LayerType()
	outerLayer := layers.LayerTypeEthernet
	opts := gopacket.DecodeOptions{Lazy: true, NoCopy: true}
	for {
		block, err := pcap.ReadBlock(s.span)
		if err != nil {
			if err == io.EOF {
				err = nil
			}
			return err
		}
		if block == nil {
			break
		}
		if s.matcher == nil {
			n++
			if _, err = w.Write(block); err != nil {
				return err
			}
			continue
		}

		//XXX need to support other protocols like ICMP, etc

		pktBuf := block[packetHeaderLen:]
		packet := gopacket.NewPacket(pktBuf, outerLayer, opts)
		network := packet.NetworkLayer()
		var src, dst net.IP
		if ip, ok := network.(*layers.IPv4); ok {
			src = ip.SrcIP
			dst = ip.DstIP
		} else if ip, ok := network.(*layers.IPv6); ok {
			src = ip.SrcIP
			dst = ip.DstIP
		} else {
			continue
		}
		isTcp := true
		var srcPort, dstPort int
		transport := packet.TransportLayer()
		if tcp, ok := transport.(*layers.TCP); ok {
			srcPort = int(tcp.SrcPort)
			dstPort = int(tcp.DstPort)
		} else if udp, ok := transport.(*layers.UDP); ok {
			isTcp = false
			srcPort = int(udp.SrcPort)
			dstPort = int(udp.DstPort)
		} else {
			continue
		}
		if s.matcher(isTcp, Socket{src, srcPort}, Socket{dst, dstPort}) {
			n++
			if _, err = w.Write(block); err != nil {
				return err
			}
		}
	}
	if n == 0 {
		return ErrNoPacketsFound
	}
	return nil
}
