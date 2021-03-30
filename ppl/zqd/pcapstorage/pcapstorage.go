package pcapstorage

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sync"

	"github.com/brimdata/zed/api"
	"github.com/brimdata/zed/pcap"
	"github.com/brimdata/zed/pcap/pcapio"
	"github.com/brimdata/zed/pkg/iosrc"
	"github.com/brimdata/zed/pkg/nano"
	"github.com/brimdata/zed/zqe"
)

const (
	MetaFile = "pcap.json"
)

type Store struct {
	metaMu sync.Mutex
	meta   meta
	root   iosrc.URI
}

type meta struct {
	PcapURI iosrc.URI
	Span    nano.Span
	Index   pcap.Index
}

type Info struct {
	PcapURI  iosrc.URI
	PcapSize int64
	Span     nano.Span
}

func New(root iosrc.URI) *Store {
	return &Store{root: root}
}

func Load(ctx context.Context, u iosrc.URI) (*Store, error) {
	metauri := u.AppendPath(MetaFile)
	b, err := iosrc.ReadFile(ctx, metauri)
	if os.IsNotExist(err) {
		return nil, zqe.E(zqe.NotFound, "%s: pcap store not found", metauri)
	}
	if err != nil {
		return nil, err
	}
	var m meta
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, err
	}
	return &Store{
		root: u,
		meta: m,
	}, nil
}

func (s *Store) Update(ctx context.Context, pcapuri iosrc.URI, warningCh chan<- string) error {
	s.metaMu.Lock()
	defer s.metaMu.Unlock()
	pcapfile, err := iosrc.NewReader(ctx, pcapuri)
	if err != nil {
		return err
	}
	defer pcapfile.Close()
	idx, err := pcap.CreateIndexWithWarnings(pcapfile, 10000, warningCh)
	if err != nil {
		return err
	}
	m := meta{
		PcapURI: pcapuri,
		Span:    idx.Span(),
		Index:   idx,
	}
	err = iosrc.Replace(ctx, s.root.AppendPath(MetaFile), func(w io.Writer) error {
		return json.NewEncoder(w).Encode(m)
	})
	if err != nil {
		return err
	}
	s.meta = m
	return nil
}

func (s *Store) Empty() bool {
	s.metaMu.Lock()
	defer s.metaMu.Unlock()
	return s.meta.PcapURI.IsZero()
}

func (s *Store) Info(ctx context.Context) (Info, error) {
	s.metaMu.Lock()
	defer s.metaMu.Unlock()
	fi, err := iosrc.Stat(ctx, s.meta.PcapURI)
	if err != nil {
		return Info{}, err
	}
	return Info{
		PcapURI:  s.meta.PcapURI,
		PcapSize: fi.Size(),
		Span:     s.meta.Span,
	}, nil
}

func (s *Store) PcapURI() iosrc.URI {
	s.metaMu.Lock()
	defer s.metaMu.Unlock()
	return s.meta.PcapURI
}

func (s *Store) Delete(ctx context.Context) error {
	s.metaMu.Lock()
	defer s.metaMu.Unlock()
	s.meta = meta{}
	return iosrc.Remove(ctx, s.root.AppendPath(MetaFile))
}

type Search struct {
	*pcap.SearchReader
	io.Closer
}

// NewSearch returns a *Search that streams all the packets meeting
// the provided search request. If pcaps are not supported in this Space,
// ErrPcapOpsNotSupported is returned.
func (s *Store) NewSearch(ctx context.Context, req api.PcapSearch) (*Search, error) {
	s.metaMu.Lock()
	defer s.metaMu.Unlock()
	var search *pcap.Search
	// We add two microseconds to the end of the span as fudge to deal with the
	// fact that zeek truncates timestamps to microseconds where pcap-ng
	// timestamps have nanosecond precision.  We need two microseconds because
	// both the base timestamp of a conn record as well as the duration time
	// can be truncated downward.
	span := nano.NewSpanTs(req.Span.Ts, req.Span.End()+2000)
	switch req.Proto {
	case "tcp":
		flow := pcap.NewFlow(req.SrcHost, int(req.SrcPort), req.DstHost, int(req.DstPort))
		search = pcap.NewTCPSearch(span, flow)
	case "udp":
		flow := pcap.NewFlow(req.SrcHost, int(req.SrcPort), req.DstHost, int(req.DstPort))
		search = pcap.NewUDPSearch(span, flow)
	case "icmp":
		search = pcap.NewICMPSearch(span, req.SrcHost, req.DstHost)
	default:
		return nil, fmt.Errorf("unsupported proto type: %s", req.Proto)
	}
	f, err := iosrc.NewReader(ctx, s.meta.PcapURI)
	if err != nil {
		return nil, err
	}
	slicer, err := pcap.NewSlicer(f, s.meta.Index, span)
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
	return &Search{r, f}, nil
}

const metaFileV0 = "packets.idx.json"

func MigrateV3(u iosrc.URI, pcapuri iosrc.URI) error {
	b, err := iosrc.ReadFile(context.Background(), u.AppendPath(metaFileV0))
	if err != nil {
		return err
	}
	var idx pcap.Index
	if err := json.Unmarshal(b, &idx); err != nil {
		return err
	}
	m := meta{
		PcapURI: pcapuri,
		Span:    idx.Span(),
		Index:   idx,
	}
	out, err := json.Marshal(m)
	if err != nil {
		return err
	}
	if err := iosrc.WriteFile(context.Background(), u.AppendPath(MetaFile), out); err != nil {
		return err
	}
	return iosrc.Remove(context.Background(), u.AppendPath(metaFileV0))
}
