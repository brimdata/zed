package space

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"

	"github.com/brimsec/zq/pcap"
	"github.com/brimsec/zq/pcap/pcapio"
	"github.com/brimsec/zq/pkg/fs"
	"github.com/brimsec/zq/pkg/nano"
	"github.com/brimsec/zq/zqd/api"
	"github.com/brimsec/zq/zqd/storage"
	"github.com/brimsec/zq/zqe"
	"github.com/segmentio/ksuid"
)

const (
	configFile        = "config.json"
	PcapIndexFile     = "packets.idx.json"
	defaultStreamSize = 5000
)

var (
	ErrPcapOpsNotSupported = zqe.E(zqe.Invalid, "space does not support pcap operations")
	ErrSpaceExists         = zqe.E(zqe.Exists, "space exists")
	ErrSpaceNotExist       = zqe.E(zqe.NotFound, "space does not exist")
)

func newSpaceID() api.SpaceID {
	id := ksuid.New()
	return api.SpaceID(fmt.Sprintf("sp_%s", id.String()))
}

type Space struct {
	Storage *storage.ZngStorage

	path string
	conf config

	// state about operations in progress
	opMutex       sync.Mutex
	active        int
	deletePending bool

	wg sync.WaitGroup
	// closed to signal non-delete ops should terminate
	cancelChan chan struct{}
}

func newSpace(path string, conf config) (*Space, error) {
	s, err := storage.OpenZng(path, conf.ZngStreamSize)
	if err != nil {
		return nil, err
	}
	return &Space{
		Storage:    s,
		path:       path,
		conf:       conf,
		cancelChan: make(chan struct{}, 0),
	}, nil
}

// StartSpaceOp registers that an operation on this space is in progress.
// If the space is pending deletion, an error is returned.
// Otherwise, this returns a new context, and a done function that must
// be called when the operation completes.
func (s *Space) StartSpaceOp(ctx context.Context) (context.Context, context.CancelFunc, error) {
	s.opMutex.Lock()
	defer s.opMutex.Unlock()

	if s.deletePending {
		return ctx, func() {}, zqe.E(zqe.Conflict, "space is pending deletion")
	}

	s.wg.Add(1)

	ctx, cancel := context.WithCancel(ctx)
	go func() {
		select {
		case <-ctx.Done():
		case <-s.cancelChan:
			cancel()
		}
	}()

	done := func() {
		s.opMutex.Lock()
		defer s.opMutex.Unlock()

		s.wg.Done()
		cancel()
	}

	return ctx, done, nil
}

func (s *Space) ID() api.SpaceID {
	return api.SpaceID(filepath.Base(s.path))
}

func (s *Space) Update(req api.SpacePutRequest) error {
	if req.Name == "" {
		return zqe.E(zqe.Invalid, "cannot set name to an empty string")
	}
	// XXX This is not thread safe. Will fix in upcoming pr.
	s.conf.Name = req.Name
	return s.conf.save(s.path)
}

func (s *Space) Info() (api.SpaceInfo, error) {
	logsize, err := s.Storage.Size()
	if err != nil {
		return api.SpaceInfo{}, err
	}
	pcapsize, err := s.PcapSize()
	if err != nil {
		return api.SpaceInfo{}, err
	}
	var span *nano.Span
	sp := s.Storage.Span()
	if sp.Dur > 0 {
		span = &sp
	}
	spaceInfo := api.SpaceInfo{
		ID:          s.ID(),
		Name:        s.conf.Name,
		DataPath:    s.conf.DataPath,
		Size:        logsize,
		Span:        span,
		PcapSupport: s.PcapPath() != "",
		PcapPath:    s.PcapPath(),
		PcapSize:    pcapsize,
	}
	return spaceInfo, nil
}

// PcapSearch returns a *pcap.SearchReader that streams all the packets meeting
// the provided search request. If pcaps are not supported in this Space,
// ErrPcapOpsNotSupported is returned.
func (s *Space) PcapSearch(ctx context.Context, req api.PcapSearch) (*SearchReadCloser, error) {
	if s.PcapPath() == "" || !s.hasFile(PcapIndexFile) {
		return nil, ErrPcapOpsNotSupported
	}
	index, err := pcap.LoadIndex(s.DataPath(PcapIndexFile))
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
	f, err := fs.Open(s.PcapPath())
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
	return &SearchReadCloser{r, f}, nil

}

type SearchReadCloser struct {
	*pcap.SearchReader
	f *os.File
}

func (c *SearchReadCloser) Close() error {
	return c.f.Close()
}

// PcapSize returns the size in bytes of the packet capture in the space.
func (s *Space) PcapSize() (int64, error) {
	return sizeof(s.PcapPath())
}

func sizeof(path string) (int64, error) {
	f, err := os.Stat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return 0, nil
		}
		return 0, err
	}
	return f.Size(), nil
}

func (s *Space) DataPath(elem ...string) string {
	return filepath.Join(append([]string{s.conf.DataPath}, elem...)...)
}

func (s *Space) hasFile(file string) bool {
	info, err := os.Stat(s.DataPath(file))
	if err != nil {
		return false
	}
	return !info.IsDir()
}

func (s *Space) SetPcapPath(pcapPath string) error {
	s.conf.PcapPath = pcapPath
	return s.conf.save(s.path)
}

func (s *Space) PcapPath() string {
	return s.conf.PcapPath
}

func (s *Space) StreamSize() int {
	return s.conf.ZngStreamSize
}

// Delete removes the space's path and data dir (should the data dir be
// different then the space's path).
// Don't call this directly, used Manager.Delete()
func (s *Space) delete() error {
	s.opMutex.Lock()

	if s.deletePending {
		s.opMutex.Unlock()
		return zqe.E(zqe.Conflict, "space is pending deletion")
	}

	s.deletePending = true
	s.opMutex.Unlock()

	close(s.cancelChan)
	s.wg.Wait()

	if err := os.RemoveAll(s.path); err != nil {
		return err
	}
	return os.RemoveAll(s.conf.DataPath)
}

type config struct {
	Name     string `json:"name"`
	DataPath string `json:"data_path"`
	// XXX PcapPath should be named pcap_path in json land. To avoid having to
	// do a migration we'll keep this as-is for now.
	PcapPath      string `json:"packet_path"`
	ZngStreamSize int    `json:"zng_stream_size"`
}

// loadConfig loads the contents of config.json in a space's path.
func loadConfig(spacePath string) (config, error) {
	var c config
	b, err := ioutil.ReadFile(filepath.Join(spacePath, configFile))
	if err != nil {
		return c, err
	}
	if err := json.Unmarshal(b, &c); err != nil {
		return c, err
	}
	return c, nil
}

func (c config) save(spacePath string) error {
	path := filepath.Join(spacePath, configFile)
	tmppath := path + ".tmp"
	f, err := fs.Create(tmppath)
	if err != nil {
		return err
	}
	if err := json.NewEncoder(f).Encode(c); err != nil {
		f.Close()
		os.Remove(tmppath)
		return err
	}
	if err = f.Close(); err != nil {
		os.Remove(tmppath)
		return err
	}
	return os.Rename(tmppath, path)
}
