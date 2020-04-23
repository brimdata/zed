package space

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/brimsec/zq/pcap"
	"github.com/brimsec/zq/pcap/pcapio"
	"github.com/brimsec/zq/pkg/fs"
	"github.com/brimsec/zq/pkg/nano"
	"github.com/brimsec/zq/zbuf"
	"github.com/brimsec/zq/zio/zngio"
	"github.com/brimsec/zq/zng/resolver"
	"github.com/brimsec/zq/zqd/api"
	"github.com/brimsec/zq/zqe"
	"go.uber.org/zap"
)

const (
	AllZngFile        = "all.zng"
	configFile        = "config.json"
	infoFile          = "info.json"
	PcapIndexFile     = "packets.idx.json"
	defaultStreamSize = 5000
)

var (
	ErrPcapOpsNotSupported = zqe.E(zqe.Invalid, "space does not support pcap operations")
	ErrSpaceExists         = zqe.E(zqe.Exists, "space exists")
	ErrSpaceNotExist       = zqe.E(zqe.NotFound, "space does not exist")
)

type Manager struct {
	rootPath string
	mapLock  sync.Mutex
	spaces   map[string]*Space
	logger   *zap.Logger
}

type Space struct {
	path string
	conf config

	index *zngio.TimeIndex

	// state about operations in progress
	opMutex       sync.Mutex
	active        int
	deletePending bool

	wg sync.WaitGroup
	// closed to signal non-delete ops should terminate
	cancelChan chan struct{}
}

func NewManager(root string, logger *zap.Logger) *Manager {
	mgr := &Manager{
		rootPath: root,
		spaces:   make(map[string]*Space),
		logger:   logger,
	}

	dirs, err := ioutil.ReadDir(root)
	if err != nil {
		return mgr
	}

	for _, dir := range dirs {
		if !dir.IsDir() {
			continue
		}

		path := filepath.Join(root, dir.Name())
		config, err := loadConfig(path)
		if err != nil {
			logger.Error("Error loading config", zap.Error(err))
			continue
		}

		space := newSpace(path, config)
		mgr.spaces[space.Name()] = space
	}

	return mgr
}

func (m *Manager) Create(name, dataPath string) (*api.SpacePostResponse, error) {
	m.mapLock.Lock()
	_, exists := m.spaces[name]
	if exists {
		m.mapLock.Unlock()
		return nil, ErrSpaceExists
	}

	defer m.mapLock.Unlock()

	if name == "" && dataPath == "" {
		return nil, zqe.E(zqe.Invalid, "must supply non-empty name or dataPath")
	}
	var path string
	if name == "" {
		var err error
		if path, err = fs.UniqueDir(m.rootPath, filepath.Base(dataPath)); err != nil {
			return nil, err
		}
		name = filepath.Base(path)
	} else {
		path = filepath.Join(m.rootPath, name)
		if err := os.Mkdir(path, 0700); err != nil {
			if os.IsExist(err) {
				return nil, ErrSpaceExists
			}
			return nil, err
		}
		dataPath = path
	}
	c := config{
		DataPath:      dataPath,
		ZngStreamSize: defaultStreamSize,
	}
	if err := c.save(path); err != nil {
		os.RemoveAll(path)
		return nil, err
	}

	m.spaces[name] = newSpace(path, c)
	return &api.SpacePostResponse{
		Name:    name,
		DataDir: dataPath,
	}, nil
}

func (m *Manager) Get(name string) (*Space, error) {
	m.mapLock.Lock()
	defer m.mapLock.Unlock()
	space, exists := m.spaces[name]
	if !exists {
		return nil, ErrSpaceNotExist
	}

	return space, nil
}

func (m *Manager) Delete(name string) error {
	m.mapLock.Lock()
	defer m.mapLock.Unlock()

	space, exists := m.spaces[name]
	if !exists {
		return ErrSpaceNotExist
	}

	err := space.delete()
	if err != nil {
		return err
	}

	delete(m.spaces, name)
	return nil
}

func (m *Manager) ListNames() []string {
	result := []string{}

	m.mapLock.Lock()
	defer m.mapLock.Unlock()
	for name := range m.spaces {
		result = append(result, name)
	}
	return result
}

func (m *Manager) List() []api.SpaceInfo {
	result := []api.SpaceInfo{}

	m.mapLock.Lock()
	defer m.mapLock.Unlock()
	for _, space := range m.spaces {
		info, err := space.Info()
		if err != nil {
			m.logger.Error("error reading space info", zap.Error(err))
			continue
		}
		result = append(result, info)
	}
	return result
}

