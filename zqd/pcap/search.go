package pcap

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"os"

	"github.com/brimsec/zq/pkg/nano"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcapgo"
)

var (
	// ErrNoPacketsFound is an error indicating no packets have been found.
	ErrNoPacketsFound = errors.New("no packets found for specified connection")
)

// Connection describes the parameters for a packet search on a store.
type Connection struct {
	span          nano.Span
	proto         string
	networkflow   gopacket.Flow
	transportflow gopacket.Flow
}

// NewConnection creates a new connection.
func NewConnection(span nano.Span, proto string, srcHost string, srcPort *uint16, dstHost string, dstPort *uint16) (*Connection, error) {
	switch proto {
	case "icmp", "tcp", "udp":
	default:
		return nil, fmt.Errorf("unsupported proto type: %s", proto)
	}

	c := &Connection{span: span, proto: proto}

	// convert ips
	srcip := net.ParseIP(srcHost)
	if srcip == nil {
		return nil, fmt.Errorf("invalid ip: %s", srcHost)
	}
	dstip := net.ParseIP(dstHost)
	if dstip == nil {
		return nil, fmt.Errorf("invalid ip: %s", dstHost)
	}
	endpoint := layers.NewIPEndpoint(srcip)
	if endpoint.EndpointType() == layers.EndpointIPv4 {
		srcip = srcip.To4()
		dstip = dstip.To4()
	}
	c.networkflow = gopacket.NewFlow(endpoint.EndpointType(), srcip, dstip)

	if c.proto == "icmp" {
		return c, nil
	}

	if srcPort == nil || dstPort == nil {
		return nil, fmt.Errorf("%s connections must have src port and dst port", c.proto)
	}

	// convert ports
	src := convertPort(*srcPort)
	dst := convertPort(*dstPort)
	if c.proto == "tcp" {
		c.transportflow = gopacket.NewFlow(layers.EndpointTCPPort, src, dst)
	} else {
		c.transportflow = gopacket.NewFlow(layers.EndpointUDPPort, src, dst)
	}

	return c, nil
}

func convertPort(port uint16) []byte {
	b := make([]byte, 2)
	binary.BigEndian.PutUint16(b, port)
	return b
}

// ID returns an identifier for the search performed.
func (cs Connection) ID() string {
	return fmt.Sprintf(
		"%s_%s_%v",
		cs.span.Ts.StringFloat(),
		cs.proto,
		cs.networkflow,
	)
}

// Search xxx
func Search(store *Store, conn *Connection) (io.Reader, error) {

	// get packets
	buf := &bytes.Buffer{}
	w := pcapgo.NewWriter(buf)
	var pcount int
	if err := w.WriteFileHeader(65536, layers.LinkTypeEthernet); err != nil {
		return nil, err
	}

	for _, fname := range store.Range(conn.span) {
		f, err := os.Open(fname)
		if err != nil {
			return nil, err
		}

		handle, err := pcapgo.NewReader(f)
		if err != nil {
			return nil, err
		}
		src := gopacket.NewPacketSource(handle, handle.LinkType())
		for packet := range src.Packets() {
			if packet.Metadata().CaptureInfo.Timestamp.Before(conn.span.Ts.Time()) ||
				packet.Metadata().CaptureInfo.Timestamp.After(conn.span.End().Time()) {
				continue
			}
			if match(conn, packet) {
				if err := w.WritePacket(packet.Metadata().CaptureInfo, packet.Data()); err != nil {
					return nil, err
				}
				pcount++
			}
		}
	}

	if pcount == 0 {
		return nil, ErrNoPacketsFound
	}

	return buf, nil
}

func match(conn *Connection, packet gopacket.Packet) bool {
	if t := packet.NetworkLayer(); t != nil {
		// fmt.Println(t.NetworkFlow().FastHash(), t.)
		if t.NetworkFlow().FastHash() == conn.networkflow.FastHash() {
			switch conn.proto {
			case "icmp":
				return checkICMP(packet)
			case "tcp", "udp":
				return checkTransport(conn, packet)
			default:
				return false
			}

		}
	}
	return false
}

func checkICMP(packet gopacket.Packet) bool {
	return packet.LayerClass(layers.LayerClassIPControl) != nil
}

func checkTransport(c *Connection, packet gopacket.Packet) bool {
	trans := packet.TransportLayer()
	return trans != nil && trans.TransportFlow().FastHash() == c.transportflow.FastHash()
}
