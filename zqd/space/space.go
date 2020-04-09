package space

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/brimsec/zq/pcap"
	"github.com/brimsec/zq/pcap/pcapio"
	"github.com/brimsec/zq/pkg/fs"
	"github.com/brimsec/zq/pkg/nano"
	"github.com/brimsec/zq/zqd/api"
)

const (
	PcapIndexFile = "packets.idx.json"
	configFile    = "config.json"
	infoFile      = "info.json"
	AllBzngFile   = "all.bzng"
)

var (
	ErrSpaceNotExist       = errors.New("space does not exist")
	ErrSpaceExists         = errors.New("space exists")
	ErrPcapOpsNotSupported = errors.New("space does not support pcap operations")
)

type Space struct {
	path string
	conf config
}

func Open(root, name string) (*Space, error) {
	path := filepath.Join(root, name)
	c, err := loadConfig(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrSpaceNotExist
		}
		return nil, err
	}
	return &Space{path, c}, nil
}

func Create(root, name, dataPath string) (*Space, error) {
	// XXX this should be validated before reaching here.
	if name == "" && dataPath == "" {
		return nil, errors.New("must supply non-empty name or dataPath")
	}
	var path string
	if name == "" {
		var err error
		if path, err = fs.UniqueDir(root, filepath.Base(dataPath)); err != nil {
			return nil, err
		}
	} else {
		path = filepath.Join(root, name)
		if err := os.Mkdir(path, 0700); err != nil {
			if os.IsExist(err) {
				return nil, ErrSpaceExists
			}
			return nil, err
		}
	}
	if dataPath == "" {
		dataPath = path
	}
	c := config{DataPath: dataPath}
	if err := c.save(path); err != nil {
		os.RemoveAll(path)
		return nil, err
	}
	return &Space{path, c}, nil
}

func (s Space) Name() string {
	return filepath.Base(s.path)
}

func (s Space) Info() (api.SpaceInfo, error) {
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
// the provided search request. If pcaps are not supported in this Space
// ErrPcapOpsNotSupported is returned.
func (s Space) PcapSearch(req api.PacketSearch) (*pcap.SearchReader, io.Closer, error) {
	if s.PacketPath() == "" || !s.HasFile(PcapIndexFile) {
		return nil, nil, ErrPcapOpsNotSupported
	}
	index, err := pcap.LoadIndex(s.DataPath(PcapIndexFile))
	if err != nil {
		return nil, nil, err
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
		return nil, nil, fmt.Errorf("unsupported proto type: %s", req.Proto)
	}
	f, err := os.Open(s.PacketPath())
	if err != nil {
		return nil, nil, err
	}
	slicer, err := pcap.NewSlicer(f, index, req.Span)
	if err != nil {
		f.Close()
		return nil, nil, err
	}
	pcapReader, err := pcapio.NewReader(slicer)
	if err != nil {
		f.Close()
		return nil, nil, err
	}
	return search.Reader(pcapReader), f, nil
}

// LogSize returns the size in bytes of the logs in space.
func (s Space) LogSize() (int64, error) {
	return sizeof(s.DataPath(AllBzngFile))
}

// PacketSize returns the size in bytes of the packet capture in the space.
func (s Space) PacketSize() (int64, error) {
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

func (s Space) DataPath(elem ...string) string {
	return filepath.Join(append([]string{s.conf.DataPath}, elem...)...)
}

func (s Space) OpenFile(file string) (*os.File, error) {
	return os.Open(s.DataPath(file))
}

func (s Space) CreateFile(file string) (*os.File, error) {
	return os.Create(s.DataPath(file))
}

func (s Space) HasFile(file string) bool {
	info, err := os.Stat(s.DataPath(file))
	if err != nil {
		return false
	}
	return !info.IsDir()
}

func (s Space) ConfigPath() string {
	return filepath.Join(s.path, configFile)
}

func (s *Space) SetPacketPath(pcapPath string) error {
	s.conf.PacketPath = pcapPath
	return s.conf.save(s.path)
}

func (s Space) PacketPath() string {
	return s.conf.PacketPath
}

// Delete removes the space's path and data dir (should the data dir be
// different then the space's path).
func (s Space) Delete() error {
	if err := os.RemoveAll(s.path); err != nil {
		return err
	}
	return os.RemoveAll(s.conf.DataPath)
}

type config struct {
	DataPath   string `json:"data_path"`
	PacketPath string `json:"packet_path"`
}

type info struct {
	MinTime nano.Ts `json:"min_time"`
	MaxTime nano.Ts `json:"max_time"`
}

// UnsetTimes nils out the cached time range value for the space.
// XXX For right now this simply deletes the info file as nothing else is stored
// there. When we get to brimsec/zq#541 the time range should be represented as
// as a pointer to a nano.Span.
func (s Space) UnsetTimes() error {
	return os.Remove(s.DataPath(infoFile))
}

func (s Space) SetTimes(minTs, maxTs nano.Ts) error {
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

func (s Space) GetTimes() (*nano.Ts, *nano.Ts, error) {
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