func newSpace(path string, conf config) *Space {
	return &Space{
		path:       path,
		conf:       conf,
		index:      zngio.NewTimeIndex(),
		cancelChan: make(chan struct{}, 0),
	}
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

func (s *Space) Name() string {
	return filepath.Base(s.path)
}

func (s *Space) Info() (api.SpaceInfo, error) {
	logsize, err := s.LogSize()
	if err != nil {
		return api.SpaceInfo{}, err
	}
	packetsize, err := s.PacketSize()
	if err != nil {
		return api.SpaceInfo{}, err
	}
	spaceInfo := api.SpaceInfo{
		Name:          s.Name(),
		Size:          logsize,
		PacketSupport: s.PacketPath() != "",
		PacketPath:    s.PacketPath(),
		PacketSize:    packetsize,
	}
	i, err := loadInfoFile(s.conf.DataPath)
	if err == nil {
		spaceInfo.MinTime = &i.MinTime
		spaceInfo.MaxTime = &i.MaxTime
	} else if !errors.Is(err, os.ErrNotExist) {
		return api.SpaceInfo{}, err
	}
	return spaceInfo, nil
}

// PcapSearch returns a *pcap.SearchReader that streams all the packets meeting
// the provided search request. If pcaps are not supported in this Space,
// ErrPcapOpsNotSupported is returned.
func (s *Space) PcapSearch(ctx context.Context, req api.PacketSearch) (*SearchReadCloser, error) {
	if s.PacketPath() == "" || !s.HasFile(PcapIndexFile) {
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
	f, err := os.Open(s.PacketPath())
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

// LogSize returns the size in bytes of the logs in space.
func (s *Space) LogSize() (int64, error) {
	return sizeof(s.DataPath(AllZngFile))
}

// PacketSize returns the size in bytes of the packet capture in the space.
func (s *Space) PacketSize() (int64, error) {
	return sizeof(s.PacketPath())
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

func (s *Space) OpenZng(span nano.Span) (zbuf.ReadCloser, error) {
	zctx := resolver.NewContext()

	f, err := os.Open(s.DataPath(AllZngFile))
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, err
		}

		// Couldn't read all.zng, check for an old space with all.bzng
		bzngFile := strings.TrimSuffix(AllZngFile, filepath.Ext(AllZngFile)) + ".bzng"
		f, err = os.Open(s.DataPath(bzngFile))
		if err != nil {
			if !os.IsNotExist(err) {
				return nil, err
			}
			r := zngio.NewReader(strings.NewReader(""), zctx)
			return zbuf.NopReadCloser(r), nil
		}
	}

	return s.index.NewReader(f, zctx, span)
}

func (s *Space) CreateFile(file string) (*os.File, error) {
	return os.Create(s.DataPath(file))
}

func (s *Space) HasFile(file string) bool {
	info, err := os.Stat(s.DataPath(file))
	if err != nil {
		return false
	}
	return !info.IsDir()
}

func (s *Space) SetPacketPath(pcapPath string) error {
	s.conf.PacketPath = pcapPath
	return s.conf.save(s.path)
}

func (s *Space) PacketPath() string {
	return s.conf.PacketPath
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
	DataPath      string `json:"data_path"`
	PacketPath    string `json:"packet_path"`
	ZngStreamSize int    `json:"zng_stream_size"`
}

type info struct {
	MinTime nano.Ts `json:"min_time"`
	MaxTime nano.Ts `json:"max_time"`
}

// UnsetTimes nils out the cached time range value for the space.
// XXX For right now this simply deletes the info file as nothing else is stored
// there. When we get to brimsec/zq#541 the time range should be represented as
// as a pointer to a nano.Span.
func (s *Space) UnsetTimes() error {
	return os.Remove(s.DataPath(infoFile))
}

func (s *Space) SetTimes(minTs, maxTs nano.Ts) error {
	cur, err := loadInfoFile(s.conf.DataPath)
	if err != nil {
		if !os.IsNotExist(err) {
			return err
		}
		cur = info{nano.MaxTs, nano.MinTs}
	}
	cur.MinTime = nano.Min(cur.MinTime, minTs)
	cur.MaxTime = nano.Max(cur.MaxTime, maxTs)
	return cur.save(s.conf.DataPath)
}

func (s *Space) GetTimes() (*nano.Ts, *nano.Ts, error) {
	i, err := loadInfoFile(s.conf.DataPath)
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, nil, err
		}
		return nil, nil, nil
	}
	return &i.MinTime, &i.MaxTime, nil
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
	f, err := os.Create(tmppath)
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

func loadInfoFile(path string) (info, error) {
	var i info
	b, err := ioutil.ReadFile(filepath.Join(path, infoFile))
	if err != nil {
		return info{}, err
	}
	if err := json.Unmarshal(b, &i); err != nil {
		return i, err
	}
	return i, nil
}

func (i info) save(path string) error {
	path = filepath.Join(path, infoFile)
	tmppath := path + ".tmp"
	f, err := os.Create(tmppath)
	if err != nil {
		return err
	}
	if err := json.NewEncoder(f).Encode(i); err != nil {
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
