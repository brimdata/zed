package search

import (
	"context"
	"fmt"
	"os"

	"github.com/brimsec/zq/pcap"
	"github.com/brimsec/zq/pcap/pcapio"
	"github.com/brimsec/zq/pkg/fs"
	"github.com/brimsec/zq/zqd/api"
)

type PcapSpace interface {
	PcapIndexPath() string
	PcapPath() string
}

// NewPcapSearch returns a *pcap.SearchReader that streams all the packets meeting
// the provided search request. If pcaps are not supported in this Space,
// ErrPcapOpsNotSupported is returned.
func NewPcapSearch(ctx context.Context, pspace PcapSpace, req api.PcapSearch) (*PcapSearchReadCloser, error) {
	index, err := pcap.LoadIndex(pspace.PcapIndexPath())
	if err != nil {
		return nil, err
	}
	var search *pcap.Search
	switch req.Proto {
	case "tcp":
		flow := pcap.NewFlow(req.SrcHost, int(req.SrcPort), req.DstHost, int(req.DstPort))
		search = pcap.NewTCPSearch(req.Span, flow)
	case "udp":
		flow := pcap.NewFlow(req.SrcHost, int(req.SrcPort), req.DstHost, int(req.DstPort))
		search = pcap.NewUDPSearch(req.Span, flow)
	case "icmp":
		search = pcap.NewICMPSearch(req.Span, req.SrcHost, req.DstHost)
	default:
		return nil, fmt.Errorf("unsupported proto type: %s", req.Proto)
	}
	f, err := fs.Open(pspace.PcapPath())
	if err != nil {
		return nil, err
	}
	slicer, err := pcap.NewSlicer(f, index, req.Span)
	if err != nil {
		f.Close()
		return nil, err
	}
	pcapReader, err := pcapio.NewReader(slicer)
	if err != nil {
		f.Close()
		return nil, err
	}
	r, err := search.Reader(ctx, pcapReader)
	if err != nil {
		f.Close()
		return nil, err
	}
	return &PcapSearchReadCloser{r, f}, nil
}

type PcapSearchReadCloser struct {
	*pcap.SearchReader
	f *os.File
}

func (c *PcapSearchReadCloser) Close() error {
	return c.f.Close()
}
