package pcapstorage

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sync"

	"github.com/brimsec/zq/pcap"
	"github.com/brimsec/zq/pcap/pcapio"
	"github.com/brimsec/zq/pkg/iosrc"
	"github.com/brimsec/zq/pkg/nano"
	"github.com/brimsec/zq/zqd/api"
	"github.com/brimsec/zq/zqe"
)

const (
	MetaFile = "pcap.meta.json"
)

type Store struct {
	meta meta
	root iosrc.URI
	mu   sync.Mutex
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

func Load(u iosrc.URI) (*Store, error) {
	metauri := u.AppendPath(MetaFile)
	b, err := iosrc.ReadFile(metauri)
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

func (s *Store) Rewrite(pcapuri iosrc.URI) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	pcapfile, err := iosrc.NewReader(pcapuri)
	if err != nil {
		return err
	}
	defer pcapfile.Close()
	idx, err := pcap.CreateIndex(pcapfile, 10000)
	if err != nil {
		return err
	}
	m := meta{
		PcapURI: pcapuri,
		Span:    idx.Span(),
		Index:   idx,
	}
	b, err := json.Marshal(m)
	if err != nil {
		return err
	}
	src, err := iosrc.GetSource(s.root)
	if err != nil {
		return err
	}
	metauri := s.root.AppendPath(MetaFile)
	var w io.WriteCloser
	if replace, ok := src.(iosrc.ReplacerAble); ok {
		w, err = replace.NewReplacer(metauri)
	} else {
		w, err = src.NewWriter(metauri)
	}
	if err != nil {
		return err
	}
	if _, err := w.Write(b); err != nil {
		return err
	}
	if err := w.Close(); err != nil {
		return err
	}
	s.meta = m
	return nil
}

func (s *Store) Empty() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.meta.PcapURI.IsZero()
}

func (s *Store) Info() (Info, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	fi, err := iosrc.Stat(s.meta.PcapURI)
	if err != nil {
		return Info{}, err
	}
	return Info{
		PcapURI:  s.meta.PcapURI,
		PcapSize: fi.Size(),
		Span:     s.meta.Span,
	}, nil
}

func (s *Store) Delete() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.meta = meta{}
	return iosrc.Remove(s.root.AppendPath(MetaFile))
}

type Search struct {
	*pcap.SearchReader
	f io.ReadCloser
}

// NewSearch returns a *Search that streams all the packets meeting
// the provided search request. If pcaps are not supported in this Space,
// ErrPcapOpsNotSupported is returned.
func (s *Store) NewSearch(ctx context.Context, req api.PcapSearch) (*Search, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
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
	f, err := iosrc.NewReader(s.meta.PcapURI)
	if err != nil {
		return nil, err
	}
	slicer, err := pcap.NewSlicer(f, s.meta.Index, req.Span)
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

func (s *Search) Close() error {
	return s.f.Close()
}

const metaFileV0 = "packets.idx.json"

func MigrateV3(u iosrc.URI, pcapuri iosrc.URI) error {
	b, err := iosrc.ReadFile(u.AppendPath(metaFileV0))
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
	if err := iosrc.WriteFile(u.AppendPath(MetaFile), out); err != nil {
		return err
	}
	return iosrc.Remove(u.AppendPath(metaFileV0))
}
