package pcap

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"sync"

	"github.com/brimsec/zq/pkg/nano"
	"github.com/brimsec/zq/zqd/api"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcapgo"
)

var (
	// ErrNoPacketsFound is an error indicating no packets have been found.
	ErrNoPacketsFound = errors.New("no packets found for specified connection")
)

// Search executes a search for the packets matching the provided search
// parameters in the provided search. If the packets are found a reader stream
// of the emitting the packets in pcap form are returned only with a string
// representing the connection. If no packets are found ErrNoPacketsFound is
// returned as an error.
func Search(spacePath string, s api.PacketSearch) (io.Reader, string, error) {
	store, err := getPcapStore(spacePath)
	if err != nil {
		return nil, "", err
	}
	conn, err := newConnection(s)
	if err != nil {
		return nil, "", err
	}
	buf := &bytes.Buffer{}
	w := pcapgo.NewWriter(buf)
	var pcount int
	if err := w.WriteFileHeader(65536, layers.LinkTypeEthernet); err != nil {
		return nil, "", err
	}

	for _, fname := range store.Range(conn.span) {
		f, err := os.Open(fname)
		if err != nil {
			return nil, "", err
		}
		handle, err := pcapgo.NewReader(f)
		if err != nil {
			return nil, "", err
		}
		src := gopacket.NewPacketSource(handle, handle.LinkType())
		for packet := range src.Packets() {
			if packet.Metadata().CaptureInfo.Timestamp.Before(conn.span.Ts.Time()) ||
				packet.Metadata().CaptureInfo.Timestamp.After(conn.span.End().Time()) {
				continue
			}
			if match(conn, packet) {
				if err := w.WritePacket(packet.Metadata().CaptureInfo, packet.Data()); err != nil {
					return nil, "", err
				}
				pcount++
			}
		}
	}
	if pcount == 0 {
		return nil, "", ErrNoPacketsFound
	}
	return buf, conn.id(), nil
}

// connection describes the parameters for a packet search on a store.
type connection struct {
	span          nano.Span
	proto         string
	networkflow   gopacket.Flow
	transportflow gopacket.Flow
}

// newConnection creates a new connection.
func newConnection(s api.PacketSearch) (*connection, error) {
	switch s.Proto {
	case "icmp":
	case "tcp", "udp":
		if s.SrcPort == nil || s.DstPort == nil {
			return nil, fmt.Errorf("%s connections must have src port and dst port", s.Proto)
		}
	default:
		return nil, fmt.Errorf("unsupported proto type: %s", s.Proto)
	}

	c := &connection{span: s.Span, proto: s.Proto}

	// convert ips
	srcip := net.ParseIP(s.SrcHost)
	if srcip == nil {
		return nil, fmt.Errorf("invalid ip: %s", s.SrcHost)
	}
	dstip := net.ParseIP(s.DstHost)
	if dstip == nil {
		return nil, fmt.Errorf("invalid ip: %s", s.DstHost)
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
	// convert ports
	src := convertPort(*s.SrcPort)
	dst := convertPort(*s.DstPort)
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
func (cs connection) id() string {
	return fmt.Sprintf(
		"%s_%s_%v",
		cs.span.Ts.StringFloat(),
		cs.proto,
		cs.networkflow,
	)
}

func HasPcaps(spacePath string) bool {
	dirPath := filepath.Join(spacePath, "packets")
	return isDir(dirPath)
}

var pmap sync.Map

func isDir(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}

func getPcapStore(spaceName string) (*Store, error) {
	dirPath := filepath.Join(spaceName, "packets")
	if !isDir(dirPath) {
		return nil, fmt.Errorf("%s: space has no pcaps", spaceName)
	}
	if s, ok := pmap.Load(spaceName); ok {
		return s.(*Store), nil
	}
	s, err := NewStore(dirPath)
	if err != nil {
		return nil, err
	}
	pmap.Store(spaceName, s)
	return s, nil
}

func match(conn *connection, packet gopacket.Packet) bool {
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

func checkTransport(c *connection, packet gopacket.Packet) bool {
	trans := packet.TransportLayer()
	return trans != nil && trans.TransportFlow().FastHash() == c.transportflow.FastHash()
}
